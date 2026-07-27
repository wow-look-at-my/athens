// Package sumlocal implements a local Go checksum database server.
// It computes and serves module hashes so that Go clients can verify
// module integrity without contacting sum.golang.org.
package sumlocal

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/mod/module"
	"golang.org/x/mod/sumdb"
	"golang.org/x/mod/sumdb/dirhash"
	"golang.org/x/mod/sumdb/note"
	"golang.org/x/mod/sumdb/tlog"

	"github.com/gomods/athens/pkg/storage"
)

// Fetcher downloads a module from upstream and saves it into the storage
// backend. It is satisfied by stash.Stasher and lets the local sumdb populate
// storage on demand for modules that have not been requested through the
// module endpoints yet.
type Fetcher interface {
	Stash(ctx context.Context, mod, ver string) (string, error)
}

// Server implements sumdb.ServerOps for a local checksum database.
// It lazily computes hashes from the storage backend on first lookup,
// fetching the module into storage on demand if it is not present.
type Server struct {
	mu      sync.Mutex
	dir     string
	signer  note.Signer
	vkey    string
	name    string
	storage storage.Backend
	checker storage.Checker
	fetcher Fetcher

	nrecords int64
	lookup   map[string]int64 // "module@version" → record ID
	hashes   *os.File         // flat file of stored hashes (tlog internal nodes)
}

// New creates or opens a local sumdb server.
// dir is the directory for persistent state.
// name is the server name for signatures (e.g. "athens.local").
// s is the storage backend used to fetch module zips and go.mod files for hashing.
// fetcher, when non-nil, is used to download a module into storage on demand
// when a lookup arrives for a module that is not yet stored. Pass nil to only
// serve modules that are already present in storage.
func New(dir, name string, s storage.Backend, fetcher Fetcher) (*Server, error) {
	if err := os.MkdirAll(filepath.Join(dir, "records"), 0o750); err != nil {
		return nil, fmt.Errorf("sumlocal: mkdir: %w", err)
	}

	srv := &Server{
		dir:     dir,
		name:    name,
		storage: s,
		checker: storage.WithChecker(s),
		fetcher: fetcher,
		lookup:  make(map[string]int64),
	}

	if err := srv.loadOrGenerateKey(); err != nil {
		return nil, fmt.Errorf("sumlocal: key: %w", err)
	}

	hashPath := filepath.Join(dir, "hashes.bin")
	f, err := os.OpenFile(hashPath, os.O_RDWR|os.O_CREATE, 0o600) //nolint:gosec // path is constructed from trusted dir parameter
	if err != nil {
		return nil, fmt.Errorf("sumlocal: open hashes: %w", err)
	}
	srv.hashes = f

	if err := srv.rebuild(); err != nil {
		return nil, fmt.Errorf("sumlocal: rebuild: %w", err)
	}

	return srv, nil
}

// Handler returns an HTTP handler implementing the sumdb protocol.
func (s *Server) Handler() *sumdb.Server {
	return sumdb.NewServer(s)
}

// VerifierKey returns the public verifier key string for GOSUMDB configuration.
func (s *Server) VerifierKey() string {
	return s.vkey
}

// Name returns the server name.
func (s *Server) Name() string {
	return s.name
}

func (s *Server) loadOrGenerateKey() error {
	skeyPath := filepath.Join(s.dir, "skey")
	vkeyPath := filepath.Join(s.dir, "vkey")

	skeyData, err := os.ReadFile(skeyPath) //nolint:gosec // path is constructed from trusted dir
	switch {
	case os.IsNotExist(err):
		skey, vkey, err := note.GenerateKey(rand.Reader, s.name)
		if err != nil {
			return err
		}
		if err := os.WriteFile(skeyPath, []byte(skey), 0o600); err != nil {
			return err
		}
		if err := os.WriteFile(vkeyPath, []byte(vkey), 0o600); err != nil {
			return err
		}
		skeyData = []byte(skey)
		s.vkey = vkey
	case err != nil:
		return err
	default:
		vkeyData, err := os.ReadFile(vkeyPath) //nolint:gosec // path is constructed from trusted dir
		if err != nil {
			return err
		}
		s.vkey = string(vkeyData)
	}

	signer, err := note.NewSigner(string(skeyData))
	if err != nil {
		return err
	}
	s.signer = signer
	return nil
}

// rebuild reconstructs in-memory lookup from on-disk records.
func (s *Server) rebuild() error {
	for id := int64(0); ; id++ {
		data, err := os.ReadFile(filepath.Join(s.dir, "records", strconv.FormatInt(id, 10)))
		if os.IsNotExist(err) {
			s.nrecords = id
			return nil
		}
		if err != nil {
			return err
		}
		lines := strings.SplitN(string(data), "\n", 2)
		if len(lines) >= 1 {
			parts := strings.Fields(lines[0])
			if len(parts) >= 2 {
				s.lookup[parts[0]+"@"+parts[1]] = id
			}
		}
	}
}

// Signed implements sumdb.ServerOps.
func (s *Server) Signed(_ context.Context) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.signedTreeHead()
}

func (s *Server) signedTreeHead() ([]byte, error) {
	tree, err := s.currentTree()
	if err != nil {
		return nil, err
	}
	text := tlog.FormatTree(tree)
	return note.Sign(&note.Note{Text: string(text)}, s.signer)
}

func (s *Server) currentTree() (tlog.Tree, error) {
	if s.nrecords == 0 {
		return tlog.Tree{N: 0}, nil
	}
	th, err := tlog.TreeHash(s.nrecords, s.hashReader())
	if err != nil {
		return tlog.Tree{}, err
	}
	return tlog.Tree{N: s.nrecords, Hash: th}, nil
}

// hashReader returns a HashReader that reads directly from the flat hash file.
// This avoids TileHashReader's tree authentication which is unnecessary
// since we are the server and trust our own data.
func (s *Server) hashReader() tlog.HashReaderFunc {
	return func(indexes []int64) ([]tlog.Hash, error) {
		hashes := make([]tlog.Hash, len(indexes))
		for i, idx := range indexes {
			if _, err := s.hashes.ReadAt(hashes[i][:], idx*int64(tlog.HashSize)); err != nil {
				return nil, fmt.Errorf("read hash at index %d: %w", idx, err)
			}
		}
		return hashes, nil
	}
}

// ReadRecords implements sumdb.ServerOps.
func (s *Server) ReadRecords(_ context.Context, id, n int64) ([][]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records := make([][]byte, n)
	for i := int64(0); i < n; i++ {
		data, err := os.ReadFile(filepath.Join(s.dir, "records", strconv.FormatInt(id+i, 10)))
		if err != nil {
			return nil, err
		}
		records[i] = data
	}
	return records, nil
}

// Lookup implements sumdb.ServerOps.
// If the module@version is not yet in the log, it ensures the module is present
// in storage (fetching it on demand when a fetcher is configured), computes the
// zip and go.mod hashes, and adds the record.
func (s *Server) Lookup(ctx context.Context, m module.Version) (int64, error) {
	key := m.Path + "@" + m.Version

	s.mu.Lock()
	if id, ok := s.lookup[key]; ok {
		s.mu.Unlock()
		return id, nil
	}
	s.mu.Unlock()

	// Make sure the module is in storage and compute its hashes. This is done
	// without the lock held because fetching a module can be slow (the
	// golang.org/toolchain module is tens of megabytes), and we do not want to
	// block every other sumdb request while one module downloads.
	if err := s.ensureStored(ctx, m.Path, m.Version); err != nil {
		return 0, &os.PathError{Op: "lookup", Path: key, Err: os.ErrNotExist}
	}
	zipHash, modHash, err := s.computeHashes(ctx, m.Path, m.Version)
	if err != nil {
		return 0, &os.PathError{Op: "lookup", Path: key, Err: os.ErrNotExist}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// Another goroutine may have added this record while we were fetching.
	if id, ok := s.lookup[key]; ok {
		return id, nil
	}
	id, err := s.addRecord(m.Path, m.Version, zipHash, modHash)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// ensureStored makes sure module@version is present in the storage backend,
// fetching it on demand via the configured fetcher when it is missing. When no
// fetcher is configured it only reports whether the module is already stored.
func (s *Server) ensureStored(ctx context.Context, mod, ver string) error {
	exists, err := s.checker.Exists(ctx, mod, ver)
	if err == nil && exists {
		return nil
	}
	if s.fetcher == nil {
		if err != nil {
			return err
		}
		return &os.PathError{Op: "exists", Path: mod + "@" + ver, Err: os.ErrNotExist}
	}
	// Stash downloads the module from upstream and saves it into storage. It is
	// singleflighted, so concurrent lookups for the same module coalesce into a
	// single download. Note this deliberately bypasses the HTTP module-filter
	// middleware: a client asking the sumdb to verify a module needs the real
	// checksum regardless of whether the proxy would serve the module bytes.
	if _, err := s.fetcher.Stash(ctx, mod, ver); err != nil {
		return err
	}
	return nil
}

// ReadTileData implements sumdb.ServerOps (hash tiles only, L >= 0).
func (s *Server) ReadTileData(_ context.Context, t tlog.Tile) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readHashTile(t)
}

func (s *Server) readHashTile(tile tlog.Tile) ([]byte, error) {
	result := make([]byte, 0, tile.W*tlog.HashSize)
	for i := 0; i < tile.W; i++ {
		level := tile.L * tile.H
		n := tile.N*int64(1<<uint(tile.H)) + int64(i) //nolint:gosec // tile.H is always a small positive int from tlog
		idx := tlog.StoredHashIndex(level, n)

		h := make([]byte, tlog.HashSize)
		if _, err := s.hashes.ReadAt(h, idx*int64(tlog.HashSize)); err != nil {
			return nil, fmt.Errorf("read hash at stored index %d: %w", idx, err)
		}
		result = append(result, h...)
	}
	return result, nil
}

// addRecord appends a new record to the log. Caller must hold s.mu.
func (s *Server) addRecord(mod, ver, zipHash, modHash string) (int64, error) {
	id := s.nrecords
	text := fmt.Sprintf("%s %s %s\n%s %s/go.mod %s\n", mod, ver, zipHash, mod, ver, modHash)

	// Compute record hash
	recHash := tlog.RecordHash([]byte(text))

	// Compute new stored hashes using direct hash reader
	newHashes, err := tlog.StoredHashesForRecordHash(id, recHash, s.hashReader())
	if err != nil {
		return 0, fmt.Errorf("compute stored hashes: %w", err)
	}

	// Write new hashes to flat file
	baseIdx := tlog.StoredHashCount(id)
	for i, h := range newHashes {
		offset := (baseIdx + int64(i)) * int64(tlog.HashSize)
		if _, err := s.hashes.WriteAt(h[:], offset); err != nil {
			return 0, fmt.Errorf("write hash: %w", err)
		}
	}

	// Save record text
	recordPath := filepath.Join(s.dir, "records", strconv.FormatInt(id, 10))
	if err := os.WriteFile(recordPath, []byte(text), 0o600); err != nil {
		return 0, err
	}

	s.lookup[mod+"@"+ver] = id
	s.nrecords = id + 1
	return id, nil
}

// computeHashes fetches a module's zip and go.mod from storage and computes
// the h1: hashes used in go.sum.
func (s *Server) computeHashes(ctx context.Context, mod, ver string) (zipHash, modHash string, err error) {
	// Get go.mod and compute its hash
	gomod, err := s.storage.GoMod(ctx, mod, ver)
	if err != nil {
		return "", "", fmt.Errorf("get go.mod: %w", err)
	}
	modHash, err = HashGoMod(gomod)
	if err != nil {
		return "", "", fmt.Errorf("hash go.mod: %w", err)
	}

	// Get zip and compute its hash
	zipRC, err := s.storage.Zip(ctx, mod, ver)
	if err != nil {
		return "", "", fmt.Errorf("get zip: %w", err)
	}
	defer zipRC.Close()

	zipData, err := io.ReadAll(zipRC)
	if err != nil {
		return "", "", fmt.Errorf("read zip: %w", err)
	}

	zipHash, err = HashZipData(zipData)
	if err != nil {
		return "", "", fmt.Errorf("hash zip: %w", err)
	}

	return zipHash, modHash, nil
}

// HashZipData computes the h1: hash of a module zip from in-memory data.
// This is equivalent to dirhash.HashZip but works without a file on disk.
func HashZipData(data []byte) (string, error) {
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}

	files := make([]string, 0, len(z.File))
	zfiles := make(map[string]*zip.File)
	for _, f := range z.File {
		files = append(files, f.Name)
		zfiles[f.Name] = f
	}
	sort.Strings(files)

	return dirhash.Hash1(files, func(name string) (io.ReadCloser, error) {
		f := zfiles[name]
		if f == nil {
			return nil, fmt.Errorf("file not found: %s", name)
		}
		return f.Open()
	})
}

// HashGoMod computes the h1: hash of a go.mod file.
func HashGoMod(mod []byte) (string, error) {
	return dirhash.Hash1([]string{"go.mod"}, func(string) (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(mod)), nil
	})
}

// Close closes the server and its resources.
func (s *Server) Close() error {
	return s.hashes.Close()
}

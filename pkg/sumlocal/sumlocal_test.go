package sumlocal

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"golang.org/x/mod/module"
	"golang.org/x/mod/sumdb"
	"golang.org/x/mod/sumdb/tlog"

	"github.com/gomods/athens/pkg/storage"
	"github.com/gomods/athens/pkg/storage/mem"
	"github.com/wow-look-at-my/testify/assert"
	"github.com/wow-look-at-my/testify/require"
)

// makeModuleZip creates a minimal valid Go module zip for testing.
func makeModuleZip(t *testing.T, modPath, version, goModContent string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	prefix := modPath + "@" + version + "/"

	// Add go.mod
	f, err := w.Create(prefix + "go.mod")
	require.Nil(t, err)
	_, err = f.Write([]byte(goModContent))
	require.Nil(t, err)

	// Add a simple .go file
	f, err = w.Create(prefix + "main.go")
	require.Nil(t, err)
	_, err = f.Write([]byte("package main\n"))
	require.Nil(t, err)

	require.Nil(t, w.Close())
	return buf.Bytes()
}

// setupTestServer creates a Server with an in-memory storage backend
// containing test modules.
func setupTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()

	store, _ := mem.NewStorage()
	ctx := context.Background()

	// Add a test module to storage
	modContent := []byte("module example.com/foo\n\ngo 1.21\n")
	zipData := makeModuleZip(t, "example.com/foo", "v1.0.0", string(modContent))
	err := store.Save(ctx, "example.com/foo", "v1.0.0", modContent, bytes.NewReader(zipData), nil, []byte(`{"Version":"v1.0.0"}`))
	require.Nil(t, err)

	srv, err := New(dir, "test.athens.local", store, nil)
	require.Nil(t, err)
	t.Cleanup(func() { srv.Close() })

	return srv, dir
}

func TestNew(t *testing.T) {
	dir := t.TempDir()
	store, _ := mem.NewStorage()

	srv, err := New(dir, "test.local", store, nil)
	require.Nil(t, err)
	defer srv.Close()

	assert.Equal(t, "test.local", srv.Name())
	assert.NotEmpty(t, srv.VerifierKey())
	assert.True(t, strings.HasPrefix(srv.VerifierKey(), "test.local+"))
}

func TestNewReopensExistingState(t *testing.T) {
	dir := t.TempDir()
	store, _ := mem.NewStorage()
	ctx := context.Background()

	// Add a module to storage
	modContent := []byte("module example.com/bar\n\ngo 1.21\n")
	zipData := makeModuleZip(t, "example.com/bar", "v1.0.0", string(modContent))
	require.Nil(t, store.Save(ctx, "example.com/bar", "v1.0.0", modContent, bytes.NewReader(zipData), nil, []byte(`{"Version":"v1.0.0"}`)))

	// Create server and add a record
	srv1, err := New(dir, "test.local", store, nil)
	require.Nil(t, err)
	_, err = srv1.Lookup(ctx, module.Version{Path: "example.com/bar", Version: "v1.0.0"})
	require.Nil(t, err)
	vkey1 := srv1.VerifierKey()
	srv1.Close()

	// Reopen with same dir - should preserve state
	srv2, err := New(dir, "test.local", store, nil)
	require.Nil(t, err)
	defer srv2.Close()

	assert.Equal(t, vkey1, srv2.VerifierKey())
	assert.Equal(t, int64(1), srv2.nrecords)

	// Should find the previously added record
	id, err := srv2.Lookup(ctx, module.Version{Path: "example.com/bar", Version: "v1.0.0"})
	require.Nil(t, err)
	assert.Equal(t, int64(0), id)
}

func TestLookup(t *testing.T) {
	srv, _ := setupTestServer(t)
	ctx := context.Background()

	// First lookup should compute hash and add record
	id, err := srv.Lookup(ctx, module.Version{Path: "example.com/foo", Version: "v1.0.0"})
	require.Nil(t, err)
	assert.Equal(t, int64(0), id)

	// Second lookup should return same ID (cached)
	id2, err := srv.Lookup(ctx, module.Version{Path: "example.com/foo", Version: "v1.0.0"})
	require.Nil(t, err)
	assert.Equal(t, id, id2)
}

func TestLookupNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	ctx := context.Background()

	_, err := srv.Lookup(ctx, module.Version{Path: "example.com/nonexistent", Version: "v1.0.0"})
	require.NotNil(t, err)
}

func TestReadRecords(t *testing.T) {
	srv, _ := setupTestServer(t)
	ctx := context.Background()

	// Add a record first
	_, err := srv.Lookup(ctx, module.Version{Path: "example.com/foo", Version: "v1.0.0"})
	require.Nil(t, err)

	records, err := srv.ReadRecords(ctx, 0, 1)
	require.Nil(t, err)
	require.Len(t, records, 1)

	text := string(records[0])
	assert.Contains(t, text, "example.com/foo v1.0.0 h1:")
	assert.Contains(t, text, "example.com/foo v1.0.0/go.mod h1:")
}

func TestSigned(t *testing.T) {
	srv, _ := setupTestServer(t)
	ctx := context.Background()

	// Empty tree
	signed, err := srv.Signed(ctx)
	require.Nil(t, err)
	assert.NotEmpty(t, signed)

	// After adding a record
	_, err = srv.Lookup(ctx, module.Version{Path: "example.com/foo", Version: "v1.0.0"})
	require.Nil(t, err)

	signed2, err := srv.Signed(ctx)
	require.Nil(t, err)
	assert.NotEmpty(t, signed2)
	// Tree should be different after adding a record
	assert.NotEqual(t, string(signed), string(signed2))
}

func TestReadTileData(t *testing.T) {
	srv, _ := setupTestServer(t)
	ctx := context.Background()

	// Add a record
	_, err := srv.Lookup(ctx, module.Version{Path: "example.com/foo", Version: "v1.0.0"})
	require.Nil(t, err)

	// Read the leaf hash tile
	tile := tlog.Tile{H: 8, L: 0, N: 0, W: 1}
	data, err := srv.ReadTileData(ctx, tile)
	require.Nil(t, err)
	assert.Len(t, data, tlog.HashSize)
}

func TestMultipleRecords(t *testing.T) {
	dir := t.TempDir()
	store, _ := mem.NewStorage()
	ctx := context.Background()

	// Add multiple modules
	for _, name := range []string{"a", "b", "c"} {
		modPath := "example.com/" + name
		modContent := []byte("module " + modPath + "\n\ngo 1.21\n")
		zipData := makeModuleZip(t, modPath, "v1.0.0", string(modContent))
		require.Nil(t, store.Save(ctx, modPath, "v1.0.0", modContent, bytes.NewReader(zipData), nil, []byte(`{"Version":"v1.0.0"}`)))
	}

	srv, err := New(dir, "test.local", store, nil)
	require.Nil(t, err)
	defer srv.Close()

	// Lookup all three
	for i, name := range []string{"a", "b", "c"} {
		id, err := srv.Lookup(ctx, module.Version{Path: "example.com/" + name, Version: "v1.0.0"})
		require.Nil(t, err)
		assert.Equal(t, int64(i), id)
	}

	assert.Equal(t, int64(3), srv.nrecords)

	// Verify tree is consistent
	signed, err := srv.Signed(ctx)
	require.Nil(t, err)
	assert.NotEmpty(t, signed)
}

func TestHashZipData(t *testing.T) {
	modContent := "module example.com/test\n\ngo 1.21\n"
	zipData := makeModuleZip(t, "example.com/test", "v1.0.0", modContent)

	hash, err := HashZipData(zipData)
	require.Nil(t, err)
	assert.True(t, strings.HasPrefix(hash, "h1:"))
	assert.Len(t, hash, 47) // "h1:" + 44 base64 chars
}

func TestHashGoMod(t *testing.T) {
	mod := []byte("module example.com/test\n\ngo 1.21\n")
	hash, err := HashGoMod(mod)
	require.Nil(t, err)
	assert.True(t, strings.HasPrefix(hash, "h1:"))

	// Same content should give same hash
	hash2, err := HashGoMod(mod)
	require.Nil(t, err)
	assert.Equal(t, hash, hash2)

	// Different content should give different hash
	hash3, err := HashGoMod([]byte("module example.com/other\n"))
	require.Nil(t, err)
	assert.NotEqual(t, hash, hash3)
}

func TestHTTPHandler(t *testing.T) {
	srv, _ := setupTestServer(t)
	ctx := context.Background()

	// Pre-populate a record so the lookup endpoint works
	_, err := srv.Lookup(ctx, module.Version{Path: "example.com/foo", Version: "v1.0.0"})
	require.Nil(t, err)

	handler := srv.Handler()
	ts := httptest.NewServer(handler)
	defer ts.Close()

	t.Run("latest", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/latest")
		require.Nil(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.Nil(t, err)
		// Should contain tree head text
		assert.Contains(t, string(body), "go.sum database tree")
	})

	t.Run("lookup", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/lookup/example.com/foo@v1.0.0")
		require.Nil(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.Nil(t, err)
		bodyStr := string(body)
		assert.Contains(t, bodyStr, "example.com/foo v1.0.0 h1:")
		assert.Contains(t, bodyStr, "example.com/foo v1.0.0/go.mod h1:")
		assert.Contains(t, bodyStr, "go.sum database tree")
	})

	t.Run("lookup not found", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/lookup/example.com/missing@v1.0.0")
		require.Nil(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("tile hash", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/tile/8/0/000.p/1")
		require.Nil(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.Nil(t, err)
		assert.Len(t, body, tlog.HashSize)
	})

	t.Run("tile data", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/tile/8/data/000.p/1")
		require.Nil(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.Nil(t, err)
		assert.Contains(t, string(body), "example.com/foo v1.0.0")
	})
}

func TestServerPaths(t *testing.T) {
	// Verify we implement all required ServerOps methods
	srv, _ := setupTestServer(t)
	var _ sumdb.ServerOps = srv
}

func TestClose(t *testing.T) {
	dir := t.TempDir()
	store, _ := mem.NewStorage()
	srv, err := New(dir, "test.local", store, nil)
	require.Nil(t, err)
	require.Nil(t, srv.Close())
}

func TestKeyPersistence(t *testing.T) {
	dir := t.TempDir()
	store, _ := mem.NewStorage()

	// Create server - generates keys
	srv1, err := New(dir, "test.local", store, nil)
	require.Nil(t, err)
	vkey1 := srv1.VerifierKey()
	srv1.Close()

	// Key files should exist
	_, err = os.Stat(dir + "/skey")
	require.Nil(t, err)
	_, err = os.Stat(dir + "/vkey")
	require.Nil(t, err)

	// Reopen - should load same keys
	srv2, err := New(dir, "test.local", store, nil)
	require.Nil(t, err)
	defer srv2.Close()
	assert.Equal(t, vkey1, srv2.VerifierKey())
}

// fakeFetcher implements Fetcher by saving a pre-built module into storage the
// first time it is asked for, mimicking how stash.Stasher populates storage
// from upstream.
type fakeFetcher struct {
	t     *testing.T
	store storage.Backend
	calls int
	zip   []byte
	gomod []byte
}

func (f *fakeFetcher) Stash(ctx context.Context, mod, ver string) (string, error) {
	f.calls++
	err := f.store.Save(ctx, mod, ver, f.gomod, bytes.NewReader(f.zip), nil, []byte(`{"Version":"`+ver+`"}`))
	require.Nil(f.t, err)
	return ver, nil
}

// TestLookupFetchesOnDemand verifies that a lookup for a module that is not yet
// in storage triggers the fetcher to download it, then succeeds. This is the
// path Go relies on when verifying golang.org/toolchain during a toolchain
// switch: the module may have been downloaded via a GOPROXY "direct" fallback,
// so it is never stashed through the module endpoints.
func TestLookupFetchesOnDemand(t *testing.T) {
	dir := t.TempDir()
	store, _ := mem.NewStorage()
	ctx := context.Background()

	const modPath, version = "golang.org/toolchain", "v0.0.1-go1.99.0.linux-amd64"
	modContent := []byte("module golang.org/toolchain\n\ngo 1.99\n")
	zipData := makeModuleZip(t, modPath, version, string(modContent))

	fetcher := &fakeFetcher{t: t, store: store, zip: zipData, gomod: modContent}

	srv, err := New(dir, "test.local", store, fetcher)
	require.Nil(t, err)
	defer srv.Close()

	// The module is not in storage yet, so the lookup must fetch it.
	id, err := srv.Lookup(ctx, module.Version{Path: modPath, Version: version})
	require.Nil(t, err)
	assert.Equal(t, int64(0), id)
	assert.Equal(t, 1, fetcher.calls, "fetcher should be called exactly once")

	// The record must carry the real hashes computed from the fetched zip/go.mod.
	records, err := srv.ReadRecords(ctx, 0, 1)
	require.Nil(t, err)
	require.Len(t, records, 1)
	text := string(records[0])
	assert.Contains(t, text, modPath+" "+version+" h1:")
	assert.Contains(t, text, modPath+" "+version+"/go.mod h1:")

	// A second lookup is served from cache without re-fetching.
	_, err = srv.Lookup(ctx, module.Version{Path: modPath, Version: version})
	require.Nil(t, err)
	assert.Equal(t, 1, fetcher.calls, "second lookup should not re-fetch")
}

// TestLookupFetcherFailureIsNotFound verifies that when the fetcher cannot
// supply a module, the lookup surfaces a not-found error (which the sumdb HTTP
// layer turns into a 404) rather than hanging or panicking.
func TestLookupFetcherFailureIsNotFound(t *testing.T) {
	dir := t.TempDir()
	store, _ := mem.NewStorage()
	ctx := context.Background()

	srv, err := New(dir, "test.local", store, failingFetcher{})
	require.Nil(t, err)
	defer srv.Close()

	_, err = srv.Lookup(ctx, module.Version{Path: "example.com/missing", Version: "v1.0.0"})
	require.NotNil(t, err)
	assert.True(t, os.IsNotExist(err))
}

type failingFetcher struct{}

func (failingFetcher) Stash(context.Context, string, string) (string, error) {
	return "", os.ErrNotExist
}

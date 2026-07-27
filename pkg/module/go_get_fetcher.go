package module

import (
	"bytes"
	"context"
	"crypto/md5" //nolint:gosec
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/observ"
	"github.com/gomods/athens/pkg/storage"
	"github.com/spf13/afero"
	gomodule "golang.org/x/mod/module"
)

type goGetFetcher struct {
	fs           afero.Fs
	goBinaryName string
	envVars      []string
	gogetDir     string
}

type goModule struct {
	Path     string `json:"path"`     // module path
	Version  string `json:"version"`  // module version
	Error    string `json:"error"`    // error loading module
	Info     string `json:"info"`     // absolute path to cached .info file
	GoMod    string `json:"goMod"`    // absolute path to cached .mod file
	Zip      string `json:"zip"`      // absolute path to cached .zip file
	Dir      string `json:"dir"`      // absolute path to cached source root directory
	Sum      string `json:"sum"`      // checksum for path, version (as in go.sum)
	GoModSum string `json:"goModSum"` // checksum for go.mod (as in go.sum)
}

// NewGoGetFetcher creates fetcher which uses go get tool to fetch modules.
func NewGoGetFetcher(goBinaryName, gogetDir string, envVars []string, fs afero.Fs) (Fetcher, error) {
	const op errors.Op = "module.NewGoGetFetcher"
	if err := validGoBinary(goBinaryName); err != nil {
		return nil, errors.E(op, err)
	}
	return &goGetFetcher{
		fs:           fs,
		goBinaryName: goBinaryName,
		envVars:      envVars,
		gogetDir:     gogetDir,
	}, nil
}

// Fetch downloads the sources from the go binary and returns the corresponding
// .info, .mod, and .zip files.
func (g *goGetFetcher) Fetch(ctx context.Context, mod, ver string) (*storage.Version, error) {
	const op errors.Op = "goGetFetcher.Fetch"
	ctx, span := observ.StartSpan(ctx, op.String())
	defer span.End()

	if err := gomodule.CheckPath(mod); err != nil {
		return nil, errors.E(op, err, errors.KindNotFound)
	}

	// setup the GOPATH
	goPathRoot, err := afero.TempDir(g.fs, g.gogetDir, "athens")
	if err != nil {
		return nil, errors.E(op, err)
	}
	sourcePath := filepath.Join(goPathRoot, "src")
	modPath := filepath.Join(sourcePath, getRepoDirName(mod, ver))
	if err := g.fs.MkdirAll(modPath, os.ModeDir|os.ModePerm); err != nil {
		_ = clearFiles(g.fs, goPathRoot)
		return nil, errors.E(op, err)
	}

	m, err := downloadModule(
		ctx,
		g.goBinaryName,
		g.envVars,
		goPathRoot,
		modPath,
		mod,
		ver,
	)
	if err != nil {
		_ = clearFiles(g.fs, goPathRoot)
		return nil, errors.E(op, err)
	}

	var storageVer storage.Version
	storageVer.Semver = m.Version
	info, err := afero.ReadFile(g.fs, m.Info)
	if err != nil {
		return nil, errors.E(op, err)
	}
	storageVer.Info = info

	gomod, err := afero.ReadFile(g.fs, m.GoMod)
	if err != nil {
		return nil, errors.E(op, err)
	}
	storageVer.Mod = gomod

	zipMD5, err := func() ([]byte, error) {
		// Perform in a separate function to ensure file is closed
		zipForChecksum, err := g.fs.Open(m.Zip)
		if err != nil {
			return nil, errors.E(op, err)
		}
		defer zipForChecksum.Close()

		//nolint:gosec
		hash := md5.New()
		if _, err := io.Copy(hash, zipForChecksum); err != nil {
			return nil, errors.E(op, err)
		}

		return hash.Sum(nil), nil
	}()
	if err != nil {
		return nil, err
	}

	zip, err := g.fs.Open(m.Zip)
	if err != nil {
		return nil, errors.E(op, err)
	}
	// note: don't close zip here so that the caller can read directly from disk.
	//
	// if we close, then the caller will panic, and the alternative to make this work is
	// that we read into memory and return an io.ReadCloser that reads out of memory
	storageVer.Zip = &zipReadCloser{zip, g.fs, goPathRoot}
	storageVer.ZipMD5 = zipMD5

	return &storageVer, nil
}

// given a filesystem, gopath, repository root, module and version, runs 'go mod download -json'
// on module@version from the repoRoot with GOPATH=gopath, and returns a non-nil error if anything went wrong.
func downloadModule(
	ctx context.Context,
	goBinaryName string,
	envVars []string,
	gopath,
	repoRoot,
	module,
	version string,
) (goModule, error) {
	const op errors.Op = "module.downloadModule"

	uri := strings.TrimSuffix(module, "/")
	fullURI := fmt.Sprintf("%s@%s", uri, version)

	cmd := exec.CommandContext(ctx, goBinaryName, "mod", "download", "-json", fullURI)
	env := prepareEnv(gopath, envVars)
	if uri == toolchainModulePath {
		env = withProxyUserAgent(env)
	}
	cmd.Env = env
	cmd.Dir = repoRoot
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	if err != nil && !errors.IsNoChildProcessesErr(err) {
		err = fmt.Errorf("%w: %s", err, stderr)
		var m goModule
		if jsonErr := json.NewDecoder(stdout).Decode(&m); jsonErr != nil {
			return goModule{}, errors.E(op, err, errors.KindNotFound)
		}
		// github quota exceeded
		if isLimitHit(m.Error) {
			return goModule{}, errors.E(op, m.Error, errors.KindRateLimit)
		}
		return goModule{}, errors.E(op, m.Error, errors.KindNotFound)
	}

	var m goModule
	if err = json.NewDecoder(stdout).Decode(&m); err != nil {
		return goModule{}, errors.E(op, err, errors.KindNotFound)
	}
	if m.Error != "" {
		return goModule{}, errors.E(op, m.Error, errors.KindNotFound)
	}

	return m, nil
}

// toolchainModulePath is the synthetic module Go uses to distribute toolchains
// for its automatic toolchain switching (since Go 1.21).
const toolchainModulePath = "golang.org/toolchain"

// proxyUserAgent is a GIT_HTTP_USER_AGENT value carrying the substring cmd/go
// looks for to recognise that the caller is itself a Go module proxy doing the
// initial toolchain download.
//
// cmd/go requires a checksum-database lookup for golang.org/toolchain and
// ignores GONOSUMCHECK/GONOSUMDB/GOPRIVATE, so under GOSUMDB=off the download
// otherwise fails with "checksum database disabled by GOSUMDB=off". cmd/go's
// own escape hatch is "the Go proxy+checksum database cannot check itself while
// doing the initial download": useSumDB() returns false for the toolchain when
// GIT_HTTP_USER_AGENT contains "proxy.golang.org". Athens is exactly that -- a
// module proxy fetching the toolchain to cache and re-serve it -- so we set the
// agent and let the toolchain be fetched (and hashed by Athens) like any other
// module, with no checksum database, and therefore no contact with
// sum.golang.org.
//
// See cmd/go/internal/modfetch/sumdb.go, func useSumDB.
const proxyUserAgent = "GIT_HTTP_USER_AGENT=Athens module proxy (proxy.golang.org)"

// withProxyUserAgent returns env with GIT_HTTP_USER_AGENT set to proxyUserAgent,
// replacing any existing value, so golang.org/toolchain can be fetched under
// GOSUMDB=off via cmd/go's proxy self-download exception.
func withProxyUserAgent(env []string) []string {
	out := make([]string, 0, len(env)+1)
	for _, kv := range env {
		if strings.HasPrefix(kv, "GIT_HTTP_USER_AGENT=") {
			continue
		}
		out = append(out, kv)
	}
	return append(out, proxyUserAgent)
}

func isLimitHit(o string) bool {
	return strings.Contains(o, "403 response from api.github.com")
}

// getRepoDirName takes a raw repository URI and a version and creates a directory name that the
// repository contents can be put into.
func getRepoDirName(repoURI, version string) string {
	escapedURI := strings.ReplaceAll(repoURI, "/", "-")
	return fmt.Sprintf("%s-%s", escapedURI, version)
}

func validGoBinary(name string) error {
	const op errors.Op = "module.validGoBinary"
	err := exec.Command(name).Run()
	eErr := &exec.ExitError{}
	if err != nil && !errors.AsErr(err, &eErr) {
		return errors.E(op, err)
	}
	return nil
}

package module

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"

	"github.com/gomods/athens/pkg/errors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

func TestGoGetFetcherInvalidModulePaths(t *testing.T) {
	t.Parallel()
	fetcher, err := NewGoGetFetcher("go", "", nil, afero.NewMemMapFs())
	require.Nil(t, err)

	tests := []struct {
		name string
		mod  string
	}{
		{"bare host", "github.com"},
		{"host with owner only", "github.com/owner"},
		{"empty string", ""},
		{"single element", "foo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, fetchErr := fetcher.Fetch(context.Background(), tt.mod, "v1.0.0")
			require.NotNil(t, fetchErr)

			kind := errors.Kind(fetchErr)
			assert.Equal(t, errors.KindNotFound, kind)

		})
	}
}

func TestWithProxyUserAgent(t *testing.T) {
	t.Parallel()

	const want = "GIT_HTTP_USER_AGENT=Athens module proxy (proxy.golang.org)"

	t.Run("appends the agent and leaves GOSUMDB=off and other vars alone", func(t *testing.T) {
		t.Parallel()
		got := withProxyUserAgent([]string{"GOPATH=/x", "GOSUMDB=off", "GOPROXY=direct"})
		// The toolchain is fetched like any other module: GOSUMDB is untouched,
		// only GIT_HTTP_USER_AGENT is added to trigger cmd/go's proxy exception.
		assert.Equal(t, []string{"GOPATH=/x", "GOSUMDB=off", "GOPROXY=direct", want}, got)
	})

	t.Run("replaces any pre-existing GIT_HTTP_USER_AGENT", func(t *testing.T) {
		t.Parallel()
		got := withProxyUserAgent([]string{"GIT_HTTP_USER_AGENT=something", "GOPROXY=direct"})
		assert.Equal(t, []string{"GOPROXY=direct", want}, got)
	})

	t.Run("agent carries the substring cmd/go's useSumDB exception matches", func(t *testing.T) {
		t.Parallel()
		got := withProxyUserAgent(nil)
		require.Len(t, got, 1)
		assert.Contains(t, got[0], "proxy.golang.org")
	})
}

func (s *ModuleSuite) TestNewGoGetFetcher() {
	r := s.Require()
	fetcher, err := NewGoGetFetcher(s.goBinaryName, "", s.env, s.fs)
	r.NoError(err)
	_, ok := fetcher.(*goGetFetcher)
	r.True(ok)
}

func (s *ModuleSuite) TestGoGetFetcherError() {
	fetcher, err := NewGoGetFetcher("invalidpath", "", s.env, afero.NewOsFs())

	assert.Nil(s.T(), fetcher)
	if runtime.GOOS == "windows" {
		assert.EqualError(s.T(), err, "exec: \"invalidpath\": executable file not found in %PATH%")
	} else {
		assert.EqualError(s.T(), err, "exec: \"invalidpath\": executable file not found in $PATH")
	}
}

func (s *ModuleSuite) TestGoGetFetcherFetch() {
	r := s.Require()
	// we need to use an OS filesystem because fetch executes vgo on the command line, which
	// always writes to the filesystem
	fetcher, err := NewGoGetFetcher(s.goBinaryName, "", s.env, afero.NewOsFs())
	r.NoError(err)
	ver, err := fetcher.Fetch(ctx, repoURI, version)
	r.NoError(err)
	defer ver.Zip.Close()

	r.True(len(ver.Info) > 0)

	r.True(len(ver.Mod) > 0)

	zipBytes, err := io.ReadAll(ver.Zip)
	r.NoError(err)
	r.True(len(zipBytes) > 0)

	// close the version's zip file (which also cleans up the underlying GOPATH) and expect it to fail again
	r.NoError(ver.Zip.Close())
}

func (s *ModuleSuite) TestNotFoundFetches() {
	r := s.Require()
	fetcher, err := NewGoGetFetcher(s.goBinaryName, "", s.env, afero.NewOsFs())
	r.NoError(err)
	// when someone buys laks47dfjoijskdvjxuyyd.com, and implements
	// a git server on top of it, this test will fail :)
	_, err = fetcher.Fetch(ctx, "laks47dfjoijskdvjxuyyd.com/pkg/errors", "v0.8.1")
	if err == nil {
		s.Fail("expected an error but got nil")
	}
	if errors.Kind(err) != errors.KindNotFound {
		s.Failf("incorrect error kind", "expected a not found error but got %v", errors.Kind(err))
	}
}

func (s *ModuleSuite) TestGoGetFetcherSumDB() {
	if os.Getenv("SKIP_UNTIL_113") != "" {
		return
	}
	r := s.Require()
	zipBytes, err := os.ReadFile("test_data/mockmod.xyz@v1.2.3.zip")
	r.NoError(err)
	mp := &mockProxy{paths: map[string][]byte{
		"/mockmod.xyz/@v/v1.2.3.info": []byte(`{"Version":"v1.2.3"}`),
		"/mockmod.xyz/@v/v1.2.3.mod":  []byte(`{"module mod}`),
		"/mockmod.xyz/@v/v1.2.3.zip":  zipBytes,
	}}
	proxyAddr, close := s.getProxy(mp)
	defer close()

	fetcher, err := NewGoGetFetcher(s.goBinaryName, "", []string{"GOPROXY=" + proxyAddr}, afero.NewOsFs())
	r.NoError(err)
	_, err = fetcher.Fetch(ctx, "mockmod.xyz", "v1.2.3")
	if err == nil {
		s.T().Fatal("expected a gosum error but got nil")
	}
	fetcher, err = NewGoGetFetcher(s.goBinaryName, "", []string{"GONOSUMDB=mockmod.xyz", "GOPROXY=" + proxyAddr}, afero.NewOsFs())
	r.NoError(err)
	_, err = fetcher.Fetch(ctx, "mockmod.xyz", "v1.2.3")
	r.NoError(err, "expected the go sum to not be consulted but got an error")
}

func (s *ModuleSuite) TestGoGetDir() {
	r := s.Require()
	t := s.T()
	dir, err := os.MkdirTemp("", "nested")
	r.NoError(err)
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	fetcher, err := NewGoGetFetcher(s.goBinaryName, dir, s.env, afero.NewOsFs())
	r.NoError(err)

	ver, err := fetcher.Fetch(ctx, repoURI, version)
	r.NoError(err)
	defer ver.Zip.Close()

	dirInfo, err := os.ReadDir(dir)
	r.NoError(err)

	require.Greater(t, len(dirInfo), 0)

}

func (s *ModuleSuite) getProxy(h http.Handler) (addr string, close func()) {
	srv := httptest.NewServer(h)
	return srv.URL, srv.Close
}

type mockProxy struct {
	paths map[string][]byte
}

func (m *mockProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp, ok := m.paths[r.URL.Path]
	if !ok {
		w.WriteHeader(404)
		return
	}
	w.Write(resp)
}

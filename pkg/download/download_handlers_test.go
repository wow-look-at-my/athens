package download

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gomods/athens/pkg/download/mode"
	"github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/log"
	"github.com/gomods/athens/pkg/storage"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

type successProtocol struct {
	Protocol
}

func (sp *successProtocol) List(ctx context.Context, mod string) ([]string, error) {
	return []string{"v1.0.0", "v1.1.0"}, nil
}

func (sp *successProtocol) Latest(ctx context.Context, mod string) (*storage.RevInfo, error) {
	return &storage.RevInfo{Version: "v1.1.0"}, nil
}

func (sp *successProtocol) Info(ctx context.Context, mod, ver string) ([]byte, error) {
	return []byte(`{"Version":"v1.0.0"}`), nil
}

func (sp *successProtocol) GoMod(ctx context.Context, mod, ver string) ([]byte, error) {
	return []byte("module github.com/test/mod\n"), nil
}

func (sp *successProtocol) Zip(ctx context.Context, mod, ver string) (storage.SizeReadCloser, error) {
	data := []byte("fake zip data")
	return &mockSizeReadCloser{Reader: bytes.NewReader(data), size: int64(len(data))}, nil
}

type mockSizeReadCloser struct {
	*bytes.Reader
	size int64
}

func (m *mockSizeReadCloser) Close() error { return nil }
func (m *mockSizeReadCloser) Size() int64  { return m.size }

type notFoundProtocol struct {
	Protocol
}

func (nf *notFoundProtocol) List(ctx context.Context, mod string) ([]string, error) {
	return nil, errors.E("test", "not found", errors.KindNotFound)
}

func (nf *notFoundProtocol) Latest(ctx context.Context, mod string) (*storage.RevInfo, error) {
	return nil, errors.E("test", "not found", errors.KindNotFound)
}

func (nf *notFoundProtocol) Info(ctx context.Context, mod, ver string) ([]byte, error) {
	return nil, errors.E("test", "not found", errors.KindNotFound)
}

func (nf *notFoundProtocol) GoMod(ctx context.Context, mod, ver string) ([]byte, error) {
	return nil, errors.E("test", "not found", errors.KindNotFound)
}

func (nf *notFoundProtocol) Zip(ctx context.Context, mod, ver string) (storage.SizeReadCloser, error) {
	return nil, errors.E("test", "not found", errors.KindNotFound)
}

func newTestRouter(p Protocol) *mux.Router {
	r := mux.NewRouter()
	df := &mode.DownloadFile{Mode: mode.Sync}
	RegisterHandlers(r, &HandlerOpts{
		Protocol:     p,
		Logger:       log.NoOpLogger(),
		DownloadFile: df,
	})
	return r
}

func TestListHandler_Success(t *testing.T) {
	r := newTestRouter(&successProtocol{})
	req := httptest.NewRequest("GET", "/github.com/test/mod/@v/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "v1.0.0")
	require.Contains(t, w.Body.String(), "v1.1.0")
}

func TestListHandler_NotFound(t *testing.T) {
	r := newTestRouter(&notFoundProtocol{})
	req := httptest.NewRequest("GET", "/github.com/test/mod/@v/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestLatestHandler_Success(t *testing.T) {
	r := newTestRouter(&successProtocol{})
	req := httptest.NewRequest("GET", "/github.com/test/mod/@latest", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "v1.1.0")
}

func TestLatestHandler_NotFound(t *testing.T) {
	r := newTestRouter(&notFoundProtocol{})
	req := httptest.NewRequest("GET", "/github.com/test/mod/@latest", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestInfoHandler_Success(t *testing.T) {
	r := newTestRouter(&successProtocol{})
	req := httptest.NewRequest("GET", "/github.com/test/mod/@v/v1.0.0.info", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "v1.0.0")
}

func TestInfoHandler_NotFound(t *testing.T) {
	r := newTestRouter(&notFoundProtocol{})
	req := httptest.NewRequest("GET", "/github.com/test/mod/@v/v1.0.0.info", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestModuleHandler_Success(t *testing.T) {
	r := newTestRouter(&successProtocol{})
	req := httptest.NewRequest("GET", "/github.com/test/mod/@v/v1.0.0.mod", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "module github.com/test/mod")
}

func TestModuleHandler_NotFound(t *testing.T) {
	r := newTestRouter(&notFoundProtocol{})
	req := httptest.NewRequest("GET", "/github.com/test/mod/@v/v1.0.0.mod", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestZipHandler_Success(t *testing.T) {
	r := newTestRouter(&successProtocol{})
	req := httptest.NewRequest("GET", "/github.com/test/mod/@v/v1.0.0.zip", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/zip", w.Header().Get("Content-Type"))
	body, _ := io.ReadAll(w.Body)
	require.Equal(t, "fake zip data", string(body))
}

func TestZipHandler_Head(t *testing.T) {
	r := newTestRouter(&successProtocol{})
	req := httptest.NewRequest("HEAD", "/github.com/test/mod/@v/v1.0.0.zip", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/zip", w.Header().Get("Content-Type"))
	require.Equal(t, "13", w.Header().Get("Content-Length"))
	require.Empty(t, w.Body.Bytes())
}

func TestZipHandler_NotFound(t *testing.T) {
	r := newTestRouter(&notFoundProtocol{})
	req := httptest.NewRequest("GET", "/github.com/test/mod/@v/v1.0.0.zip", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestLogEntryHandler(t *testing.T) {
	df := &mode.DownloadFile{Mode: mode.Sync}
	opts := &HandlerOpts{
		Protocol:     &successProtocol{},
		Logger:       log.NoOpLogger(),
		DownloadFile: df,
	}

	handler := LogEntryHandler(ListHandler, opts)
	require.NotNil(t, handler)

	r := mux.NewRouter()
	r.Handle("/{module:.+}/@v/list", handler)

	req := httptest.NewRequest("GET", "/github.com/test/mod/@v/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

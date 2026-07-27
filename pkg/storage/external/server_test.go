package external

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gomods/athens/pkg/storage/mem"
	"github.com/stretchr/testify/require"
)

func TestNewServer_List(t *testing.T) {
	s, err := mem.NewStorage()
	require.NoError(t, err)

	handler := NewServer(s)
	req := httptest.NewRequest(http.MethodGet, "/github.com/test/mod/@v/list", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestNewServer_Info_NotFound(t *testing.T) {
	s, err := mem.NewStorage()
	require.NoError(t, err)

	handler := NewServer(s)
	req := httptest.NewRequest(http.MethodGet, "/github.com/test/mod/@v/v1.0.0.info", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestNewServer_GoMod_NotFound(t *testing.T) {
	s, err := mem.NewStorage()
	require.NoError(t, err)

	handler := NewServer(s)
	req := httptest.NewRequest(http.MethodGet, "/github.com/test/mod/@v/v1.0.0.mod", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestNewServer_Zip_NotFound(t *testing.T) {
	s, err := mem.NewStorage()
	require.NoError(t, err)

	handler := NewServer(s)
	req := httptest.NewRequest(http.MethodGet, "/github.com/test/mod/@v/v1.0.0.zip", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestNewServer_Delete_NotFound(t *testing.T) {
	s, err := mem.NewStorage()
	require.NoError(t, err)

	handler := NewServer(s)
	req := httptest.NewRequest(http.MethodDelete, "/github.com/test/mod/@v/v1.0.0.delete", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestNewServer_WithData(t *testing.T) {
	s, err := mem.NewStorage()
	require.NoError(t, err)

	ctx := context.Background()
	modContent := []byte("module github.com/test/mod")
	infoContent := []byte(`{"Version":"v1.0.0"}`)

	zipContent := bytes.NewReader([]byte("fake zip content"))
	err = s.Save(ctx, "github.com/test/mod", "v1.0.0", modContent, zipContent, nil, infoContent)
	require.NoError(t, err)

	handler := NewServer(s)

	// Test list
	req := httptest.NewRequest(http.MethodGet, "/github.com/test/mod/@v/list", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "v1.0.0")

	// Test info
	req = httptest.NewRequest(http.MethodGet, "/github.com/test/mod/@v/v1.0.0.info", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Test mod
	req = httptest.NewRequest(http.MethodGet, "/github.com/test/mod/@v/v1.0.0.mod", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Test zip
	req = httptest.NewRequest(http.MethodGet, "/github.com/test/mod/@v/v1.0.0.zip", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.NotEmpty(t, w.Body.Bytes())
	require.NotEmpty(t, w.Header().Get("Content-Length"))

	// Test delete
	req = httptest.NewRequest(http.MethodDelete, "/github.com/test/mod/@v/v1.0.0.delete", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Verify deleted
	req = httptest.NewRequest(http.MethodGet, "/github.com/test/mod/@v/v1.0.0.info", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestNewServer_Save(t *testing.T) {
	s, err := mem.NewStorage()
	require.NoError(t, err)

	handler := NewServer(s)

	// Create multipart form
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	// Add info file
	infoWriter, err := mw.CreateFormFile("mod.info", "mod.info")
	require.NoError(t, err)
	_, err = infoWriter.Write([]byte(`{"Version":"v2.0.0"}`))
	require.NoError(t, err)

	// Add mod file
	modWriter, err := mw.CreateFormFile("mod.mod", "mod.mod")
	require.NoError(t, err)
	_, err = modWriter.Write([]byte("module github.com/test/savemod"))
	require.NoError(t, err)

	// Add zip file
	zipWriter, err := mw.CreateFormFile("mod.zip", "mod.zip")
	require.NoError(t, err)
	_, err = zipWriter.Write([]byte("fake zip data"))
	require.NoError(t, err)

	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/github.com/test/savemod/@v/v2.0.0.save", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Verify the module was saved by fetching its info
	req = httptest.NewRequest(http.MethodGet, "/github.com/test/savemod/@v/v2.0.0.info", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

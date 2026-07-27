package actions

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gomods/athens/pkg/config"
	"github.com/gomods/athens/pkg/storage/mem"
	"github.com/stretchr/testify/require"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	healthHandler(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestVersionHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	versionHandler(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestRobotsHandler(t *testing.T) {
	c := &config.Config{RobotsFile: "robots.txt"}
	handler := robotsHandler(c)

	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	require.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestReadinessHandler_Healthy(t *testing.T) {
	s, err := mem.NewStorage()
	require.NoError(t, err)

	handler := getReadinessHandler(s)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	// mem storage List returns empty, which is fine — no error
	require.Equal(t, http.StatusOK, w.Code)
}

func TestProxyHomeHandler(t *testing.T) {
	c := &config.Config{}
	handler := proxyHomeHandler(c)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "localhost:3000"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "Welcome to Athens")
}

func TestProxyHomeHandler_WithNoSumPatterns(t *testing.T) {
	c := &config.Config{
		NoSumPatterns: []string{"github.com/private/*"},
	}
	handler := proxyHomeHandler(c)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "localhost:3000"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "github.com/private/*")
}

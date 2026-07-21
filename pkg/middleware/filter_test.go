package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gomods/athens/pkg/module"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestNewFilterMiddleware_NoModule(t *testing.T) {
	mf := &module.Filter{}
	mw := NewFilterMiddleware(mf, "http://upstream.example.com")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := mw(inner)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestRedirectToUpstreamURL(t *testing.T) {
	tests := []struct {
		name     string
		upstream string
		path     string
		expected string
	}{
		{
			name:     "simple",
			upstream: "http://upstream.example.com",
			path:     "/github.com/test/@v/list",
			expected: "http://upstream.example.com/github.com/test/@v/list",
		},
		{
			name:     "trailing slash",
			upstream: "http://upstream.example.com/",
			path:     "/github.com/test/@v/list",
			expected: "http://upstream.example.com/github.com/test/@v/list",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			result := redirectToUpstreamURL(tc.upstream, req.URL)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestNewFilterMiddleware_WithRouter(t *testing.T) {
	mf := &module.Filter{}
	mw := NewFilterMiddleware(mf, "http://upstream.example.com")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := mux.NewRouter()
	r.Handle("/{module:.+}/@v/list", mw(inner)).Methods(http.MethodGet)

	req := httptest.NewRequest(http.MethodGet, "/github.com/test/mod/@v/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

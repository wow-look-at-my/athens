package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestNewValidationMiddleware_NoModule(t *testing.T) {
	mw := NewValidationMiddleware(&http.Client{}, "http://validator.example.com")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := mw(inner)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestNewValidationMiddleware_Valid(t *testing.T) {
	// Set up a mock validator server
	validatorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"valid": true})
	}))
	defer validatorServer.Close()

	mw := NewValidationMiddleware(validatorServer.Client(), validatorServer.URL)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := mux.NewRouter()
	r.Handle("/{module:.+}/@v/{version}.info", mw(inner)).Methods(http.MethodGet)

	req := httptest.NewRequest(http.MethodGet, "/github.com/test/mod/@v/v1.0.0.info", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestNewValidationMiddleware_Forbidden(t *testing.T) {
	validatorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("module not allowed"))
	}))
	defer validatorServer.Close()

	mw := NewValidationMiddleware(validatorServer.Client(), validatorServer.URL)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := mux.NewRouter()
	r.Handle("/{module:.+}/@v/{version}.info", mw(inner)).Methods(http.MethodGet)

	req := httptest.NewRequest(http.MethodGet, "/github.com/test/mod/@v/v1.0.0.info", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestNewValidationMiddleware_ListNoVersion(t *testing.T) {
	mw := NewValidationMiddleware(&http.Client{}, "http://validator.example.com")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := mux.NewRouter()
	r.Handle("/{module:.+}/@v/list", mw(inner)).Methods(http.MethodGet)

	req := httptest.NewRequest(http.MethodGet, "/github.com/test/mod/@v/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// List doesn't have version, so validation is skipped
	require.Equal(t, http.StatusOK, w.Code)
}

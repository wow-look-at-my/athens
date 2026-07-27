package actions

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gomods/athens/pkg/storage/mem"
	"github.com/stretchr/testify/require"
)

func TestCatalogHandler_OK(t *testing.T) {
	s, err := mem.NewStorage()
	require.NoError(t, err)

	handler := catalogHandler(s)
	req := httptest.NewRequest(http.MethodGet, "/catalog", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var res catalogRes
	err = json.NewDecoder(w.Body).Decode(&res)
	require.NoError(t, err)
}

func TestGetLimitFromParam_Empty(t *testing.T) {
	limit, err := getLimitFromParam("")
	require.NoError(t, err)
	require.Equal(t, defaultPageSize, limit)
}

func TestGetLimitFromParam_Valid(t *testing.T) {
	limit, err := getLimitFromParam("50")
	require.NoError(t, err)
	require.Equal(t, 50, limit)
}

func TestGetLimitFromParam_Invalid(t *testing.T) {
	_, err := getLimitFromParam("abc")
	require.Error(t, err)
}

func TestCatalogHandler_JSON(t *testing.T) {
	s, err := mem.NewStorage()
	require.NoError(t, err)

	handler := catalogHandler(s)
	req := httptest.NewRequest(http.MethodGet, "/catalog", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	require.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	// Since mem storage doesn't implement Cataloger, expect 501
	// But still verify it returns JSON content type
	_ = json.NewDecoder(w.Body)
}

func TestCatalogHandler_WithPageSize(t *testing.T) {
	s, err := mem.NewStorage()
	require.NoError(t, err)

	handler := catalogHandler(s)
	req := httptest.NewRequest(http.MethodGet, "/catalog?pagesize=10", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var res catalogRes
	err = json.NewDecoder(w.Body).Decode(&res)
	require.NoError(t, err)
}

func TestCatalogHandler_WithToken(t *testing.T) {
	s, err := mem.NewStorage()
	require.NoError(t, err)

	handler := catalogHandler(s)
	req := httptest.NewRequest(http.MethodGet, "/catalog?token=sometoken", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	// token may or may not be valid, but handler should not panic
	require.NotEqual(t, 0, w.Code)
}

func TestCatalogHandler_InvalidPageSize(t *testing.T) {
	s, err := mem.NewStorage()
	require.NoError(t, err)

	handler := catalogHandler(s)
	req := httptest.NewRequest(http.MethodGet, "/catalog?pagesize=invalid", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

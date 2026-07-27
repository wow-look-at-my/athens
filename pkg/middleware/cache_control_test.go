package middleware

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCacheControl(t *testing.T) {
	h := func(w http.ResponseWriter, r *http.Request) {}
	expected := "private, no-store"
	ch := CacheControl(expected)
	handler := ch(http.HandlerFunc(h))

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/test", nil)
	handler.ServeHTTP(w, r)

	given := w.Result().Header.Get("Cache-Control")
	require.Equal(t, expected, given)

}

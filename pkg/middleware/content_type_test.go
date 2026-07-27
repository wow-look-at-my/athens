package middleware

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContentType(t *testing.T) {
	h := func(w http.ResponseWriter, r *http.Request) {}
	expected := "application/json"
	handler := ContentType(http.HandlerFunc(h))

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/test", nil)
	handler.ServeHTTP(w, r)

	given := w.Result().Header.Get("Content-Type")
	require.Equal(t, expected, given)

}

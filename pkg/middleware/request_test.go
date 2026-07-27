package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gomods/athens/pkg/log"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestRequestLogger(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	l := log.New("none", logrus.DebugLevel, "plain")
	handler := LogEntryMiddleware(l)(RequestLogger(inner))
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestFmtResponseCode(t *testing.T) {
	tests := []struct {
		code int
	}{
		{0},   // default to 200
		{200}, // success
		{301}, // redirect
		{400}, // client error
		{404}, // not found
		{500}, // server error
	}

	for _, tc := range tests {
		result := fmtResponseCode(tc.code)
		require.NotEmpty(t, result)
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w, statusCode: 0}
	rw.WriteHeader(http.StatusNotFound)
	require.Equal(t, http.StatusNotFound, rw.statusCode)
	require.Equal(t, http.StatusNotFound, w.Code)
}

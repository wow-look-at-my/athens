package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gomods/athens/pkg/requestid"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWithRequestID(t *testing.T) {
	var givenRequestID string
	h := WithRequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		givenRequestID = requestid.FromContext(r.Context())
	}))
	req := httptest.NewRequest("GET", "/", nil)
	expectedRequestID := uuid.New().String()
	req.Header.Set(requestid.HeaderKey, expectedRequestID)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	require.Equal(t, expectedRequestID, givenRequestID)

	req = httptest.NewRequest("GET", "/", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	require.NotEqual(t, "", givenRequestID)

}

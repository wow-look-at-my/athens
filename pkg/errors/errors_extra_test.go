package errors

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestIs(t *testing.T) {
	err := E("op", "msg", KindNotFound)
	require.True(t, Is(err, KindNotFound))
	require.False(t, Is(err, KindBadRequest))
	require.False(t, Is(nil, KindNotFound))
}

func TestIsErr(t *testing.T) {
	target := fmt.Errorf("target")
	wrapped := fmt.Errorf("wrapped: %w", target)
	require.True(t, IsErr(wrapped, target))
	require.False(t, IsErr(wrapped, fmt.Errorf("other")))
}

func TestAsErr(t *testing.T) {
	inner := E("op", "msg")
	var athensErr Error
	require.True(t, AsErr(inner, &athensErr))
	require.Equal(t, Op("op"), athensErr.Op)

	require.False(t, AsErr(fmt.Errorf("regular"), &athensErr))
}

func TestE_NoArgs(t *testing.T) {
	err := E("op")
	require.Contains(t, err.Error(), "errors.E called with 0 args")
}

func TestE_WithModuleAndVersion(t *testing.T) {
	err := E("op", M("github.com/test/mod"), V("v1.0.0"), "error msg")
	var athensErr Error
	require.True(t, errors.As(err, &athensErr))
	require.Equal(t, M("github.com/test/mod"), athensErr.Module)
	require.Equal(t, V("v1.0.0"), athensErr.Version)
}

func TestE_KindOnly(t *testing.T) {
	err := E("op", KindNotFound)
	require.Equal(t, http.StatusText(http.StatusNotFound), err.Error())
}

func TestKind_NonAthensError(t *testing.T) {
	err := fmt.Errorf("regular error")
	require.Equal(t, KindUnexpected, Kind(err))
}

func TestKind_Nested(t *testing.T) {
	inner := E("inner", "msg", KindBadRequest)
	outer := E("outer", inner)
	require.Equal(t, KindBadRequest, Kind(outer))
}

func TestKindText_NotFound(t *testing.T) {
	err := E("op", "msg", KindNotFound)
	require.Equal(t, "Not Found", KindText(err))
}

func TestSeverity_NonAthensError(t *testing.T) {
	err := fmt.Errorf("regular")
	require.Equal(t, logrus.ErrorLevel, Severity(err))
}

func TestError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("inner")
	err := E("op", inner)
	require.True(t, errors.Is(err, inner))
}

func TestKindConstants(t *testing.T) {
	require.Equal(t, http.StatusNotFound, KindNotFound)
	require.Equal(t, http.StatusBadRequest, KindBadRequest)
	require.Equal(t, http.StatusInternalServerError, KindUnexpected)
	require.Equal(t, http.StatusConflict, KindAlreadyExists)
	require.Equal(t, http.StatusTooManyRequests, KindRateLimit)
	require.Equal(t, http.StatusNotImplemented, KindNotImplemented)
	require.Equal(t, http.StatusMovedPermanently, KindRedirect)
	require.Equal(t, http.StatusGatewayTimeout, KindGatewayTimeout)
}

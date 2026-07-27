package fs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTokenFromModVer(t *testing.T) {
	token := tokenFromModVer("github.com/gomods/athens", "v1.0.0")
	require.Equal(t, "github.com/gomods/athens|v1.0.0", token)
}

func TestModVerFromToken_Valid(t *testing.T) {
	mod, ver, err := modVerFromToken("github.com/gomods/athens|v1.0.0")
	require.NoError(t, err)
	require.Equal(t, "github.com/gomods/athens", mod)
	require.Equal(t, "v1.0.0", ver)
}

func TestModVerFromToken_Empty(t *testing.T) {
	mod, ver, err := modVerFromToken("")
	require.NoError(t, err)
	require.Equal(t, "", mod)
	require.Equal(t, "", ver)
}

func TestModVerFromToken_Invalid(t *testing.T) {
	_, _, err := modVerFromToken("invalid-no-separator")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Invalid token")
}

func TestTokenRoundTrip(t *testing.T) {
	mod := "github.com/gomods/athens"
	ver := "v1.2.3"
	token := tokenFromModVer(mod, ver)
	gotMod, gotVer, err := modVerFromToken(token)
	require.NoError(t, err)
	require.Equal(t, mod, gotMod)
	require.Equal(t, ver, gotVer)
}

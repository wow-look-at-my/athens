package actions

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitializeAuthFile_Empty(t *testing.T) {
	err := initializeAuthFile("")
	require.NoError(t, err)
}

func TestInitializeAuthFile_NotFound(t *testing.T) {
	err := initializeAuthFile("/nonexistent/file")
	require.Error(t, err)
	require.Contains(t, err.Error(), "reading")
}

func TestInitializeAuthFile_Valid(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, ".netrc")
	err := os.WriteFile(src, []byte("machine example.com login user"), 0o600)
	require.NoError(t, err)

	err = initializeAuthFile(src)
	require.NoError(t, err)
}

func TestTransformAuthFileName_Netrc(t *testing.T) {
	result := transformAuthFileName(".netrc")
	expected := ".netrc"
	if runtime.GOOS == "windows" {
		expected = "_netrc"
	}
	require.Equal(t, expected, result)
}

func TestTransformAuthFileName_UnderscoreNetrc(t *testing.T) {
	result := transformAuthFileName("_netrc")
	expected := ".netrc"
	if runtime.GOOS == "windows" {
		expected = "_netrc"
	}
	require.Equal(t, expected, result)
}

func TestTransformAuthFileName_Other(t *testing.T) {
	result := transformAuthFileName(".hgrc")
	require.Equal(t, ".hgrc", result)
}

func TestGetNETRCFilename(t *testing.T) {
	result := getNETRCFilename()
	if runtime.GOOS == "windows" {
		require.Equal(t, "_netrc", result)
	} else {
		require.Equal(t, ".netrc", result)
	}
}

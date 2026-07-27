package mode

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewFile_ValidModes(t *testing.T) {
	modes := []Mode{Sync, Async, Redirect, AsyncRedirect, None}
	for _, m := range modes {
		t.Run(string(m), func(t *testing.T) {
			df, err := NewFile(m, "https://proxy.golang.org")
			require.NoError(t, err)
			require.Equal(t, m, df.Mode)
			require.Equal(t, "https://proxy.golang.org", df.DownloadURL)
		})
	}
}

const validHCL = `
mode = "sync"
downloadURL = "https://proxy.golang.org"

download "github.com/gomods/*" {
  mode = "none"
}
`

func TestNewFile_FilePrefix(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "download.hcl")
	err := os.WriteFile(fp, []byte(validHCL), 0o644)
	require.NoError(t, err)

	df, err := NewFile(Mode("file:"+fp), "")
	require.NoError(t, err)
	require.Equal(t, Sync, df.Mode)
	require.Equal(t, "https://proxy.golang.org", df.DownloadURL)
	require.Len(t, df.Paths, 1)
	require.Equal(t, "github.com/gomods/*", df.Paths[0].Pattern)
	require.Equal(t, None, df.Paths[0].Mode)
}

func TestNewFile_FilePrefix_NotFound(t *testing.T) {
	_, err := NewFile(Mode("file:/nonexistent/path.hcl"), "")
	require.Error(t, err)
}

func TestNewFile_CustomPrefix(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte(validHCL))
	df, err := NewFile(Mode("custom:"+encoded), "")
	require.NoError(t, err)
	require.Equal(t, Sync, df.Mode)
	require.Len(t, df.Paths, 1)
}

func TestNewFile_CustomPrefix_InvalidBase64(t *testing.T) {
	_, err := NewFile(Mode("custom:not-valid-base64!!!"), "")
	require.Error(t, err)
}

func TestNewFile_CustomPrefix_InvalidHCL(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("this is { not valid hcl"))
	_, err := NewFile(Mode("custom:"+encoded), "")
	require.Error(t, err)
}

func TestParseFile_InvalidMode(t *testing.T) {
	hcl := `
mode = "sync"
downloadURL = "https://proxy.golang.org"
download "github.com/bad/*" {
  mode = "invalid_mode"
}
`
	_, err := parseFile([]byte(hcl))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unrecognized mode")
}

func TestURL_PatternMatchEmptyURL(t *testing.T) {
	df := &DownloadFile{
		Mode:        Sync,
		DownloadURL: "https://default.proxy",
		Paths: []*DownloadPath{
			{Pattern: "github.com/gomods/*", Mode: Redirect, DownloadURL: ""},
		},
	}
	// Pattern matches but has empty DownloadURL, should fall through to default
	url := df.URL("github.com/gomods/athens")
	require.Equal(t, "https://default.proxy", url)
}

package fs

import (
	"bytes"
	"context"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestCatalog_Empty(t *testing.T) {
	s := getStorage(t, afero.NewMemMapFs())
	defer s.Clear()

	res, token, err := s.Catalog(context.Background(), "", 10)
	require.NoError(t, err)
	require.Empty(t, res)
	require.Empty(t, token)
}

func TestCatalog_WithModules(t *testing.T) {
	s := getStorage(t, afero.NewMemMapFs())
	defer s.Clear()

	ctx := context.Background()
	// Save two modules
	err := s.Save(ctx, "github.com/test/mod1", "v1.0.0", []byte("mod1"), bytes.NewReader([]byte("zip1")), nil, []byte(`{"Version":"v1.0.0"}`))
	require.NoError(t, err)
	err = s.Save(ctx, "github.com/test/mod2", "v2.0.0", []byte("mod2"), bytes.NewReader([]byte("zip2")), nil, []byte(`{"Version":"v2.0.0"}`))
	require.NoError(t, err)

	res, token, err := s.Catalog(ctx, "", 10)
	require.NoError(t, err)
	require.Len(t, res, 2)
	require.Empty(t, token) // less than pageSize, no token
}

func TestCatalog_Pagination(t *testing.T) {
	s := getStorage(t, afero.NewMemMapFs())
	defer s.Clear()

	ctx := context.Background()
	// Save 3 modules with ascending versions so token filtering works correctly
	versions := []string{"v1.0.0", "v2.0.0", "v3.0.0"}
	for i, name := range []string{"a", "b", "c"} {
		err := s.Save(ctx, "github.com/test/"+name, versions[i], []byte("mod"), bytes.NewReader([]byte("zip")), nil, []byte(`{"Version":"`+versions[i]+`"}`))
		require.NoError(t, err)
	}

	// Get first page of 2
	res, token, err := s.Catalog(ctx, "", 2)
	require.NoError(t, err)
	require.Len(t, res, 2)
	require.NotEmpty(t, token)

	// Get second page
	res2, token2, err := s.Catalog(ctx, token, 2)
	require.NoError(t, err)
	require.Len(t, res2, 1)
	require.Empty(t, token2)
}

func TestCatalog_InvalidToken(t *testing.T) {
	s := getStorage(t, afero.NewMemMapFs())
	defer s.Clear()

	_, _, err := s.Catalog(context.Background(), "invalid-no-separator", 10)
	require.Error(t, err)
}

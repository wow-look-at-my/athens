package download

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetRedirectURL(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		path     string
		expected string
		hasErr   bool
	}{
		{
			name:     "simple",
			base:     "https://proxy.golang.org",
			path:     "/github.com/mod/@v/v1.0.0.info",
			expected: "https://proxy.golang.org/github.com/mod/@v/v1.0.0.info",
		},
		{
			name:     "with existing path",
			base:     "https://proxy.golang.org/prefix",
			path:     "/mod/@v/v1.0.0.zip",
			expected: "https://proxy.golang.org/prefix/mod/@v/v1.0.0.zip",
		},
		{
			name:   "invalid url",
			base:   "://invalid",
			path:   "/test",
			hasErr: true,
		},
		{
			name:     "empty path",
			base:     "https://proxy.golang.org",
			path:     "",
			expected: "https://proxy.golang.org",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := getRedirectURL(tc.base, tc.path)
			if tc.hasErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestRemovePseudoVersions(t *testing.T) {
	versions := []string{
		"v1.0.0",
		"v1.1.0",
		"v0.0.0-20200101120000-abcdef123456",
		"v1.2.0",
		"v2.0.0-20210615123456-deadbeef0000+incompatible",
	}
	result := removePseudoVersions(versions)
	require.Equal(t, []string{"v1.0.0", "v1.1.0", "v1.2.0"}, result)
}

func TestRemovePseudoVersions_Empty(t *testing.T) {
	result := removePseudoVersions(nil)
	require.Nil(t, result)
}

func TestRemovePseudoVersions_NoPseudo(t *testing.T) {
	versions := []string{"v1.0.0", "v2.0.0"}
	result := removePseudoVersions(versions)
	require.Equal(t, versions, result)
}

func TestUnion(t *testing.T) {
	tests := []struct {
		name     string
		list1    []string
		list2    []string
		expected []string
	}{
		{
			name:     "no duplicates",
			list1:    []string{"a", "b"},
			list2:    []string{"c", "d"},
			expected: []string{"a", "b", "c", "d"},
		},
		{
			name:     "with duplicates",
			list1:    []string{"a", "b", "c"},
			list2:    []string{"b", "c", "d"},
			expected: []string{"a", "b", "c", "d"},
		},
		{
			name:     "nil lists",
			list1:    nil,
			list2:    nil,
			expected: []string{},
		},
		{
			name:     "one nil list",
			list1:    []string{"a"},
			list2:    nil,
			expected: []string{"a"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := union(tc.list1, tc.list2)
			require.Equal(t, tc.expected, result)
		})
	}
}

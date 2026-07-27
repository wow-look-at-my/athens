package paths

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestGetModule(t *testing.T) {
	tests := []struct {
		name     string
		vars     map[string]string
		expected string
		hasErr   bool
	}{
		{
			name:     "valid module",
			vars:     map[string]string{"module": "github.com/gomods/athens"},
			expected: "github.com/gomods/athens",
		},
		{
			name:   "empty module",
			vars:   map[string]string{},
			hasErr: true,
		},
		{
			name:   "invalid encoding",
			vars:   map[string]string{"module": "github.com/Azure"},
			hasErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r = mux.SetURLVars(r, tc.vars)

			mod, err := GetModule(r)
			if tc.hasErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, mod)
			}
		})
	}
}

func TestGetVersion(t *testing.T) {
	tests := []struct {
		name     string
		vars     map[string]string
		expected string
		hasErr   bool
	}{
		{
			name:     "valid version",
			vars:     map[string]string{"version": "v1.0.0"},
			expected: "v1.0.0",
		},
		{
			name:   "empty version",
			vars:   map[string]string{},
			hasErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r = mux.SetURLVars(r, tc.vars)

			ver, err := GetVersion(r)
			if tc.hasErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, ver)
			}
		})
	}
}

func TestGetAllParams(t *testing.T) {
	tests := []struct {
		name    string
		vars    map[string]string
		hasErr  bool
		module  string
		version string
	}{
		{
			name:    "valid params",
			vars:    map[string]string{"module": "github.com/gomods/athens", "version": "v1.0.0"},
			module:  "github.com/gomods/athens",
			version: "v1.0.0",
		},
		{
			name:   "missing module",
			vars:   map[string]string{"version": "v1.0.0"},
			hasErr: true,
		},
		{
			name:   "missing version",
			vars:   map[string]string{"module": "github.com/gomods/athens"},
			hasErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r = mux.SetURLVars(r, tc.vars)

			params, err := GetAllParams(r)
			if tc.hasErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.module, params.Module)
				require.Equal(t, tc.version, params.Version)
			}
		})
	}
}

package paths

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasErr   bool
	}{
		{
			name:     "simple lowercase path",
			input:    "github.com/gomods/athens",
			expected: "github.com/gomods/athens",
		},
		{
			name:     "uppercase encoding with bang",
			input:    "github.com/!my!repo",
			expected: "github.com/MyRepo",
		},
		{
			name:     "single uppercase letter",
			input:    "github.com/!azure-sdk",
			expected: "github.com/Azure-sdk",
		},
		{
			name:   "invalid bang at end",
			input:  "github.com/test!",
			hasErr: true,
		},
		{
			name:   "bang followed by non-lowercase",
			input:  "github.com/!1test",
			hasErr: true,
		},
		{
			name:   "raw uppercase letter (invalid)",
			input:  "github.com/Azure",
			hasErr: true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "path with numbers and dashes",
			input:    "github.com/go-mod-123/pkg",
			expected: "github.com/go-mod-123/pkg",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := DecodePath(tc.input)
			if tc.hasErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestDecodeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		ok       bool
	}{
		{
			name:     "simple",
			input:    "hello",
			expected: "hello",
			ok:       true,
		},
		{
			name:     "with bang",
			input:    "!hello",
			expected: "Hello",
			ok:       true,
		},
		{
			name:  "non-ascii",
			input: "héllo",
			ok:    false,
		},
		{
			name:  "trailing bang",
			input: "hello!",
			ok:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, ok := decodeString(tc.input)
			require.Equal(t, tc.ok, ok)
			if ok {
				require.Equal(t, tc.expected, result)
			}
		})
	}
}

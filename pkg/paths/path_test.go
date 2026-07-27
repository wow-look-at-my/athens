package paths

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestMatchesPattern(t *testing.T) {
	type args struct {
		pattern string
		name    string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "standard match",
			args: args{
				pattern: "example.com/*",
				name:    "example.com/athens",
			},
			want: true,
		},
		{
			name: "mutiple depth match",
			args: args{
				pattern: "example.com/*",
				name:    "example.com/athens/pkg",
			},
			want: true,
		},
		{
			name: "subdomain match",
			args: args{
				pattern: "*.example.com/*",
				name:    "go.example.com/athens/pkg",
			},
			want: true,
		},
		{
			name: "subdirectory exact match",
			args: args{
				pattern: "*.example.com/mod",
				name:    "go.example.com/mod/example",
			},
			want: true,
		},
		{
			name: "subdirectory mismatch",
			args: args{
				pattern: "*.example.com/mod",
				name:    "go.example.com/pkg/example",
			},
			want: false,
		},
		{
			name: "shorter name mismatch",
			args: args{
				pattern: "*.example.com/mod/pkg",
				name:    "go.example.com/pkg",
			},
			want: false,
		},
		{
			name: "no subdirectory mismatch",
			args: args{
				pattern: "*.example.com/mod/pkg",
				name:    "go.example.com/pkg",
			},
			want: false,
		},
		{
			name: "bad pattern",
			args: args{
				pattern: "[]a]",
				name:    "go.example.com/pkg",
			},
			want: false,
		},
		{
			name: "matches everything",
			args: args{
				pattern: "*",
				name:    "github.com/gomods/athen",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesPattern(tt.args.pattern, tt.args.name)
			assert.Equal(t, tt.want, got)

		})
	}
}

func BenchmarkMatchesPattern(b *testing.B) {
	for i := 1; i < 5; i++ {
		target := "git.example.com" + strings.Repeat("/path", i) + "/pkg"
		b.Run(fmt.Sprintf("MatchPattern/%d", i), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				assert.True(b, MatchesPattern("*.example.com/*", target))

			}
		})
	}
}

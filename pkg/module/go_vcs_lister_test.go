package module

import (
	"context"
	"testing"
	"time"

	"github.com/gomods/athens/pkg/errors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVCSListerInvalidModulePaths(t *testing.T) {
	t.Parallel()
	lister := NewVCSLister("go", nil, afero.NewMemMapFs(), 30*time.Second)

	tests := []struct {
		name string
		mod  string
	}{
		{"bare host", "github.com"},
		{"host with owner only", "github.com/owner"},
		{"empty string", ""},
		{"single element", "foo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := lister.List(context.Background(), tt.mod)
			require.NotNil(t, err)

			kind := errors.Kind(err)
			assert.Equal(t, errors.KindNotFound, kind)

		})
	}
}

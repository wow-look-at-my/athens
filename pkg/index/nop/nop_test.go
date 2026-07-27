package nop

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	idx := New()
	require.NotNil(t, idx)

	err := idx.Index(context.Background(), "github.com/test/mod", "v1.0.0")
	require.NoError(t, err)

	lines, err := idx.Lines(context.Background(), time.Time{}, 100)
	require.NoError(t, err)
	require.Empty(t, lines)
}

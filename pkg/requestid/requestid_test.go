package requestid

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetAndFromContext(t *testing.T) {
	ctx := context.Background()
	ctx = SetInContext(ctx, "test-id-123")

	id := FromContext(ctx)
	require.Equal(t, "test-id-123", id)
}

func TestFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	id := FromContext(ctx)
	require.Equal(t, "", id)
}

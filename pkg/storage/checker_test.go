package storage

import (
	"context"
	"testing"

	"github.com/gomods/athens/pkg/errors"
	"github.com/stretchr/testify/require"
)

type mockBackend struct {
	Backend
	infoErr error
}

func (m *mockBackend) Info(ctx context.Context, module, version string) ([]byte, error) {
	if m.infoErr != nil {
		return nil, m.infoErr
	}
	return []byte(`{"Version":"v1.0.0"}`), nil
}

func TestWithChecker_Exists(t *testing.T) {
	mb := &mockBackend{}
	c := WithChecker(mb)

	exists, err := c.Exists(context.Background(), "github.com/test/mod", "v1.0.0")
	require.NoError(t, err)
	require.True(t, exists)
}

func TestWithChecker_NotFound(t *testing.T) {
	mb := &mockBackend{infoErr: errors.E("test", errors.KindNotFound)}
	c := WithChecker(mb)

	exists, err := c.Exists(context.Background(), "github.com/test/mod", "v1.0.0")
	require.NoError(t, err)
	require.False(t, exists)
}

func TestWithChecker_Error(t *testing.T) {
	mb := &mockBackend{infoErr: errors.E("test", "some error", errors.KindUnexpected)}
	c := WithChecker(mb)

	_, err := c.Exists(context.Background(), "github.com/test/mod", "v1.0.0")
	require.Error(t, err)
}

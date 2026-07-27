package log

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestSetEntryInContext(t *testing.T) {
	ctx := context.Background()
	lggr := New("none", logrus.DebugLevel, "plain")
	e := lggr.WithFields(map[string]any{"test": "value"})

	newCtx := SetEntryInContext(ctx, e)
	require.NotNil(t, newCtx)

	retrieved := EntryFromContext(newCtx)
	require.NotNil(t, retrieved)
}

func TestEntryFromContext_NoEntry(t *testing.T) {
	ctx := context.Background()
	e := EntryFromContext(ctx)
	require.NotNil(t, e)
	// Should return NoOpLogger
}

func TestEntryFromContext_NilValue(t *testing.T) {
	ctx := context.WithValue(context.Background(), logEntryKey, nil)
	e := EntryFromContext(ctx)
	require.NotNil(t, e)
}

func TestEntryFromContext_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), logEntryKey, "not an entry")
	e := EntryFromContext(ctx)
	require.NotNil(t, e)
}

//go:build unix

package shutdown

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestChildProcReaper(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	done := ChildProcReaper(ctx, logger)
	require.NotNil(t, done)

	// Cancel parent context to stop the reaper
	cancel()

	// Wait for the reaper to finish
	<-done.Done()
}

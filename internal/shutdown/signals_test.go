//go:build unix

package shutdown

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetSignals(t *testing.T) {
	signals := GetSignals()
	require.Len(t, signals, 2)
}

package build

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestString(t *testing.T) {
	s := String()
	require.Contains(t, s, "Build Details:")
	require.Contains(t, s, "Version:")
	require.Contains(t, s, "Date:")
}

func TestData(t *testing.T) {
	d := Data()
	require.IsType(t, Details{}, d)
}

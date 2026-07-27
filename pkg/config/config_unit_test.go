package config

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"
)

func TestValidateStorage_Memory(t *testing.T) {
	v := validator.New()
	require.NoError(t, validateStorage(v, "memory", nil))
}

func TestValidateStorage_Unknown(t *testing.T) {
	v := validator.New()
	require.Error(t, validateStorage(v, "unknown_type", nil))
}

func TestValidateIndex_Empty(t *testing.T) {
	v := validator.New()
	require.NoError(t, validateIndex(v, "", nil))
}

func TestValidateIndex_None(t *testing.T) {
	v := validator.New()
	require.NoError(t, validateIndex(v, "none", nil))
}

func TestValidateIndex_Memory(t *testing.T) {
	v := validator.New()
	require.NoError(t, validateIndex(v, "memory", nil))
}

func TestValidateIndex_Unknown(t *testing.T) {
	v := validator.New()
	require.Error(t, validateIndex(v, "unknown_type", nil))
}

func TestValidateConfig_Default(t *testing.T) {
	cfg := defaultConfig()
	err := validateConfig(*cfg)
	require.NoError(t, err)
}

func TestValidateStorage_Disk(t *testing.T) {
	v := validator.New()
	disk := &Storage{Disk: &DiskConfig{RootPath: "/tmp/test"}}
	require.NoError(t, validateStorage(v, "disk", disk))
}

func TestValidateStorage_External(t *testing.T) {
	v := validator.New()
	ext := &Storage{External: &External{URL: "http://example.com"}}
	require.NoError(t, validateStorage(v, "external", ext))
}

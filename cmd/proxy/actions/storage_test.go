package actions

import (
	"net/http"
	"testing"
	"time"

	"github.com/gomods/athens/pkg/config"
	"github.com/gomods/athens/pkg/storage"
	"github.com/gomods/athens/pkg/storage/mem"
	"github.com/stretchr/testify/require"
)

func TestGetStorage_Memory(t *testing.T) {
	s, err := GetStorage("memory", nil, time.Second, nil)
	require.NoError(t, err)
	require.NotNil(t, s)
}

func TestGetStorage_Disk(t *testing.T) {
	dir := t.TempDir()
	sc := &config.Storage{
		Disk: &config.DiskConfig{RootPath: dir},
	}
	s, err := GetStorage("disk", sc, time.Second, nil)
	require.NoError(t, err)
	require.NotNil(t, s)
}

func TestGetStorage_Disk_NilConfig(t *testing.T) {
	_, err := GetStorage("disk", &config.Storage{}, time.Second, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Invalid Disk Storage Configuration")
}

func TestGetStorage_External(t *testing.T) {
	sc := &config.Storage{
		External: &config.External{URL: "http://localhost:8080"},
	}
	s, err := GetStorage("external", sc, time.Second, &http.Client{})
	require.NoError(t, err)
	require.NotNil(t, s)
}

func TestGetStorage_External_NilConfig(t *testing.T) {
	_, err := GetStorage("external", &config.Storage{}, time.Second, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Invalid External Storage Configuration")
}

func TestGetStorage_Unknown(t *testing.T) {
	_, err := GetStorage("unknown_type", nil, time.Second, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown")
}

func TestRegisterStorage_Custom(t *testing.T) {
	RegisterStorage("test_backend", func(sc *config.Storage, timeout time.Duration, client *http.Client) (storage.Backend, error) {
		return mem.NewStorage()
	})

	s, err := GetStorage("test_backend", &config.Storage{}, time.Second, nil)
	require.NoError(t, err)
	require.NotNil(t, s)

	// Clean up
	delete(storageRegistry, "test_backend")
}

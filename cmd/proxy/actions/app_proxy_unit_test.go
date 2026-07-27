package actions

import (
	"context"
	"testing"

	"github.com/gomods/athens/pkg/config"
	"github.com/gomods/athens/pkg/index"
	"github.com/gomods/athens/pkg/index/mem"
	"github.com/gomods/athens/pkg/log"
	"github.com/gomods/athens/pkg/stash"
	"github.com/gomods/athens/pkg/storage"
	storagemem "github.com/gomods/athens/pkg/storage/mem"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestGetSingleFlight_Default(t *testing.T) {
	c := &config.Config{}
	s, _ := storagemem.NewStorage()
	checker := storage.WithChecker(s)
	l := log.New("none", logrus.DebugLevel, "plain")

	wrapper, err := getSingleFlight(l, c, s, checker)
	require.NoError(t, err)
	require.NotNil(t, wrapper)
}

func TestGetSingleFlight_Memory(t *testing.T) {
	c := &config.Config{SingleFlightType: "memory"}
	s, _ := storagemem.NewStorage()
	checker := storage.WithChecker(s)
	l := log.New("none", logrus.DebugLevel, "plain")

	wrapper, err := getSingleFlight(l, c, s, checker)
	require.NoError(t, err)
	require.NotNil(t, wrapper)
}

func TestGetSingleFlight_Etcd_NilConfig(t *testing.T) {
	c := &config.Config{SingleFlightType: "etcd"}
	s, _ := storagemem.NewStorage()
	checker := storage.WithChecker(s)
	l := log.New("none", logrus.DebugLevel, "plain")

	_, err := getSingleFlight(l, c, s, checker)
	require.Error(t, err)
	require.Contains(t, err.Error(), "etcd config must be present")
}

func TestGetSingleFlight_Redis_NilConfig(t *testing.T) {
	c := &config.Config{SingleFlightType: "redis"}
	s, _ := storagemem.NewStorage()
	checker := storage.WithChecker(s)
	l := log.New("none", logrus.DebugLevel, "plain")

	_, err := getSingleFlight(l, c, s, checker)
	require.Error(t, err)
	require.Contains(t, err.Error(), "redis config must be present")
}

func TestGetSingleFlight_RedisSentinel_NilConfig(t *testing.T) {
	c := &config.Config{SingleFlightType: "redis-sentinel"}
	s, _ := storagemem.NewStorage()
	checker := storage.WithChecker(s)
	l := log.New("none", logrus.DebugLevel, "plain")

	_, err := getSingleFlight(l, c, s, checker)
	require.Error(t, err)
	require.Contains(t, err.Error(), "redis config must be present")
}

func TestGetSingleFlight_Unknown(t *testing.T) {
	c := &config.Config{SingleFlightType: "unknown"}
	s, _ := storagemem.NewStorage()
	checker := storage.WithChecker(s)
	l := log.New("none", logrus.DebugLevel, "plain")

	_, err := getSingleFlight(l, c, s, checker)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown")
}

func TestGetSingleFlight_Registry(t *testing.T) {
	RegisterSingleFlight("test_sf", func(l *log.Logger, c *config.Config, s storage.Backend, checker storage.Checker) (stash.Wrapper, error) {
		return stash.WithSingleflight, nil
	})
	defer delete(singleFlightRegistry, "test_sf")

	c := &config.Config{SingleFlightType: "test_sf"}
	s, _ := storagemem.NewStorage()
	checker := storage.WithChecker(s)
	l := log.New("none", logrus.DebugLevel, "plain")

	wrapper, err := getSingleFlight(l, c, s, checker)
	require.NoError(t, err)
	require.NotNil(t, wrapper)
}

func TestGetIndex_Default(t *testing.T) {
	c := &config.Config{}
	idx, err := getIndex(c)
	require.NoError(t, err)
	require.NotNil(t, idx)
}

func TestGetIndex_None(t *testing.T) {
	c := &config.Config{IndexType: "none"}
	idx, err := getIndex(c)
	require.NoError(t, err)
	require.NotNil(t, idx)
}

func TestGetIndex_Memory(t *testing.T) {
	c := &config.Config{IndexType: "memory"}
	idx, err := getIndex(c)
	require.NoError(t, err)
	require.NotNil(t, idx)
}

func TestGetIndex_Unknown(t *testing.T) {
	c := &config.Config{IndexType: "unknown_index"}
	_, err := getIndex(c)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown")
}

func TestGetIndex_Registry(t *testing.T) {
	RegisterIndex("test_idx", func(c *config.Config) (index.Indexer, error) {
		return mem.New(), nil
	})
	defer delete(indexRegistry, "test_idx")

	c := &config.Config{IndexType: "test_idx"}
	idx, err := getIndex(c)
	require.NoError(t, err)
	require.NotNil(t, idx)
}

func TestAthensLoggerForRedis_Printf(t *testing.T) {
	l := log.New("none", logrus.DebugLevel, "plain")
	rl := &athensLoggerForRedis{logger: l}
	// Just ensure it doesn't panic
	rl.Printf(context.Background(), "test %s", "message")
}

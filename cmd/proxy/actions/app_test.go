package actions

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gomods/athens/pkg/config"
	"github.com/gomods/athens/pkg/log"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestApp_Minimal(t *testing.T) {
	logger := log.New("none", logrus.DebugLevel, "plain")
	conf := &config.Config{
		GoBinary:         "go",
		GoBinaryEnvVars:  config.EnvList{"GOPROXY=direct"},
		GoEnv:            "development",
		GoGetWorkers:     10,
		ProtocolWorkers:  30,
		LogLevel:         "debug",
		LogFormat:        "plain",
		CloudRuntime:     "none",
		StorageType:      "memory",
		Port:             ":3000",
		SingleFlightType: "memory",
		TimeoutConf:      config.TimeoutConf{Timeout: 300},
		SumDBs:           []string{"https://sum.golang.org"},
		NoSumPatterns:    []string{},
		DownloadMode:     "sync",
		NetworkMode:      "strict",
		RobotsFile:       "robots.txt",
		IndexType:        "none",
		SingleFlight: &config.SingleFlight{
			Etcd:          &config.Etcd{Endpoints: "localhost:2379"},
			Redis:         &config.Redis{Endpoint: "127.0.0.1:6379", Password: "", LockConfig: config.DefaultRedisLockConfig()},
			RedisSentinel: &config.RedisSentinel{Endpoints: []string{"127.0.0.1:26379"}, MasterName: "redis-1", LockConfig: config.DefaultRedisLockConfig()},
			GCP:           config.DefaultGCPConfig(),
		},
		Storage: &config.Storage{},
		Index:   &config.Index{},
	}

	handler, err := App(logger, conf)
	require.NoError(t, err)
	require.NotNil(t, handler)

	// Verify health endpoint works
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestApp_WithBasicAuth(t *testing.T) {
	logger := log.New("none", logrus.DebugLevel, "plain")
	conf := &config.Config{
		GoBinary:         "go",
		GoBinaryEnvVars:  config.EnvList{"GOPROXY=direct"},
		GoEnv:            "development",
		GoGetWorkers:     10,
		ProtocolWorkers:  30,
		LogLevel:         "debug",
		LogFormat:        "plain",
		CloudRuntime:     "none",
		StorageType:      "memory",
		Port:             ":3000",
		SingleFlightType: "memory",
		TimeoutConf:      config.TimeoutConf{Timeout: 300},
		SumDBs:           []string{"https://sum.golang.org"},
		NoSumPatterns:    []string{},
		DownloadMode:     "sync",
		NetworkMode:      "strict",
		RobotsFile:       "robots.txt",
		IndexType:        "none",
		BasicAuthUser:    "testuser",
		BasicAuthPass:    "testpass",
		SingleFlight: &config.SingleFlight{
			Etcd:          &config.Etcd{Endpoints: "localhost:2379"},
			Redis:         &config.Redis{Endpoint: "127.0.0.1:6379", Password: "", LockConfig: config.DefaultRedisLockConfig()},
			RedisSentinel: &config.RedisSentinel{Endpoints: []string{"127.0.0.1:26379"}, MasterName: "redis-1", LockConfig: config.DefaultRedisLockConfig()},
			GCP:           config.DefaultGCPConfig(),
		},
		Storage: &config.Storage{},
		Index:   &config.Index{},
	}

	handler, err := App(logger, conf)
	require.NoError(t, err)
	require.NotNil(t, handler)

	// healthz is excluded from basic auth
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// root without auth should be 401
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestApp_WithPathPrefix(t *testing.T) {
	logger := log.New("none", logrus.DebugLevel, "plain")
	conf := &config.Config{
		GoBinary:         "go",
		GoBinaryEnvVars:  config.EnvList{"GOPROXY=direct"},
		GoEnv:            "development",
		GoGetWorkers:     10,
		ProtocolWorkers:  30,
		LogLevel:         "debug",
		LogFormat:        "plain",
		CloudRuntime:     "none",
		StorageType:      "memory",
		Port:             ":3000",
		SingleFlightType: "memory",
		TimeoutConf:      config.TimeoutConf{Timeout: 300},
		SumDBs:           []string{"https://sum.golang.org"},
		NoSumPatterns:    []string{},
		DownloadMode:     "sync",
		NetworkMode:      "strict",
		RobotsFile:       "robots.txt",
		IndexType:        "none",
		PathPrefix:       "/proxy",
		SingleFlight: &config.SingleFlight{
			Etcd:          &config.Etcd{Endpoints: "localhost:2379"},
			Redis:         &config.Redis{Endpoint: "127.0.0.1:6379", Password: "", LockConfig: config.DefaultRedisLockConfig()},
			RedisSentinel: &config.RedisSentinel{Endpoints: []string{"127.0.0.1:26379"}, MasterName: "redis-1", LockConfig: config.DefaultRedisLockConfig()},
			GCP:           config.DefaultGCPConfig(),
		},
		Storage: &config.Storage{},
		Index:   &config.Index{},
	}

	handler, err := App(logger, conf)
	require.NoError(t, err)
	require.NotNil(t, handler)

	// Root should return 200 even with prefix
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Prefixed healthz should work
	req = httptest.NewRequest(http.MethodGet, "/proxy/healthz", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestApp_WithLocalSumDB(t *testing.T) {
	logger := log.New("none", logrus.DebugLevel, "plain")
	conf := &config.Config{
		GoBinary:         "go",
		GoBinaryEnvVars:  config.EnvList{"GOPROXY=direct"},
		GoEnv:            "development",
		GoGetWorkers:     10,
		ProtocolWorkers:  30,
		LogLevel:         "debug",
		LogFormat:        "plain",
		CloudRuntime:     "none",
		StorageType:      "memory",
		Port:             ":3000",
		SingleFlightType: "memory",
		TimeoutConf:      config.TimeoutConf{Timeout: 300},
		SumDBs:           []string{},
		NoSumPatterns:    []string{},
		DownloadMode:     "sync",
		NetworkMode:      "strict",
		RobotsFile:       "robots.txt",
		IndexType:        "memory",
		LocalSumDB:       true,
		LocalSumDBDir:    t.TempDir(),
		LocalSumDBName:   "test.local",
		SingleFlight: &config.SingleFlight{
			Etcd:          &config.Etcd{Endpoints: "localhost:2379"},
			Redis:         &config.Redis{Endpoint: "127.0.0.1:6379", Password: "", LockConfig: config.DefaultRedisLockConfig()},
			RedisSentinel: &config.RedisSentinel{Endpoints: []string{"127.0.0.1:26379"}, MasterName: "redis-1", LockConfig: config.DefaultRedisLockConfig()},
			GCP:           config.DefaultGCPConfig(),
		},
		Storage: &config.Storage{},
		Index:   &config.Index{},
	}

	handler, err := App(logger, conf)
	require.NoError(t, err)
	require.NotNil(t, handler)

	// Test sumdb supported endpoint
	req := httptest.NewRequest(http.MethodGet, "/sumdb/test.local/supported", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestApp_WithValidatorHook(t *testing.T) {
	// Set up a mock validator server
	validatorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer validatorServer.Close()

	logger := log.New("none", logrus.DebugLevel, "plain")
	conf := &config.Config{
		GoBinary:         "go",
		GoBinaryEnvVars:  config.EnvList{"GOPROXY=direct"},
		GoEnv:            "development",
		GoGetWorkers:     10,
		ProtocolWorkers:  30,
		LogLevel:         "debug",
		LogFormat:        "plain",
		CloudRuntime:     "none",
		StorageType:      "memory",
		Port:             ":3000",
		SingleFlightType: "memory",
		TimeoutConf:      config.TimeoutConf{Timeout: 300},
		SumDBs:           []string{"https://sum.golang.org"},
		NoSumPatterns:    []string{},
		DownloadMode:     "sync",
		NetworkMode:      "strict",
		RobotsFile:       "robots.txt",
		IndexType:        "none",
		ValidatorHook:    validatorServer.URL,
		SingleFlight: &config.SingleFlight{
			Etcd:          &config.Etcd{Endpoints: "localhost:2379"},
			Redis:         &config.Redis{Endpoint: "127.0.0.1:6379", Password: "", LockConfig: config.DefaultRedisLockConfig()},
			RedisSentinel: &config.RedisSentinel{Endpoints: []string{"127.0.0.1:26379"}, MasterName: "redis-1", LockConfig: config.DefaultRedisLockConfig()},
			GCP:           config.DefaultGCPConfig(),
		},
		Storage: &config.Storage{},
		Index:   &config.Index{},
	}

	handler, err := App(logger, conf)
	require.NoError(t, err)
	require.NotNil(t, handler)
}

func TestApp_GithubTokenAndNetrc_Error(t *testing.T) {
	logger := log.New("none", logrus.DebugLevel, "plain")
	conf := &config.Config{
		GithubToken: "sometoken",
		NETRCPath:   "/some/path",
	}

	_, err := App(logger, conf)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot provide both")
}

func TestApp_WithGithubToken(t *testing.T) {
	logger := log.New("none", logrus.DebugLevel, "plain")
	conf := &config.Config{
		GoBinary:         "go",
		GoBinaryEnvVars:  config.EnvList{"GOPROXY=direct"},
		GoEnv:            "development",
		GoGetWorkers:     10,
		ProtocolWorkers:  30,
		LogLevel:         "debug",
		LogFormat:        "plain",
		CloudRuntime:     "none",
		StorageType:      "memory",
		Port:             ":3000",
		SingleFlightType: "memory",
		TimeoutConf:      config.TimeoutConf{Timeout: 300},
		SumDBs:           []string{"https://sum.golang.org"},
		NoSumPatterns:    []string{},
		DownloadMode:     "sync",
		NetworkMode:      "strict",
		RobotsFile:       "robots.txt",
		IndexType:        "none",
		GithubToken:      "test-token-123",
		SingleFlight: &config.SingleFlight{
			Etcd:          &config.Etcd{Endpoints: "localhost:2379"},
			Redis:         &config.Redis{Endpoint: "127.0.0.1:6379", Password: "", LockConfig: config.DefaultRedisLockConfig()},
			RedisSentinel: &config.RedisSentinel{Endpoints: []string{"127.0.0.1:26379"}, MasterName: "redis-1", LockConfig: config.DefaultRedisLockConfig()},
			GCP:           config.DefaultGCPConfig(),
		},
		Storage: &config.Storage{},
		Index:   &config.Index{},
	}

	handler, err := App(logger, conf)
	require.NoError(t, err)
	require.NotNil(t, handler)
}

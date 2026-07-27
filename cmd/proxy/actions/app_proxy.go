package actions

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gomods/athens/pkg/config"
	"github.com/gomods/athens/pkg/download"
	"github.com/gomods/athens/pkg/download/addons"
	"github.com/gomods/athens/pkg/download/mode"
	"github.com/gomods/athens/pkg/index"
	"github.com/gomods/athens/pkg/index/mem"
	"github.com/gomods/athens/pkg/index/nop"
	"github.com/gomods/athens/pkg/log"
	"github.com/gomods/athens/pkg/module"
	"github.com/gomods/athens/pkg/stash"
	"github.com/gomods/athens/pkg/storage"
	"github.com/gomods/athens/pkg/sumlocal"
	"github.com/gorilla/mux"
	"github.com/spf13/afero"
)

func addProxyRoutes(
	r *mux.Router,
	s storage.Backend,
	l *log.Logger,
	c *config.Config,
) error {
	r.HandleFunc("/", proxyHomeHandler(c))
	r.HandleFunc("/healthz", healthHandler)
	r.HandleFunc("/readyz", getReadinessHandler(s))
	r.HandleFunc("/version", versionHandler)
	r.HandleFunc("/catalog", catalogHandler(s))
	r.HandleFunc("/robots.txt", robotsHandler(c))

	indexer, err := getIndex(c)
	if err != nil {
		return err
	}
	r.HandleFunc("/index", indexHandler(indexer))

	// Build the module fetcher and stasher up front. The stasher is the
	// component that downloads a module from upstream and saves it into
	// storage; both the download protocol (below) and the local sumdb need
	// it, so it has to exist before either is wired up.
	fs := afero.NewOsFs()

	if !c.GoBinaryEnvVars.HasKey("GONOSUMDB") {
		c.GoBinaryEnvVars.Add("GONOSUMDB", strings.Join(c.NoSumPatterns, ","))
	}
	if err := c.GoBinaryEnvVars.Validate(); err != nil {
		return err
	}
	mf, err := module.NewGoGetFetcher(c.GoBinary, c.GoGetDir, c.GoBinaryEnvVars, fs)
	if err != nil {
		return err
	}

	lister := module.NewVCSLister(c.GoBinary, c.GoBinaryEnvVars, fs, c.TimeoutDuration())
	checker := storage.WithChecker(s)
	withSingleFlight, err := getSingleFlight(l, c, s, checker)
	if err != nil {
		return err
	}
	st := stash.New(mf, s, indexer, stash.WithPool(c.GoGetWorkers), withSingleFlight)

	// Local sumdb: compute and serve module hashes locally
	// so Go clients can verify modules without contacting sum.golang.org.
	if c.LocalSumDB {
		sumdbDir := c.LocalSumDBDir
		if sumdbDir == "" {
			sumdbDir = filepath.Join(os.TempDir(), "athens-sumdb")
		}
		sumdbName := c.LocalSumDBName
		if sumdbName == "" {
			sumdbName = "athens.local"
		}
		// Pass the stasher so the sumdb can fetch a module on demand when a
		// client asks to verify one that has not been stored yet (e.g. the
		// golang.org/toolchain module Go pulls during a toolchain switch,
		// which may be downloaded via a GOPROXY "direct" fallback and thus
		// never hit Athens's module endpoints).
		localSumDB, err := sumlocal.New(sumdbDir, sumdbName, s, st)
		if err != nil {
			return fmt.Errorf("local sumdb: %w", err)
		}
		l.Infof("local sumdb enabled: name=%s verifier_key=%s", sumdbName, localSumDB.VerifierKey())

		supportPath := path.Join("/sumdb", sumdbName, "/supported")
		r.HandleFunc(supportPath, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		sumHandler := localSumDB.Handler()
		pathPrefix := "/sumdb/" + sumdbName
		r.PathPrefix(pathPrefix + "/").Handler(
			http.StripPrefix(strings.TrimSuffix(c.PathPrefix, "/")+pathPrefix, sumHandler),
		)
	}

	for _, sumdb := range c.SumDBs {
		sumdbURL, err := url.Parse(sumdb)
		if err != nil {
			return err
		}
		if sumdbURL.Scheme != "https" {
			return fmt.Errorf("sumdb: %v must have an https scheme", sumdb)
		}
		supportPath := path.Join("/sumdb", sumdbURL.Host, "/supported")
		r.HandleFunc(supportPath, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		sumHandler := sumdbProxy(sumdbURL, c.NoSumPatterns)
		pathPrefix := "/sumdb/" + sumdbURL.Host
		r.PathPrefix(pathPrefix + "/").Handler(
			http.StripPrefix(strings.TrimSuffix(c.PathPrefix, "/")+pathPrefix, sumHandler),
		)
	}

	// Download Protocol:
	// the download.Protocol and the stash.Stasher interfaces are composable
	// in a middleware fashion. Therefore you can separate concerns
	// by the functionality: a download.Protocol that just takes care
	// of "go getting" things, and another Protocol that just takes care
	// of "pooling" requests etc.

	// In our case, we'd like to compose both interfaces in a particular
	// order to ensure logical ordering of execution.

	// Here's the order of an incoming request to the download.Protocol:

	// 1. The downloadpool gets hit first, and manages concurrent requests
	// 2. The downloadpool passes the request to its parent Protocol: stasher
	// 3. The stasher Protocol checks storage first, and if storage is empty
	// it makes a Stash request to the stash.Stasher interface.

	// Once the stasher picks up an order, here's how the requests go in order:
	// 1. The singleflight picks up the first request and latches duplicate ones.
	// 2. The singleflight passes the stash to its parent: stashpool.
	// 3. The stashpool manages limiting concurrent requests and passes them to stash.
	// 4. The plain stash.New just takes a request from upstream and saves it into storage.
	df, err := mode.NewFile(c.DownloadMode, c.DownloadURL)
	if err != nil {
		return err
	}

	dpOpts := &download.Opts{
		Storage:      s,
		Stasher:      st,
		Lister:       lister,
		DownloadFile: df,
		NetworkMode:  c.NetworkMode,
	}

	dp := download.New(dpOpts, addons.WithPool(c.ProtocolWorkers))

	handlerOpts := &download.HandlerOpts{Protocol: dp, Logger: l, DownloadFile: df}
	download.RegisterHandlers(r, handlerOpts)

	return nil
}

// athensLoggerForRedis implements pkg/stash.RedisLogger.
type athensLoggerForRedis struct {
	logger *log.Logger
}

func (l *athensLoggerForRedis) Printf(ctx context.Context, format string, v ...any) {
	l.logger.WithContext(ctx).Printf(format, v...)
}

// SingleFlightFactory creates a stash.Wrapper from configuration.
type SingleFlightFactory func(l *log.Logger, c *config.Config, s storage.Backend, checker storage.Checker) (stash.Wrapper, error)

// singleFlightRegistry holds registered single flight factories.
var singleFlightRegistry = map[string]SingleFlightFactory{}

// RegisterSingleFlight registers a single flight factory for a given type name.
func RegisterSingleFlight(name string, factory SingleFlightFactory) {
	singleFlightRegistry[name] = factory
}

func getSingleFlight(l *log.Logger, c *config.Config, s storage.Backend, checker storage.Checker) (stash.Wrapper, error) {
	switch c.SingleFlightType {
	case "", "memory":
		return stash.WithSingleflight, nil
	case "etcd":
		if c.SingleFlight == nil || c.SingleFlight.Etcd == nil {
			return nil, errors.New("etcd config must be present")
		}
		endpoints := strings.Split(c.SingleFlight.Etcd.Endpoints, ",")
		return stash.WithEtcd(endpoints, checker)
	case "redis":
		if c.SingleFlight == nil || c.SingleFlight.Redis == nil {
			return nil, errors.New("redis config must be present")
		}
		return stash.WithRedisLock(
			&athensLoggerForRedis{logger: l},
			c.SingleFlight.Redis.Endpoint,
			c.SingleFlight.Redis.Password,
			checker,
			c.SingleFlight.Redis.LockConfig)
	case "redis-sentinel":
		if c.SingleFlight == nil || c.SingleFlight.RedisSentinel == nil {
			return nil, errors.New("redis config must be present")
		}
		return stash.WithRedisSentinelLock(
			&athensLoggerForRedis{logger: l},
			c.SingleFlight.RedisSentinel.Endpoints,
			c.SingleFlight.RedisSentinel.MasterName,
			c.SingleFlight.RedisSentinel.SentinelPassword,
			c.SingleFlight.RedisSentinel.RedisUsername,
			c.SingleFlight.RedisSentinel.RedisPassword,
			checker,
			c.SingleFlight.RedisSentinel.LockConfig,
		)
	default:
		// Check registry for build-tagged backends
		if factory, ok := singleFlightRegistry[c.SingleFlightType]; ok {
			return factory(l, c, s, checker)
		}
		return nil, fmt.Errorf("single flight type %q is unknown (you may need to build with -tags %s)", c.SingleFlightType, c.SingleFlightType)
	}
}

// IndexFactory creates an index.Indexer from configuration.
type IndexFactory func(c *config.Config) (index.Indexer, error)

// indexRegistry holds registered index factories.
var indexRegistry = map[string]IndexFactory{}

// RegisterIndex registers an index factory for a given type name.
func RegisterIndex(name string, factory IndexFactory) {
	indexRegistry[name] = factory
}

func getIndex(c *config.Config) (index.Indexer, error) {
	switch c.IndexType {
	case "", "none":
		return nop.New(), nil
	case "memory":
		return mem.New(), nil
	default:
		// Check registry for build-tagged backends
		if factory, ok := indexRegistry[c.IndexType]; ok {
			return factory(c)
		}
		return nil, fmt.Errorf("index type %q is unknown (you may need to build with -tags %s)", c.IndexType, c.IndexType)
	}
}

package observ

import (
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestRegisterStatsExporter_Empty(t *testing.T) {
	r := mux.NewRouter()
	_, err := RegisterStatsExporter(r, "", "test-service")
	require.Error(t, err)
	require.Contains(t, err.Error(), "StatsExporter not specified")
}

func TestRegisterStatsExporter_Unsupported(t *testing.T) {
	r := mux.NewRouter()
	_, err := RegisterStatsExporter(r, "unknown", "test-service")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not supported")
}

func TestRegisterStatsExporter_Prometheus(t *testing.T) {
	r := mux.NewRouter()
	flush, err := RegisterStatsExporter(r, "prometheus", "test_service")
	require.NoError(t, err)
	require.NotNil(t, flush)
}

func TestRegisterStatsExporter_Datadog(t *testing.T) {
	r := mux.NewRouter()
	flush, err := RegisterStatsExporter(r, "datadog", "test-service")
	require.NoError(t, err)
	require.NotNil(t, flush)
	flush()
}

func TestRegisterViews(t *testing.T) {
	err := registerViews()
	require.NoError(t, err)
}

func TestCustomViews(t *testing.T) {
	views := customViews()
	require.NotEmpty(t, views)
}

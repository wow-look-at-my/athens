package observ

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegisterExporter_Empty(t *testing.T) {
	_, err := RegisterExporter("", "", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Exporter not specified")
}

func TestRegisterExporter_Unsupported(t *testing.T) {
	_, err := RegisterExporter("unknown_exporter", "", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not supported")
}

func TestRegisterExporter_Jaeger_EmptyURL(t *testing.T) {
	_, err := RegisterExporter("jaeger", "", "test-service", "development")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Exporter URL is empty")
}

func TestRegisterExporter_Jaeger(t *testing.T) {
	flush, err := RegisterExporter("jaeger", "http://localhost:14268/api/traces", "test-service", "development")
	require.NoError(t, err)
	require.NotNil(t, flush)
	flush()
}

func TestRegisterExporter_Datadog(t *testing.T) {
	flush, err := RegisterExporter("datadog", "localhost:8126", "test-service", "development")
	require.NoError(t, err)
	require.NotNil(t, flush)
	flush()
}

func TestStartSpan(t *testing.T) {
	ctx := context.Background()
	newCtx, span := StartSpan(ctx, "test.op")
	require.NotNil(t, newCtx)
	require.NotNil(t, span)
	span.End()
}

func TestTraceRegisterExporter_Development(t *testing.T) {
	// Just verify it doesn't panic with development env
	traceRegisterExporter(nil, "development")
}

func TestTraceRegisterExporter_Production(t *testing.T) {
	// Just verify it doesn't panic with non-development env
	traceRegisterExporter(nil, "production")
}

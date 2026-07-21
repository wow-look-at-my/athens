package observ

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats/view"
)

func TestCacheLookupMetric(t *testing.T) {
	// Register only the cache view
	require.NoError(t, view.Register(cacheLookupView))

	defer view.Unregister(cacheLookupView)

	ctx := context.Background()

	RecordCacheLookup(ctx, "hit", "info")

	rows, err := view.RetrieveData("cache_lookup_total")
	require.Nil(t, err)

	require.Equal(t, 1, len(rows))

	count := rows[0].Data.(*view.CountData).Value
	require.Equal(t, int64(1), count)

}

func TestUpstreamFetchCounter(t *testing.T) {
	require.NoError(t, view.Register(upstreamFetchView))

	defer view.Unregister(upstreamFetchView)

	ctx := context.Background()

	RecordUpstreamFetch(ctx, "success")

	rows, err := view.RetrieveData("upstream_fetch_total")
	require.Nil(t, err)

	require.Equal(t, 1, len(rows))

	count := rows[0].Data.(*view.CountData).Value
	require.Equal(t, int64(1), count)

}

func TestUpstreamFetchDurationHistogram(t *testing.T) {
	require.NoError(t, view.Register(upstreamFetchLatencyView))

	defer view.Unregister(upstreamFetchLatencyView)

	ctx := context.Background()

	RecordUpstreamFetchDuration(ctx, "success", 2*time.Second)

	rows, err := view.RetrieveData("upstream_fetch_duration_seconds")
	require.Nil(t, err)

	require.Equal(t, 1, len(rows))

	dist := rows[0].Data.(*view.DistributionData)

	require.Equal(t, int64(1), dist.Count)

}

func TestCacheHitsMetric(t *testing.T) {
	require.NoError(t, view.Register(cacheHitsView))
	defer view.Unregister(cacheHitsView)

	ctx := context.Background()
	RecordCacheHit(ctx, "info")

	rows, err := view.RetrieveData("athens_cache_hits_total")
	require.Nil(t, err)
	require.Equal(t, 1, len(rows))
	count := rows[0].Data.(*view.CountData).Value
	require.Equal(t, int64(1), count)
}

func TestCacheMissesMetric(t *testing.T) {
	require.NoError(t, view.Register(cacheMissesView))
	defer view.Unregister(cacheMissesView)

	ctx := context.Background()
	RecordCacheMiss(ctx, "zip")

	rows, err := view.RetrieveData("athens_cache_misses_total")
	require.Nil(t, err)
	require.Equal(t, 1, len(rows))
	count := rows[0].Data.(*view.CountData).Value
	require.Equal(t, int64(1), count)
}

func TestBytesServedMetric(t *testing.T) {
	require.NoError(t, view.Register(bytesServedView))
	defer view.Unregister(bytesServedView)

	ctx := context.Background()
	RecordBytesServed(ctx, "mod", 1024)

	rows, err := view.RetrieveData("athens_bytes_served_total")
	require.Nil(t, err)
	require.Equal(t, 1, len(rows))
	sum := rows[0].Data.(*view.SumData).Value
	require.Equal(t, float64(1024), sum)
}

func TestBytesFetchedMetric(t *testing.T) {
	require.NoError(t, view.Register(bytesFetchedView))
	defer view.Unregister(bytesFetchedView)

	ctx := context.Background()
	RecordBytesFetched(ctx, "zip", 2048)

	rows, err := view.RetrieveData("athens_bytes_fetched_total")
	require.Nil(t, err)
	require.Equal(t, 1, len(rows))
	sum := rows[0].Data.(*view.SumData).Value
	require.Equal(t, float64(2048), sum)
}

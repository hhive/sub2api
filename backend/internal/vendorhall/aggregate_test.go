package vendorhall

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAggregateMetricsUsesRawDenominatorsAndMergedHistograms(t *testing.T) {
	start := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	metrics := aggregateMinutes([]MinuteMetric{
		{
			BucketStart: start, SuccessCount: 1, EligibleCount: 2,
			DurationCount: 1, DurationSumMs: 100,
			DurationHistogram: map[string]int64{"100": 1}, FirstTokenHistogram: map[string]int64{"20": 1},
			InputTokens: 90, CacheReadTokens: 10,
		},
		{
			BucketStart: start.Add(time.Minute), SuccessCount: 8, UpstreamErrorCount: 1, EligibleCount: 8,
			DurationCount: 9, DurationSumMs: 1900,
			DurationHistogram: map[string]int64{"200": 8, "1000": 1}, FirstTokenHistogram: map[string]int64{"40": 8, "500": 1},
			InputTokens: 50, CacheReadTokens: 50,
		},
	}, start, start.Add(2*time.Minute), 48)

	require.NotNil(t, metrics.Availability)
	require.InDelta(t, 0.9, *metrics.Availability, 0.0001)
	require.NotNil(t, metrics.CacheHitRate)
	require.InDelta(t, 0.3, *metrics.CacheHitRate, 0.0001)
	require.Equal(t, float64(1000), *metrics.DurationP95Ms)
	require.Equal(t, float64(500), *metrics.FirstTokenP95Ms)
	require.Equal(t, int64(10), metrics.RequestCount)
}

func TestAggregateMetricsBoundsTrendAndKeepsEmptyValuesNull(t *testing.T) {
	start := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	minutes := make([]MinuteMetric, 0, 180)
	for i := 0; i < 180; i++ {
		minutes = append(minutes, MinuteMetric{
			BucketStart:  start.Add(time.Duration(i) * time.Minute),
			SuccessCount: 1, EligibleCount: 1,
			FirstTokenHistogram: map[string]int64{"25": 1},
		})
	}
	metrics := aggregateMinutes(minutes, start, start.Add(3*time.Hour), 48)

	require.LessOrEqual(t, len(metrics.Trend), 48)
	require.Nil(t, metrics.DurationP95Ms)
	require.Nil(t, metrics.CacheHitRate)
	require.NotNil(t, metrics.FirstTokenP95Ms)
}

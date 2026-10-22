package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakePublicUsageTotalsRepo struct {
	requests int64
	tokens   int64
	err      error
}

func (f fakePublicUsageTotalsRepo) GetPublicUsageTotals(context.Context) (int64, int64, error) {
	return f.requests, f.tokens, f.err
}

type fakePublicAvailabilityRepo struct {
	success int64
	total   int64
	err     error
}

func (f fakePublicAvailabilityRepo) GetPublicAvailability(context.Context, time.Time, time.Time) (int64, int64, error) {
	return f.success, f.total, f.err
}

func TestPublicUsageTotalsFrom(t *testing.T) {
	requests, tokens, err := publicUsageTotalsFrom(context.Background(), fakePublicUsageTotalsRepo{
		requests: 2853851,
		tokens:   318729031558,
	})
	require.NoError(t, err)
	require.Equal(t, int64(2853851), requests)
	require.Equal(t, int64(318729031558), tokens)

	_, _, err = publicUsageTotalsFrom(context.Background(), struct{}{})
	require.ErrorIs(t, err, ErrPublicStatsUnsupported)

	_, _, err = publicUsageTotalsFrom(context.Background(), nil)
	require.ErrorContains(t, err, "nil usage repository")

	_, _, err = publicUsageTotalsFrom(context.Background(), fakePublicUsageTotalsRepo{err: errors.New("boom")})
	require.ErrorContains(t, err, "boom")
}

func TestPublicAvailabilityFrom(t *testing.T) {
	start := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)

	success, total, err := publicAvailabilityFrom(context.Background(), fakePublicAvailabilityRepo{
		success: 408394,
		total:   434281,
	}, start, end)
	require.NoError(t, err)
	require.Equal(t, int64(408394), success)
	require.Equal(t, int64(434281), total)

	_, _, err = publicAvailabilityFrom(context.Background(), struct{}{}, start, end)
	require.ErrorIs(t, err, ErrPublicStatsUnsupported)

	_, _, err = publicAvailabilityFrom(context.Background(), nil, start, end)
	require.ErrorContains(t, err, "nil ops repository")
}

func TestPublicStatsServiceNilReceivers(t *testing.T) {
	var dashboard *DashboardService
	_, _, err := dashboard.PublicUsageTotals(context.Background())
	require.ErrorContains(t, err, "nil dashboard service")

	_, _, err = (&DashboardService{}).PublicUsageTotals(context.Background())
	require.ErrorContains(t, err, "nil usage repository")

	var ops *OpsService
	_, _, err = ops.PublicAvailability(context.Background(), time.Now(), time.Now())
	require.ErrorContains(t, err, "nil ops service")

	_, _, err = (&OpsService{}).PublicAvailability(context.Background(), time.Now(), time.Now())
	require.ErrorContains(t, err, "nil ops repository")
}

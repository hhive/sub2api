package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakePublicStatsUsage struct {
	requests int64
	tokens   int64
	err      error
	calls    int32
}

func (f *fakePublicStatsUsage) PublicUsageTotals(context.Context) (int64, int64, error) {
	atomic.AddInt32(&f.calls, 1)
	return f.requests, f.tokens, f.err
}

type fakePublicStatsAvailability struct {
	success int64
	total   int64
	err     error
	calls   int32
	start   time.Time
	end     time.Time
}

func (f *fakePublicStatsAvailability) PublicAvailability(_ context.Context, start, end time.Time) (int64, int64, error) {
	atomic.AddInt32(&f.calls, 1)
	f.start = start
	f.end = end
	return f.success, f.total, f.err
}

func newPublicStatsTestHandler(now *time.Time) (*PublicStatsHandler, *fakePublicStatsUsage, *fakePublicStatsAvailability) {
	usage := &fakePublicStatsUsage{requests: 2853851, tokens: 318729031558}
	availability := &fakePublicStatsAvailability{success: 408394, total: 434281}
	h := NewPublicStatsHandler(usage, availability)
	h.loc = time.FixedZone("CST", 8*3600)
	h.now = func() time.Time { return *now }
	return h, usage, availability
}

func runPublicStatsRequest(t *testing.T, h *PublicStatsHandler) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/settings/public-stats", nil)
	h.Get(c)
	return recorder
}

type publicStatsEnvelope struct {
	Code int                `json:"code"`
	Data PublicStatsPayload `json:"data"`
}

func TestPublicStatsHandlerContract(t *testing.T) {
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	h, _, availability := newPublicStatsTestHandler(&now)

	recorder := runPublicStatsRequest(t, h)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "public, max-age=60", recorder.Header().Get("Cache-Control"))

	var envelope publicStatsEnvelope
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
	require.Equal(t, "2026-05-07", envelope.Data.RunningSince)
	require.Equal(t, 168, envelope.Data.RunningDays)
	require.Equal(t, int64(2853851), envelope.Data.TotalRequests)
	require.Equal(t, int64(318729031558), envelope.Data.TotalTokens)
	require.Equal(t, publicStatsWindowDays, envelope.Data.AvailabilityWindowDays)
	require.NotNil(t, envelope.Data.Availability30d)
	require.InDelta(t, 0.9404, *envelope.Data.Availability30d, 0.00005)
	require.Equal(t, "2026-10-22T12:00:00Z", envelope.Data.UpdatedAt)

	require.Equal(t, time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC), availability.start)
	require.Equal(t, time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC), availability.end)
}

func TestPublicStatsHandlerRunningDaysBoundary(t *testing.T) {
	// 北京时间 2026-10-23 00:30，UTC 仍是 10-22。
	now := time.Date(2026, 10, 22, 16, 30, 0, 0, time.UTC)
	h, _, _ := newPublicStatsTestHandler(&now)

	recorder := runPublicStatsRequest(t, h)
	var envelope publicStatsEnvelope
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Equal(t, 169, envelope.Data.RunningDays)
}

func TestPublicStatsRunningDaysBoundaries(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	cases := []struct {
		name string
		now  time.Time
		want int
	}{
		{"before launch", time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC), 0},
		{"launch day", time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC), 0},
		{"next day", time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC), 1},
		{"2026-10-22", time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC), 168},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, publicStatsRunningDays(tc.now, loc))
		})
	}
}

func TestPublicStatsHandlerCachesWithinTTL(t *testing.T) {
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	h, usage, availability := newPublicStatsTestHandler(&now)

	_ = runPublicStatsRequest(t, h)
	_ = runPublicStatsRequest(t, h)
	require.Equal(t, int32(1), atomic.LoadInt32(&usage.calls))
	require.Equal(t, int32(1), atomic.LoadInt32(&availability.calls))

	now = now.Add(publicStatsCacheTTL + time.Second)
	_ = runPublicStatsRequest(t, h)
	require.Equal(t, int32(2), atomic.LoadInt32(&usage.calls))
	require.Equal(t, int32(2), atomic.LoadInt32(&availability.calls))
}

func TestPublicStatsHandlerNullAvailability(t *testing.T) {
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	h, _, availability := newPublicStatsTestHandler(&now)
	availability.success = 0
	availability.total = 0

	recorder := runPublicStatsRequest(t, h)
	var envelope publicStatsEnvelope
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Nil(t, envelope.Data.Availability30d)
}

func TestPublicStatsHandlerErrors(t *testing.T) {
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)

	t.Run("usage query fails", func(t *testing.T) {
		h, usage, _ := newPublicStatsTestHandler(&now)
		usage.err = errors.New("usage boom")
		recorder := runPublicStatsRequest(t, h)
		require.Equal(t, http.StatusInternalServerError, recorder.Code)
	})

	t.Run("availability query fails", func(t *testing.T) {
		h, _, availability := newPublicStatsTestHandler(&now)
		availability.err = errors.New("availability boom")
		recorder := runPublicStatsRequest(t, h)
		require.Equal(t, http.StatusInternalServerError, recorder.Code)
	})

	t.Run("handler misconfigured", func(t *testing.T) {
		h := &PublicStatsHandler{now: func() time.Time { return now }}
		recorder := runPublicStatsRequest(t, h)
		require.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

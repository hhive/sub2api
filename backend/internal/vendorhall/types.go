package vendorhall

import (
	"context"
	"errors"
	"time"
)

var ErrUnavailable = errors.New("vendor hall monitor database unavailable")

type Group struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type TrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	TTFTP95Ms *float64  `json:"ttft_p95_ms"`
}

type Metrics struct {
	SuccessCount        int64        `json:"success_count"`
	UpstreamErrorCount  int64        `json:"upstream_error_count"`
	EligibleCount       int64        `json:"eligible_count"`
	RequestCount        int64        `json:"request_count"`
	Availability        *float64     `json:"availability"`
	CacheHitRate        *float64     `json:"cache_hit_rate"`
	AverageDurationMs   *float64     `json:"average_duration_ms"`
	DurationP95Ms       *float64     `json:"duration_p95_ms"`
	FirstTokenP95Ms     *float64     `json:"first_token_p95_ms"`
	UpstreamMultiplier  *float64     `json:"upstream_multiplier"`
	Trend               []TrendPoint `json:"trend"`
	inputTokens         int64
	cacheReadTokens     int64
	cacheCreateTokens   int64
	firstTokenHistogram map[string]int64
	collectedAt         *time.Time
	hasSamples          bool
}

type Account struct {
	AccountID              int64        `json:"account_id"`
	AccountName            string       `json:"account_name"`
	Platform               string       `json:"platform"`
	GroupName              *string      `json:"group_name"`
	RateMultiplier         *float64     `json:"rate_multiplier"`
	SchedulingStatus       string       `json:"scheduling_status"`
	Availability           *float64     `json:"availability"`
	CacheHitRate           *float64     `json:"cache_hit_rate"`
	AverageLatencyMs       *float64     `json:"average_latency_ms"`
	P95LatencyMs           *float64     `json:"p95_latency_ms"`
	UserTTFTP95Ms          *float64     `json:"user_ttft_p95_ms"`
	RequestCount           int64        `json:"request_count"`
	CollectedAt            *time.Time   `json:"collected_at"`
	TempUnschedulableUntil *time.Time   `json:"temp_unschedulable_until"`
	Trend                  []TrendPoint `json:"trend"`
	groups                 []Group
	remoteStatus           *string
	schedulable            *bool
	lastSyncedAt           *time.Time
	metrics                Metrics
}

type Summary struct {
	TotalAccounts       int        `json:"total_accounts"`
	HealthyAccounts     int        `json:"healthy_accounts"`
	PausedAccounts      int        `json:"paused_accounts"`
	AverageAvailability *float64   `json:"average_availability"`
	UpdatedAt           *time.Time `json:"updated_at"`
}

type ListResult struct {
	Items       []Account `json:"items"`
	Total       int       `json:"total"`
	Page        int       `json:"page"`
	PageSize    int       `json:"page_size"`
	Pages       int       `json:"pages"`
	Window      string    `json:"window"`
	WindowStart time.Time `json:"window_start"`
	WindowEnd   time.Time `json:"window_end"`
	Summary     Summary   `json:"summary"`
}

type Lister interface {
	List(context.Context, ListParams) (*ListResult, error)
}

type MinuteMetric struct {
	AccountID           int
	BucketStart         time.Time
	SuccessCount        int64
	UpstreamErrorCount  int64
	EligibleCount       int64
	DurationCount       int64
	DurationSumMs       int64
	DurationHistogram   map[string]int64
	FirstTokenHistogram map[string]int64
	InputTokens         int64
	CacheReadTokens     int64
	CacheCreationTokens int64
	UpstreamMultiplier  *float64
}

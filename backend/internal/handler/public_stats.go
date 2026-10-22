package handler

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"

	"github.com/gin-gonic/gin"
)

const (
	publicStatsWindowDays = 30
	publicStatsCacheTTL   = 60 * time.Second
	publicStatsQueryTTL   = 2 * time.Second
)

// 运行天数起点：北京时间 2026-05-07，业务口径常量。
const publicStatsRunningSince = "2026-05-07"

var publicStatsLaunchDate = time.Date(2026, time.May, 7, 0, 0, 0, 0, time.UTC)

// PublicStatsUsageProvider 提供全量调用量与 Token 总量。
type PublicStatsUsageProvider interface {
	PublicUsageTotals(ctx context.Context) (int64, int64, error)
}

// PublicStatsAvailabilityProvider 提供时间窗口内的成功数与 SLA 分母。
type PublicStatsAvailabilityProvider interface {
	PublicAvailability(ctx context.Context, start, end time.Time) (int64, int64, error)
}

// PublicStatsHandler 提供无需认证的公开可靠性指标。
type PublicStatsHandler struct {
	usage PublicStatsUsageProvider
	ops   PublicStatsAvailabilityProvider

	mu       sync.Mutex
	cached   *PublicStatsPayload
	cachedAt time.Time

	now func() time.Time
	loc *time.Location
}

// PublicStatsPayload 是对外契约，字段只增不改。
type PublicStatsPayload struct {
	RunningSince           string   `json:"running_since"`
	RunningDays            int      `json:"running_days"`
	TotalRequests          int64    `json:"total_requests"`
	TotalTokens            int64    `json:"total_tokens"`
	Availability30d        *float64 `json:"availability_30d"`
	AvailabilityWindowDays int      `json:"availability_window_days"`
	UpdatedAt              string   `json:"updated_at"`
}

// NewPublicStatsHandler 创建公开统计 handler。
func NewPublicStatsHandler(usage PublicStatsUsageProvider, ops PublicStatsAvailabilityProvider) *PublicStatsHandler {
	return &PublicStatsHandler{
		usage: usage,
		ops:   ops,
		now:   timezone.Now,
		loc:   timezone.Location(),
	}
}

// Get 返回公开可靠性指标。
// GET /api/v1/settings/public-stats
func (h *PublicStatsHandler) Get(c *gin.Context) {
	payload, err := h.build(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// 响应为全局聚合，无用户维度。
	c.Header("Cache-Control", "public, max-age=60")
	response.Success(c, payload)
}

func (h *PublicStatsHandler) build(ctx context.Context) (*PublicStatsPayload, error) {
	if h == nil || h.usage == nil || h.ops == nil {
		return nil, fmt.Errorf("public stats handler is not configured")
	}

	now := h.now()
	h.mu.Lock()
	if h.cached != nil && now.Sub(h.cachedAt) < publicStatsCacheTTL {
		cached := *h.cached
		h.mu.Unlock()
		return &cached, nil
	}
	h.mu.Unlock()

	queryCtx, cancel := context.WithTimeout(ctx, publicStatsQueryTTL)
	defer cancel()

	requests, tokens, err := h.usage.PublicUsageTotals(queryCtx)
	if err != nil {
		return nil, fmt.Errorf("query public usage totals: %w", err)
	}

	utcNow := now.UTC()
	end := time.Date(utcNow.Year(), utcNow.Month(), utcNow.Day(), 0, 0, 0, 0, time.UTC)
	start := end.AddDate(0, 0, -publicStatsWindowDays)
	success, total, err := h.ops.PublicAvailability(queryCtx, start, end)
	if err != nil {
		return nil, fmt.Errorf("query public availability: %w", err)
	}

	payload := &PublicStatsPayload{
		RunningSince:           publicStatsRunningSince,
		RunningDays:            publicStatsRunningDays(now, h.loc),
		TotalRequests:          requests,
		TotalTokens:            tokens,
		AvailabilityWindowDays: publicStatsWindowDays,
		UpdatedAt:              now.UTC().Format(time.RFC3339),
	}
	if total > 0 {
		ratio := math.Round(float64(success)/float64(total)*10000) / 10000
		payload.Availability30d = &ratio
	}

	h.mu.Lock()
	h.cached = payload
	h.cachedAt = now
	h.mu.Unlock()

	cached := *payload
	return &cached, nil
}

// publicStatsRunningDays 按站点时区自然日计算运行天数。
func publicStatsRunningDays(now time.Time, loc *time.Location) int {
	if loc == nil {
		loc = time.Local
	}
	local := now.In(loc)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
	days := int(today.Sub(publicStatsLaunchDate).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

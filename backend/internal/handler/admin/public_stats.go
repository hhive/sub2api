package admin

import (
	"context"
	"errors"
	"time"
)

var errPublicStatsUnavailable = errors.New("public stats dependencies are not configured")

// PublicUsageTotals 供公开统计 handler 读取全量调用量与 Token 总量。
func (h *DashboardHandler) PublicUsageTotals(ctx context.Context) (int64, int64, error) {
	if h == nil || h.dashboardService == nil {
		return 0, 0, errPublicStatsUnavailable
	}
	return h.dashboardService.PublicUsageTotals(ctx)
}

// PublicAvailability 供公开统计 handler 读取 [start, end) 的成功数与 SLA 分母。
func (h *OpsHandler) PublicAvailability(ctx context.Context, start, end time.Time) (int64, int64, error) {
	if h == nil || h.opsService == nil {
		return 0, 0, errPublicStatsUnavailable
	}
	return h.opsService.PublicAvailability(ctx, start, end)
}

package service

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrPublicStatsUnsupported 表示当前仓储实现没有提供公开统计所需的查询能力。
// 该错误向上返回后由 HTTP 层转换为 500。
var ErrPublicStatsUnsupported = errors.New("public stats query is not supported by the configured repository")

type publicUsageTotalsRepository interface {
	GetPublicUsageTotals(ctx context.Context) (int64, int64, error)
}

type publicAvailabilityRepository interface {
	GetPublicAvailability(ctx context.Context, start, end time.Time) (int64, int64, error)
}

func publicUsageTotalsFrom(ctx context.Context, repo any) (int64, int64, error) {
	if repo == nil {
		return 0, 0, fmt.Errorf("nil usage repository")
	}
	provider, ok := repo.(publicUsageTotalsRepository)
	if !ok {
		return 0, 0, ErrPublicStatsUnsupported
	}
	return provider.GetPublicUsageTotals(ctx)
}

func publicAvailabilityFrom(ctx context.Context, repo any, start, end time.Time) (int64, int64, error) {
	if repo == nil {
		return 0, 0, fmt.Errorf("nil ops repository")
	}
	provider, ok := repo.(publicAvailabilityRepository)
	if !ok {
		return 0, 0, ErrPublicStatsUnsupported
	}
	return provider.GetPublicAvailability(ctx, start, end)
}

// PublicUsageTotals 返回公开看板使用的全量调用数与 Token 总量。
func (s *DashboardService) PublicUsageTotals(ctx context.Context) (int64, int64, error) {
	if s == nil {
		return 0, 0, fmt.Errorf("nil dashboard service")
	}
	return publicUsageTotalsFrom(ctx, s.usageRepo)
}

// PublicAvailability 返回公开看板使用的 [start, end) 请求成功数与 SLA 分母。
func (s *OpsService) PublicAvailability(ctx context.Context, start, end time.Time) (int64, int64, error) {
	if s == nil {
		return 0, 0, fmt.Errorf("nil ops service")
	}
	return publicAvailabilityFrom(ctx, s.opsRepo, start, end)
}

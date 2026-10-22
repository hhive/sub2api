package repository

import (
	"context"
	"fmt"
	"time"
)

// GetPublicUsageTotals 返回全量成功调用数与 Token 总量。
//
// 数据源是 dashboard 的按天预聚合表，保留窗口 730 天。
func (r *usageLogRepository) GetPublicUsageTotals(ctx context.Context) (int64, int64, error) {
	if r == nil || r.sql == nil {
		return 0, 0, fmt.Errorf("nil usage log repository")
	}

	const query = `
SELECT
  COALESCE(SUM(total_requests), 0) AS total_requests,
  COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) AS total_tokens
FROM usage_dashboard_daily`

	var requests, tokens int64
	if err := scanSingleRow(ctx, r.sql, query, nil, &requests, &tokens); err != nil {
		return 0, 0, err
	}
	return requests, tokens, nil
}

// GetPublicAvailability 返回 [start, end) 内全局请求的成功数与 SLA 分母。
//
// 只取 (platform, group_id) 均为 NULL 的整站汇总行；窗口由调用方传入的日期决定。
func (r *opsRepository) GetPublicAvailability(ctx context.Context, start, end time.Time) (int64, int64, error) {
	if r == nil || r.db == nil {
		return 0, 0, fmt.Errorf("nil ops repository")
	}

	const query = `
SELECT
  COALESCE(SUM(success_count), 0) AS success_count,
  COALESCE(SUM(success_count + error_count_sla), 0) AS sla_total
FROM ops_metrics_daily
WHERE platform IS NULL
  AND group_id IS NULL
  AND bucket_date >= $1::date
  AND bucket_date < $2::date`

	var success, total int64
	if err := r.db.QueryRowContext(ctx, query, start, end).Scan(&success, &total); err != nil {
		return 0, 0, err
	}
	return success, total, nil
}

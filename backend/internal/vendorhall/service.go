package vendorhall

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq"
)

type Service struct {
	dsn func() string
	mu  sync.Mutex
	db  *sql.DB
	now func() time.Time
}

func NewServiceFromEnv() *Service {
	return &Service{dsn: func() string { return strings.TrimSpace(os.Getenv("RELAY_MONITOR_DATABASE_URL")) }, now: time.Now}
}
func NewServiceWithDB(db *sql.DB) *Service { return &Service{db: db, now: time.Now} }

func (s *Service) database(ctx context.Context) (*sql.DB, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		return s.db, nil
	}
	if s.dsn == nil || s.dsn() == "" {
		return nil, ErrUnavailable
	}
	db, err := sql.Open("postgres", s.dsn())
	if err != nil {
		return nil, fmt.Errorf("%w: open failed", ErrUnavailable)
	}
	db.SetMaxOpenConns(3)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(5 * time.Minute)
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("%w: connection failed", ErrUnavailable)
	}
	s.db = db
	return db, nil
}

type rawAccount struct {
	monitorID                           int
	sourceID, name                      string
	platform, accountType, remoteStatus sql.NullString
	schedulable                         sql.NullBool
	priority                            int
	groupsJSON                          []byte
	tempUntil                           sql.NullTime
	tempReason                          sql.NullString
	lastSynced                          sql.NullTime
}

func (s *Service) List(ctx context.Context, params ListParams) (*ListResult, error) {
	db, err := s.database(ctx)
	if err != nil {
		return nil, err
	}
	queryCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	rows, err := db.QueryContext(queryCtx, `SELECT "id", "sourceAccountId", "name", "platform", "type", "remoteStatus", "schedulable", "priority", COALESCE("groupProjection", '[]'::jsonb)::text, "tempUnschedulableUntil", "tempUnschedulableReason", "lastSyncedAt" FROM "Sub2ApiAccount" WHERE "syncState" <> 'RETIRED' ORDER BY "id"`)
	if err != nil {
		return nil, fmt.Errorf("%w: account query failed", ErrUnavailable)
	}
	defer rows.Close()
	raw := make([]rawAccount, 0)
	for rows.Next() {
		var item rawAccount
		if err := rows.Scan(&item.monitorID, &item.sourceID, &item.name, &item.platform, &item.accountType, &item.remoteStatus, &item.schedulable, &item.priority, &item.groupsJSON, &item.tempUntil, &item.tempReason, &item.lastSynced); err != nil {
			return nil, fmt.Errorf("%w: account row invalid", ErrUnavailable)
		}
		raw = append(raw, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: account rows failed", ErrUnavailable)
	}
	now := s.now().UTC()
	end := now.Truncate(time.Minute)
	start := end.Add(-windowDurations[params.Window])
	accounts := make([]Account, 0, len(raw))
	monitorToIndex := map[int]int{}
	monitorIDs := make([]int64, 0, len(raw))
	for _, item := range raw {
		sourceID, err := strconv.ParseInt(item.sourceID, 10, 64)
		if err != nil || sourceID <= 0 {
			continue
		}
		groups := []Group{}
		_ = json.Unmarshal(item.groupsJSON, &groups)
		account := Account{AccountID: sourceID, AccountName: item.name, groups: groups, Trend: []TrendPoint{}, metrics: Metrics{Trend: []TrendPoint{}}}
		if item.platform.Valid {
			account.Platform = item.platform.String
		}
		if item.remoteStatus.Valid {
			account.remoteStatus = &item.remoteStatus.String
		}
		if item.schedulable.Valid {
			account.schedulable = &item.schedulable.Bool
		}
		if item.tempUntil.Valid {
			value := item.tempUntil.Time.UTC()
			account.TempUnschedulableUntil = &value
		}
		if item.lastSynced.Valid {
			value := item.lastSynced.Time.UTC()
			account.lastSyncedAt = &value
		}
		if len(groups) > 0 {
			names := make([]string, 0, len(groups))
			for _, group := range groups {
				if strings.TrimSpace(group.Name) != "" {
					names = append(names, strings.TrimSpace(group.Name))
				}
			}
			if len(names) > 0 {
				name := strings.Join(names, ", ")
				account.GroupName = &name
			}
		}
		if !matches(account, params, now) {
			continue
		}
		monitorToIndex[item.monitorID] = len(accounts)
		monitorIDs = append(monitorIDs, int64(item.monitorID))
		accounts = append(accounts, account)
	}
	minutesByAccount := map[int][]MinuteMetric{}
	if len(monitorIDs) > 0 {
		metricRows, err := db.QueryContext(queryCtx, `SELECT "accountId", "bucketStart", "successCount", "upstreamErrorCount", "eligibleCount", "durationCount", "durationSumMs", COALESCE("durationHistogram", '{}'::jsonb)::text, COALESCE("firstTokenHistogram", '{}'::jsonb)::text, "inputTokens", "cacheReadTokens", "cacheCreationTokens", "upstreamRateMultiplier", "balanceUsd" FROM "AccountMetricMinute" WHERE "bucketStart" >= $1 AND "bucketStart" < $2 AND "accountId" = ANY($3) ORDER BY "accountId", "bucketStart"`, start, end, pq.Array(monitorIDs))
		if err != nil {
			return nil, fmt.Errorf("%w: metric query failed", ErrUnavailable)
		}
		defer metricRows.Close()
		for metricRows.Next() {
			var row MinuteMetric
			var durationJSON, firstJSON []byte
			var multiplier sql.NullFloat64
			var balance sql.NullFloat64
			if err := metricRows.Scan(&row.AccountID, &row.BucketStart, &row.SuccessCount, &row.UpstreamErrorCount, &row.EligibleCount, &row.DurationCount, &row.DurationSumMs, &durationJSON, &firstJSON, &row.InputTokens, &row.CacheReadTokens, &row.CacheCreationTokens, &multiplier, &balance); err != nil {
				return nil, fmt.Errorf("%w: metric row invalid", ErrUnavailable)
			}
			row.DurationHistogram = map[string]int64{}
			row.FirstTokenHistogram = map[string]int64{}
			_ = json.Unmarshal(durationJSON, &row.DurationHistogram)
			_ = json.Unmarshal(firstJSON, &row.FirstTokenHistogram)
			if multiplier.Valid {
				row.UpstreamMultiplier = ptrFloat(multiplier.Float64)
			}
			if balance.Valid {
				row.BalanceUSD = ptrFloat(balance.Float64)
			}
			minutesByAccount[row.AccountID] = append(minutesByAccount[row.AccountID], row)
		}
		if err := metricRows.Err(); err != nil {
			return nil, fmt.Errorf("%w: metric rows failed", ErrUnavailable)
		}
	}
	for monitorID, index := range monitorToIndex {
		accounts[index].metrics = aggregateMinutes(minutesByAccount[monitorID], start, end, 48)
		applyMetrics(&accounts[index], now)
	}
	sortAccounts(accounts, params)
	summary := summarize(accounts)
	total := len(accounts)
	pages := (total + params.PageSize - 1) / params.PageSize
	offset := (params.Page - 1) * params.PageSize
	if offset > total {
		offset = total
	}
	limit := offset + params.PageSize
	if limit > total {
		limit = total
	}
	return &ListResult{Items: accounts[offset:limit], Total: total, Page: params.Page, PageSize: params.PageSize, Pages: pages, Window: params.Window, WindowStart: start, WindowEnd: end, Summary: summary}, nil
}

func matches(account Account, params ListParams, now time.Time) bool {
	search := strings.ToLower(params.Search)
	if search != "" {
		matched := strings.Contains(strings.ToLower(account.AccountName), search) || strings.Contains(strconv.FormatInt(account.AccountID, 10), search)
		matched = matched || strings.Contains(strings.ToLower(account.Platform), search)
		for _, group := range account.groups {
			matched = matched || strings.Contains(strings.ToLower(group.Name), search)
		}
		if !matched {
			return false
		}
	}
	if params.Group != "" {
		found := false
		for _, group := range account.groups {
			if strconv.FormatInt(group.ID, 10) == params.Group || strings.EqualFold(group.Name, params.Group) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	disabled := account.schedulable != nil && !*account.schedulable
	paused := !disabled && account.TempUnschedulableUntil != nil && account.TempUnschedulableUntil.After(now)
	schedulable := account.schedulable != nil && *account.schedulable && !paused
	unavailable := account.remoteStatus != nil && !strings.EqualFold(*account.remoteStatus, "active")
	switch params.Status {
	case "schedulable":
		return schedulable
	case "paused":
		return paused
	case "disabled":
		return disabled
	case "unavailable":
		return unavailable
	}
	return true
}

func nullableMetric(account Account, field string) *float64 {
	switch field {
	case "availability":
		return account.Availability
	case "cache_hit_rate":
		return account.CacheHitRate
	case "user_ttft":
		return account.UserTTFTP95Ms
	}
	return nil
}
func sortAccounts(accounts []Account, params ListParams) {
	sort.SliceStable(accounts, func(i, j int) bool {
		left, right := accounts[i], accounts[j]
		leftRank, rightRank := schedulingStatusRank(left.SchedulingStatus), schedulingStatusRank(right.SchedulingStatus)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		cmp := 0
		switch params.SortBy {
		case "requests":
			if !left.metrics.hasSamples || !right.metrics.hasSamples {
				if left.metrics.hasSamples == right.metrics.hasSamples {
					return left.AccountID < right.AccountID
				}
				return left.metrics.hasSamples
			}
			if left.RequestCount < right.RequestCount {
				cmp = -1
			} else if left.RequestCount > right.RequestCount {
				cmp = 1
			}
		case "updated_at":
			if left.CollectedAt == nil || right.CollectedAt == nil {
				if left.CollectedAt == nil && right.CollectedAt == nil {
					return left.AccountID < right.AccountID
				}
				return left.CollectedAt != nil
			} else {
				if left.CollectedAt.Before(*right.CollectedAt) {
					cmp = -1
				} else if left.CollectedAt.After(*right.CollectedAt) {
					cmp = 1
				}
			}
		default:
			l, r := nullableMetric(left, params.SortBy), nullableMetric(right, params.SortBy)
			if l == nil || r == nil {
				if l == nil && r == nil {
					return left.AccountID < right.AccountID
				}
				return l != nil
			} else if l != nil && r != nil {
				if *l < *r {
					cmp = -1
				} else if *l > *r {
					cmp = 1
				}
			}
		}
		if cmp == 0 {
			return left.AccountID < right.AccountID
		}
		if params.SortOrder == "desc" {
			return cmp > 0
		}
		return cmp < 0
	})
}
func schedulingStatusRank(status string) int {
	switch status {
	case "schedulable":
		return 0
	case "paused":
		return 1
	case "unknown":
		return 2
	case "disabled":
		return 3
	default:
		return 2
	}
}
func summarize(accounts []Account) Summary {
	result := Summary{TotalAccounts: len(accounts)}
	var success, eligible int64
	for _, account := range accounts {
		if account.SchedulingStatus == "paused" {
			result.PausedAccounts++
		}
		if account.SchedulingStatus == "schedulable" {
			result.HealthyAccounts++
		}
		success += account.metrics.SuccessCount
		eligible += account.metrics.EligibleCount
		if account.CollectedAt != nil && (result.UpdatedAt == nil || account.CollectedAt.After(*result.UpdatedAt)) {
			value := *account.CollectedAt
			result.UpdatedAt = &value
		}
	}
	if eligible > 0 {
		result.AverageAvailability = ptrFloat(float64(success) / float64(eligible))
	}
	return result
}

func applyMetrics(account *Account, now time.Time) {
	account.RateMultiplier = account.metrics.UpstreamMultiplier
	account.BalanceUSD = account.metrics.BalanceUSD
	account.Availability = account.metrics.Availability
	account.CacheHitRate = account.metrics.CacheHitRate
	account.AverageLatencyMs = account.metrics.AverageDurationMs
	account.P95LatencyMs = account.metrics.DurationP95Ms
	account.UserTTFTP95Ms = account.metrics.FirstTokenP95Ms
	account.UserTTFTAverageMs = account.metrics.FirstTokenAverageMs
	account.RequestCount = account.metrics.RequestCount
	account.CollectedAt = account.metrics.collectedAt
	if account.CollectedAt == nil {
		account.CollectedAt = account.lastSyncedAt
	}
	account.Trend = account.metrics.Trend
	paused := account.TempUnschedulableUntil != nil && account.TempUnschedulableUntil.After(now)
	switch {
	case account.schedulable != nil && !*account.schedulable:
		account.SchedulingStatus = "disabled"
	case paused:
		account.SchedulingStatus = "paused"
	case account.schedulable == nil:
		account.SchedulingStatus = "unknown"
	default:
		account.SchedulingStatus = "schedulable"
	}
}

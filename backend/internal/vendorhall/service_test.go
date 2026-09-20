package vendorhall

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestServiceMissingDSNIsUnavailableWithoutOpeningDatabase(t *testing.T) {
	t.Setenv("RELAY_MONITOR_DATABASE_URL", "")
	service := NewServiceFromEnv()
	params, err := ParseListParams(nil)
	require.NoError(t, err)

	_, err = service.List(context.Background(), params)

	require.ErrorIs(t, err, ErrUnavailable)
}

func TestServiceMapsPositiveSourceIDAndAggregatesMonitorRows(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 8, 17, 12, 0, 45, 0, time.UTC)
	windowEnd := now.Truncate(time.Minute)
	accountColumns := []string{"id", "sourceAccountId", "name", "platform", "type", "remoteStatus", "schedulable", "priority", "groupProjection", "tempUnschedulableUntil", "tempUnschedulableReason", "lastSyncedAt"}
	mock.ExpectQuery(`SELECT "id", "sourceAccountId"`).WillReturnRows(sqlmock.NewRows(accountColumns).
		AddRow(1, "7", "Alpha", "openai", "oauth", "active", true, 50, `[{"id":2,"name":"Premium"}]`, nil, nil, now).
		AddRow(2, "broken", "Ignored", nil, nil, nil, nil, 50, `[]`, nil, nil, nil))
	metricColumns := []string{"accountId", "bucketStart", "successCount", "upstreamErrorCount", "eligibleCount", "durationCount", "durationSumMs", "durationHistogram", "firstTokenHistogram", "inputTokens", "cacheReadTokens", "cacheCreationTokens", "upstreamRateMultiplier", "balanceUsd"}
	mock.ExpectQuery(`SELECT "accountId", "bucketStart"`).WithArgs(windowEnd.Add(-24*time.Hour), windowEnd, sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows(metricColumns).
		AddRow(1, windowEnd.Add(-time.Minute), 9, 1, 10, 10, 2000, `{"200":9,"1000":1}`, `{"40":9,"500":1}`, 50, 50, 0, 1.5, 12.34))
	service := NewServiceWithDB(db)
	service.now = func() time.Time { return now }
	params, err := ParseListParams(nil)
	require.NoError(t, err)

	result, err := service.List(context.Background(), params)

	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Equal(t, int64(7), result.Items[0].AccountID)
	require.Equal(t, "Premium", *result.Items[0].GroupName)
	require.InDelta(t, .9, *result.Items[0].Availability, .0001)
	require.InDelta(t, .9, *result.Summary.AverageAvailability, .0001)
	require.InDelta(t, 1.5, *result.Items[0].RateMultiplier, .0001)
	require.NotNil(t, result.Items[0].BalanceUSD)
	require.InDelta(t, 12.34, *result.Items[0].BalanceUSD, .0001)
	require.Equal(t, windowEnd, result.WindowEnd)
	payload, err := json.Marshal(result.Items[0])
	require.NoError(t, err)
	require.Contains(t, string(payload), `"account_name":"Alpha"`)
	require.Contains(t, string(payload), `"user_ttft_p95_ms":500`)
	require.Contains(t, string(payload), `"user_ttft_average_ms":86`)
	require.Contains(t, string(payload), `"ttft_p95_ms":500`)
	require.NotContains(t, string(payload), `"metrics"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestServiceFacetsCoverEveryVisibleAccountBeforeFilters(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 8, 17, 12, 0, 45, 0, time.UTC)
	windowEnd := now.Truncate(time.Minute)
	accountColumns := []string{"id", "sourceAccountId", "name", "platform", "type", "remoteStatus", "schedulable", "priority", "groupProjection", "tempUnschedulableUntil", "tempUnschedulableReason", "lastSyncedAt"}
	mock.ExpectQuery(`SELECT "id", "sourceAccountId"`).WillReturnRows(sqlmock.NewRows(accountColumns).
		AddRow(1, "7", "Alpha", "openai", "oauth", "active", true, 50, `[{"id":3,"name":"Premium"},{"id":1,"name":"Beta"}]`, nil, nil, now).
		AddRow(2, "9", "Alpha", "anthropic", "oauth", "active", true, 50, `[{"id":3,"name":"Premium"}]`, nil, nil, now).
		AddRow(3, "11", "NoPlatform", nil, nil, nil, nil, 50, `[]`, nil, nil, nil).
		AddRow(4, "broken", "Ignored", "openai", nil, nil, nil, 50, `[]`, nil, nil, nil))
	metricColumns := []string{"accountId", "bucketStart", "successCount", "upstreamErrorCount", "eligibleCount", "durationCount", "durationSumMs", "durationHistogram", "firstTokenHistogram", "inputTokens", "cacheReadTokens", "cacheCreationTokens", "upstreamRateMultiplier", "balanceUsd"}
	mock.ExpectQuery(`SELECT "accountId", "bucketStart"`).WithArgs(windowEnd.Add(-24*time.Hour), windowEnd, sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows(metricColumns).
		AddRow(1, windowEnd.Add(-time.Minute), 1, 0, 1, 0, 0, `{}`, `{}`, 0, 0, 0, nil, nil))
	service := NewServiceWithDB(db)
	service.now = func() time.Time { return now }
	params, err := ParseListParams(url.Values{"account_id": {"7"}})
	require.NoError(t, err)

	result, err := service.List(context.Background(), params)

	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Equal(t, []string{"anthropic", "openai"}, result.Facets.Platforms)
	require.Equal(t, []Group{{ID: 1, Name: "Beta"}, {ID: 3, Name: "Premium"}}, result.Facets.Groups)
	require.Equal(t, []FacetAccount{
		{AccountID: 7, AccountName: "Alpha", Platform: "openai"},
		{AccountID: 9, AccountName: "Alpha", Platform: "anthropic"},
		{AccountID: 11, AccountName: "NoPlatform"},
	}, result.Facets.Accounts)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestServiceEmptyFacetSlicesSerializeAsArrays(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 8, 17, 12, 0, 45, 0, time.UTC)
	accountColumns := []string{"id", "sourceAccountId", "name", "platform", "type", "remoteStatus", "schedulable", "priority", "groupProjection", "tempUnschedulableUntil", "tempUnschedulableReason", "lastSyncedAt"}
	mock.ExpectQuery(`SELECT "id", "sourceAccountId"`).WillReturnRows(sqlmock.NewRows(accountColumns))
	service := NewServiceWithDB(db)
	service.now = func() time.Time { return now }
	params, err := ParseListParams(nil)
	require.NoError(t, err)

	result, err := service.List(context.Background(), params)

	require.NoError(t, err)
	payload, err := json.Marshal(result)
	require.NoError(t, err)
	require.Contains(t, string(payload), `"facets":{"platforms":[],"groups":[],"accounts":[]}`)
}

func TestSortAccountsKeepsMissingMetricsLastForBothOrders(t *testing.T) {
	value := .9
	for _, order := range []string{"asc", "desc"} {
		accounts := []Account{{AccountID: 1, SchedulingStatus: "schedulable"}, {AccountID: 2, SchedulingStatus: "schedulable", Availability: &value}}
		sortAccounts(accounts, ListParams{SortBy: "availability", SortOrder: order})
		require.Equal(t, int64(2), accounts[0].AccountID, order)
	}
}

func TestSortAccountsAlwaysGroupsSchedulablePausedDisabledBeforeMetricOrder(t *testing.T) {
	high, low := .99, .01
	for _, order := range []string{"asc", "desc"} {
		accounts := []Account{
			{AccountID: 3, SchedulingStatus: "disabled", Availability: &high},
			{AccountID: 2, SchedulingStatus: "paused", Availability: &high},
			{AccountID: 1, SchedulingStatus: "schedulable", Availability: &low},
			{AccountID: 4, SchedulingStatus: "disabled", Availability: &low},
		}
		sortAccounts(accounts, ListParams{SortBy: "availability", SortOrder: order})
		require.Equal(t, []int64{1, 2}, []int64{accounts[0].AccountID, accounts[1].AccountID}, order)
		require.ElementsMatch(t, []int64{3, 4}, []int64{accounts[2].AccountID, accounts[3].AccountID}, order)
	}
}

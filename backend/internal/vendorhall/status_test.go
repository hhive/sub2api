package vendorhall

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestApplyMetricsPermanentDisableWinsOverTemporaryPause(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	until := now.Add(time.Hour)
	disabled := false
	account := Account{AccountID: 7, schedulable: &disabled, TempUnschedulableUntil: &until}

	applyMetrics(&account, now)

	require.Equal(t, "disabled", account.SchedulingStatus)
}

func TestPausedFilterExcludesPermanentlyDisabledAccount(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	until := now.Add(time.Hour)
	disabled := false
	account := Account{AccountID: 7, schedulable: &disabled, TempUnschedulableUntil: &until}
	params, err := ParseListParams(url.Values{"status": {"paused"}})
	require.NoError(t, err)

	require.False(t, matches(account, params, now))
}

func TestSummaryDoesNotCountPermanentlyDisabledAccountAsPaused(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	until := now.Add(time.Hour)
	account := Account{AccountID: 7, SchedulingStatus: "disabled", TempUnschedulableUntil: &until}

	summary := summarize([]Account{account})

	require.Equal(t, 0, summary.PausedAccounts)
}

func TestDisabledStatusAlwaysRanksAfterUnknown(t *testing.T) {
	require.Greater(t, schedulingStatusRank("disabled"), schedulingStatusRank("unknown"))
}

func TestRequestsSortKeepsAccountsWithoutSamplesLast(t *testing.T) {
	withRequests := Account{AccountID: 1, RequestCount: 1}
	withoutSamples := Account{AccountID: 2, RequestCount: 0}
	withRequests.metrics.hasSamples = true

	accounts := []Account{withoutSamples, withRequests}
	sortAccounts(accounts, ListParams{SortBy: "requests", SortOrder: "asc"})
	require.Equal(t, int64(1), accounts[0].AccountID)
}

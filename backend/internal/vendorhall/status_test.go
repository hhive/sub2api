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

func TestMatchesFiltersByExactAccountID(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	params, err := ParseListParams(url.Values{"account_id": {"7"}})
	require.NoError(t, err)

	require.True(t, matches(Account{AccountID: 7}, params, now))
	require.False(t, matches(Account{AccountID: 17}, params, now))
	require.False(t, matches(Account{AccountID: 70}, params, now))
}

func TestMatchesPlatformFilterIgnoresAccountAndGroupNames(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	params, err := ParseListParams(url.Values{"platform": {"openai"}})
	require.NoError(t, err)

	require.True(t, matches(Account{AccountID: 1, Platform: "openai"}, params, now))
	require.True(t, matches(Account{AccountID: 2, Platform: "OpenAI"}, params, now))
	require.False(t, matches(Account{AccountID: 3, Platform: "anthropic"}, params, now))
	require.False(t, matches(Account{AccountID: 4, Platform: "anthropic", AccountName: "OpenAI East"}, params, now))
	require.False(t, matches(Account{AccountID: 5, Platform: "anthropic", groups: []Group{{ID: 9, Name: "openai"}}}, params, now))
	require.False(t, matches(Account{AccountID: 6, AccountName: "openai"}, params, now))
}

func TestMatchesCombinesAccountPlatformGroupAndStatusFilters(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	schedulable := true
	account := Account{AccountID: 7, Platform: "openai", schedulable: &schedulable, groups: []Group{{ID: 3, Name: "Premium"}}}

	params, err := ParseListParams(url.Values{"account_id": {"7"}, "platform": {"openai"}, "group": {"3"}, "status": {"schedulable"}})
	require.NoError(t, err)
	require.True(t, matches(account, params, now))

	params, err = ParseListParams(url.Values{"account_id": {"7"}, "platform": {"openai"}, "group": {"3"}, "status": {"disabled"}})
	require.NoError(t, err)
	require.False(t, matches(account, params, now))

	params, err = ParseListParams(url.Values{"account_id": {"8"}, "platform": {"openai"}, "group": {"3"}})
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

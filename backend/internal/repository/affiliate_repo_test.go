package repository

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAffiliateUserOverviewSQLIncludesMaturedFrozenQuota(t *testing.T) {
	query := strings.Join(strings.Fields(affiliateUserOverviewSQL), " ")

	require.Contains(t, query, "ua.aff_quota + COALESCE(matured.matured_frozen_quota, 0)")
	require.Contains(t, query, "frozen_until <= NOW()")
}

func TestAffiliateRecordQueriesUseLedgerAuditFields(t *testing.T) {
	source, err := os.ReadFile("affiliate_repo.go")
	require.NoError(t, err)
	content := string(source)

	require.Contains(t, content, "JOIN payment_orders po ON po.id = ual.source_order_id")
	require.Contains(t, content, "ual.amount::double precision")
	require.Contains(t, content, "ual.balance_after::double precision")
	require.NotContains(t, content, "parseAffiliateRebateAmount")
	require.NotContains(t, content, `"current_balance": "u.balance"`)
}

func TestCountPaidInviteesSQLCountsRebatedInvitees(t *testing.T) {
	query := strings.Join(strings.Fields(countPaidInviteesSQL), " ")

	require.Contains(t, query, "user_affiliate_ledger")
	require.Contains(t, query, "COUNT(DISTINCT source_user_id)")
	require.Contains(t, query, "user_id = $1")
	require.Contains(t, query, "action = 'accrue'")
	require.Contains(t, query, "source_user_id IS NOT NULL")
	require.Contains(t, query, "amount > 0")
	require.NotContains(t, query, "payment_orders")
	require.NotContains(t, query, "redeem_codes")
}

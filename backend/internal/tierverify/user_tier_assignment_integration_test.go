//go:build integration

package tierverify

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// tierLookupStub 只解决「邮箱 → 用户」这一步（真实解析由既有 userRepository 负责），
// 让指派链路本身走真实事务与真实 SQL。
type tierLookupStub struct{ user *service.User }

func (s tierLookupStub) GetByEmail(_ context.Context, email string) (*service.User, error) {
	if s.user == nil || email != s.user.Email {
		return nil, service.ErrUserNotFound
	}
	return s.user, nil
}

func newServiceWithLookup(t *testing.T, db *sql.DB, client *dbent.Client, lookup service.UserTierUserLookup) *service.UserTierService {
	t.Helper()
	repo := repository.NewUserTierRepository(client, db)
	creditRepo := repository.NewBalanceCreditRepository(client, db)
	return service.NewUserTierService(repo, creditRepo, client, nil, nil, nil, lookup)
}

// seedUserWithEmail 造一个可被邮箱解析的用户，并按需写入一笔兑换额度（累计消费 = amount - remaining）。
func seedUserWithEmail(t *testing.T, db *sql.DB, consumed float64) *service.User {
	t.Helper()
	ctx := context.Background()
	seedSeq++
	email := fmt.Sprintf("tier-assign-%d-%d@example.test", time.Now().UnixNano(), seedSeq)

	var userID int64
	require.NoError(t, db.QueryRowContext(ctx, `
		INSERT INTO users (email, username, password_hash, role, status, balance, total_recharged, created_at, updated_at)
		VALUES ($1, 'assign-tester', 'x', 'user', 'active', 0, 0, NOW(), NOW()) RETURNING id
	`, email).Scan(&userID))

	// consumed = 0 时不写额度行：该表的 amount 有 CHECK (amount > 0)，且零消费本来就不需要行
	if consumed > 0 {
		_, err := db.ExecContext(ctx, `
			INSERT INTO user_balance_credits (user_id, email, source_type, source_id, source_code, amount, remaining_amount, status, created_at, updated_at)
			VALUES ($1, $2, 'redeem', '1', 'CODE', $3, 0, 'active', NOW(), NOW())
		`, userID, email, consumed)
		require.NoError(t, err)
	}

	return &service.User{ID: userID, Email: email, Username: "assign-tester"}
}

// 手工指派在真实事务/SQL 下的完整语义：覆盖范围、逐档领取、重指派覆盖、删除守卫、取消指派与 code 放开。
func TestVerify_ManualTierAssignmentEndToEnd(t *testing.T) {
	db, client := openHarness(t)
	_, groupID := seedUserAndGroup(t, db)
	seedTiers(t, newService(t, db, client), groupID) // consume_500 / consume_1400

	user := seedUserWithEmail(t, db, 0) // 累计消费 0：不做指派时一档也不该达成
	svc := newServiceWithLookup(t, db, client, tierLookupStub{user: user})
	ctx := context.Background()

	tiers := tierList(t, svc)
	low := findTier(t, tiers, "consume_500")
	high := findTier(t, tiers, "consume_1400")

	before, err := svc.GetUserTierView(ctx, user.ID)
	require.NoError(t, err)
	for _, state := range before.Tiers {
		require.False(t, state.Achieved, "无指派且零消费时不应达成")
	}

	// 指派到高档：该档与其之前的档位都变为可领取
	assignedUser, assignment, err := svc.AssignUserTierByEmail(ctx, user.Email, high.Code, "端到端验证", 1)
	require.NoError(t, err)
	require.Equal(t, user.ID, assignedUser.ID)
	require.Equal(t, service.UserTierAssignmentSourceAdmin, assignment.Source)
	require.Equal(t, int64(1), *assignment.AssignedBy)

	view, err := svc.GetUserTierView(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, view.Tiers, 2)
	require.True(t, view.Tiers[0].Achieved)
	require.True(t, view.Tiers[1].Achieved)
	require.True(t, view.Tiers[0].Claimable)
	require.True(t, view.Tiers[1].Claimable)
	require.NotNil(t, view.AssignedTier)
	require.Equal(t, high.Code, view.AssignedTier.Code)

	// 删除守卫：此刻该档既无授予、也尚未被领取，唯一阻止删除的就是「已有用户指派」
	require.ErrorIs(t, svc.DeleteTier(ctx, high.ID), service.ErrUserTierHasAssignments)

	// 逐档领取（真实事务 + 余额/账本写入）
	for _, tier := range []service.UserTier{low, high} {
		result, err := svc.ClaimTier(ctx, user.ID, tier.ID)
		require.NoError(t, err)
		require.False(t, result.AlreadyClaimed)
	}
	var balance float64
	require.NoError(t, db.QueryRowContext(ctx, `SELECT balance FROM users WHERE id = $1`, user.ID).Scan(&balance))
	require.InDelta(t, 200, balance, 1e-9)

	// 重复领取不重复发放
	result, err := svc.ClaimTier(ctx, user.ID, high.ID)
	require.NoError(t, err)
	require.True(t, result.AlreadyClaimed)
	require.NoError(t, db.QueryRowContext(ctx, `SELECT balance FROM users WHERE id = $1`, user.ID).Scan(&balance))
	require.InDelta(t, 200, balance, 1e-9)

	// 重指派覆盖为一行（真实 ON CONFLICT DO UPDATE）
	_, overwritten, err := svc.AssignUserTierByEmail(ctx, user.Email, low.Code, "改派低档", 2)
	require.NoError(t, err)
	require.Equal(t, low.Code, overwritten.TierCode)
	var assignmentRows int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM user_tier_assignments WHERE user_id = $1`, user.ID).Scan(&assignmentRows))
	require.Equal(t, 1, assignmentRows, "一人一条指派")

	// 取消指派：未领取档位回落（此处两档都已在覆盖范围内且均已领取，故只断言指派消失）
	_, removed, err := svc.UnassignUserTierByEmail(ctx, user.Email)
	require.NoError(t, err)
	require.True(t, removed)
	after, err := svc.GetUserTierView(ctx, user.ID)
	require.NoError(t, err)
	require.Nil(t, after.AssignedTier)
	require.False(t, after.Tiers[1].Achieved, "取消指派后高档回落为未达成（累计消费仍为 0）")

	// 等级标识放开：任意字符的 code 可建可指派（真实 VARCHAR(64) + 唯一索引）
	anyCode := "领航 VIP / 2026"
	anyName := "任意标识档"
	trigger := service.UserTierTriggerConsumption
	zero := 0.0
	created, err := svc.CreateTier(ctx, service.UserTierInput{
		Code: &anyCode, Name: &anyName, TriggerType: &trigger, ThresholdUSD: &zero,
	})
	require.NoError(t, err)
	require.Equal(t, anyCode, created.Code)

	other := seedUserWithEmail(t, db, 0)
	otherSvc := newServiceWithLookup(t, db, client, tierLookupStub{user: other})
	_, anyAssignment, err := otherSvc.AssignUserTierByEmail(ctx, other.Email, anyCode, "", 0)
	require.NoError(t, err)
	require.Equal(t, anyCode, anyAssignment.TierCode)
}

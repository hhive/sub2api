//go:build unit

package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// ---------- 手工指派：覆盖范围与领取 ----------

// 被指派档位及其之前的档位都视为已达成（因而可领取），之后的档位不受影响。
func TestUserTierAssignment_CoversAssignedTierAndEarlier(t *testing.T) {
	repo := newTierRepoStub()
	t1 := repo.addTier(balanceTier(0, "consume_500", 0, 500, 50))
	t2 := repo.addTier(balanceTier(0, "consume_1000", 1, 1000, 100))
	t3 := repo.addTier(balanceTier(0, "consume_1500", 2, 1500, 150))
	// 累计消费为 0：没有指派时一个档位都不该达成
	svc, lookup := newTierServiceWithLookup(repo, balanceUser(21, "vip@example.com"))

	before, err := svc.GetUserTierView(context.Background(), 21)
	require.NoError(t, err)
	for _, state := range before.Tiers {
		require.False(t, state.Achieved, "无指派且无消费时不应达成")
	}

	_, assignment, err := svc.AssignUserTierByEmail(context.Background(), "vip@example.com", t2.Code, "运营补偿", 42)
	require.NoError(t, err)
	require.Equal(t, UserTierAssignmentSourceAdmin, assignment.Source)
	require.Equal(t, int64(42), *assignment.AssignedBy)
	require.NotNil(t, lookup, "按邮箱解析走的是注入的 userLookup")

	view, err := svc.GetUserTierView(context.Background(), 21)
	require.NoError(t, err)
	require.True(t, view.Tiers[0].Achieved)
	require.True(t, view.Tiers[1].Achieved)
	require.False(t, view.Tiers[2].Achieved)
	require.True(t, view.Tiers[0].Claimable)
	require.True(t, view.Tiers[1].Claimable)
	require.False(t, view.Tiers[2].Claimable)
	require.NotNil(t, view.AssignedTier)
	require.Equal(t, t2.Code, view.AssignedTier.Code)

	// 逐档领取：指派档与其之前的档位都可领，超出覆盖范围的档位被拒
	for _, tierID := range []int64{t1.ID, t2.ID} {
		result, err := svc.ClaimTier(context.Background(), 21, tierID)
		require.NoError(t, err)
		require.False(t, result.AlreadyClaimed)
	}
	_, err = svc.ClaimTier(context.Background(), 21, t3.ID)
	require.ErrorIs(t, err, ErrUserTierNotAchieved)
	require.InDelta(t, 150, repo.balances[21], 1e-9, "只领到被覆盖的两档")
}

// 覆盖范围按档位顺序（sort_order）判定：调整顺序后覆盖范围随之移动。
func TestUserTierAssignment_FollowsSortOrder(t *testing.T) {
	repo := newTierRepoStub()
	assigned := repo.addTier(balanceTier(0, "consume_500", 0, 500, 50))
	other := repo.addTier(balanceTier(0, "consume_1000", 1, 1000, 100))
	svc, _ := newTierServiceWithLookup(repo, balanceUser(22, "order@example.com"))

	_, _, err := svc.AssignUserTierByEmail(context.Background(), "order@example.com", assigned.Code, "", 0)
	require.NoError(t, err)

	view, err := svc.GetUserTierView(context.Background(), 22)
	require.NoError(t, err)
	require.True(t, view.Tiers[0].Achieved)
	require.False(t, view.Tiers[1].Achieved, "排在指派档之后的档位不达成")

	// 管理员换序：把被指派档位挪到第二位
	repo.setSortOrder(assigned.ID, 1)
	repo.setSortOrder(other.ID, 0)

	view, err = svc.GetUserTierView(context.Background(), 22)
	require.NoError(t, err)
	require.True(t, view.Tiers[0].Achieved, "换序后原在后面的档位落进覆盖范围")
	require.True(t, view.Tiers[1].Achieved)
}

// 取消指派：未领取的档位回落，已领取的权益不回收。
func TestUserTierAssignment_UnassignRevertsClaimabilityWithoutRevokingGrants(t *testing.T) {
	repo := newTierRepoStub()
	t1 := repo.addTier(balanceTier(0, "consume_500", 0, 500, 50))
	t2 := repo.addTier(balanceTier(0, "consume_1000", 1, 1000, 100))
	svc, _ := newTierServiceWithLookup(repo, balanceUser(23, "revert@example.com"))

	_, _, err := svc.AssignUserTierByEmail(context.Background(), "revert@example.com", t2.Code, "", 0)
	require.NoError(t, err)
	_, err = svc.ClaimTier(context.Background(), 23, t1.ID)
	require.NoError(t, err)

	_, removed, err := svc.UnassignUserTierByEmail(context.Background(), "revert@example.com")
	require.NoError(t, err)
	require.True(t, removed)

	view, err := svc.GetUserTierView(context.Background(), 23)
	require.NoError(t, err)
	require.Nil(t, view.AssignedTier)
	require.True(t, view.Tiers[0].Claimed, "已领取的档位保持已领取")
	require.False(t, view.Tiers[1].Claimable, "未领取的档位回落为不可领取")
	require.InDelta(t, 50, repo.balances[23], 1e-9)

	// 幂等：再次取消不报错也不删除任何东西
	_, removed, err = svc.UnassignUserTierByEmail(context.Background(), "revert@example.com")
	require.NoError(t, err)
	require.False(t, removed)
}

// ---------- 手工指派：校验与拒绝路径 ----------

func TestAssignUserTierByEmail_RejectsNonConsumptionDisabledAndUnknownTargets(t *testing.T) {
	repo := newTierRepoStub()
	repo.addTier(UserTier{
		Code: "first_recharge", Name: "新客", TriggerType: UserTierTriggerFirstRecharge, Enabled: true,
	})
	disabled := balanceTier(0, "consume_500", 0, 500, 50)
	disabled.Enabled = false
	repo.addTier(disabled)
	svc, _ := newTierServiceWithLookup(repo, balanceUser(24, "target@example.com"))

	_, _, err := svc.AssignUserTierByEmail(context.Background(), "target@example.com", "first_recharge", "", 0)
	require.ErrorIs(t, err, ErrUserTierAssignmentTargetNotConsumption)

	_, _, err = svc.AssignUserTierByEmail(context.Background(), "target@example.com", "consume_500", "", 0)
	require.ErrorIs(t, err, ErrUserTierAssignmentTargetDisabled)

	_, _, err = svc.AssignUserTierByEmail(context.Background(), "target@example.com", "consume_999", "", 0)
	require.ErrorIs(t, err, ErrUserTierNotFound)

	require.Empty(t, repo.assignments, "被拒时不得落任何指派")
}

func TestAssignUserTierByEmail_RejectsUnknownAndAmbiguousEmail(t *testing.T) {
	repo := newTierRepoStub()
	repo.addTier(balanceTier(0, "consume_500", 0, 500, 50))

	// 邮箱不存在
	svc, _ := newTierServiceWithLookup(repo, balanceUser(25, "known@example.com"))
	_, _, err := svc.AssignUserTierByEmail(context.Background(), "nobody@example.com", "consume_500", "", 0)
	require.ErrorIs(t, err, ErrUserTierAssignUserNotFound)

	// 解析失败（既有多行匹配）：不得静默挑一个
	svcAmbiguous, _ := newTierServiceWithLookup(repo, balanceUser(25, "known@example.com"))
	svcAmbiguous.userLookup = &tierUserLookupStub{err: fmt.Errorf("normalized email lookup matched multiple users for %q", "dup@example.com")}
	_, _, err = svcAmbiguous.AssignUserTierByEmail(context.Background(), "dup@example.com", "consume_500", "", 0)
	require.ErrorIs(t, err, ErrUserTierAssignEmailAmbiguous)

	require.Empty(t, repo.assignments)
}

// 总开关关闭时拒绝写入：否则会留下「配置成功但用户端完全不可见」的静默状态。
func TestAssignUserTierByEmail_RejectedWhenFeatureDisabled(t *testing.T) {
	repo := newTierRepoStub()
	repo.addTier(balanceTier(0, "consume_500", 0, 500, 50))
	store := &userTierSettingRepoStub{values: map[string]string{"user_tier_enabled": "false"}}
	svc := NewUserTierService(repo, &tierCreditRepoStub{}, nil, nil, nil,
		&SettingService{settingRepo: store}, balanceUser(26, "off@example.com"))

	_, _, err := svc.AssignUserTierByEmail(context.Background(), "off@example.com", "consume_500", "", 0)
	require.ErrorIs(t, err, ErrUserTierFeatureDisabled)
	require.Empty(t, repo.assignments)
}

// 同一用户重指派即覆盖（一行），并按邮箱大小写/空白差异都能解析到同一用户。
func TestAssignUserTierByEmail_IsIdempotentAndOverwrites(t *testing.T) {
	repo := newTierRepoStub()
	t1 := repo.addTier(balanceTier(0, "consume_500", 0, 500, 50))
	t2 := repo.addTier(balanceTier(0, "consume_1000", 1, 1000, 100))
	svc, _ := newTierServiceWithLookup(repo, balanceUser(27, "Again@Example.com"))

	_, _, err := svc.AssignUserTierByEmail(context.Background(), "  again@example.com ", t1.Code, "第一次", 7)
	require.NoError(t, err)
	_, stored, err := svc.AssignUserTierByEmail(context.Background(), "AGAIN@example.com", t2.Code, "第二次", 8)
	require.NoError(t, err)

	require.Len(t, repo.assignments, 1, "一人一条指派")
	require.Equal(t, t2.Code, stored.TierCode)
	require.Equal(t, "第二次", stored.Note)
	require.Equal(t, int64(8), *stored.AssignedBy)

	user, assignment, err := svc.GetUserAssignmentByEmail(context.Background(), "again@example.com")
	require.NoError(t, err)
	require.Equal(t, int64(27), user.ID)
	require.Equal(t, t2.Code, assignment.TierCode)
}

// ---------- 等级标识（code）放开 ----------

func TestValidateTierCode_AllowsArbitraryValuesAndKeepsStructuralRules(t *testing.T) {
	repo := newTierRepoStub()
	svc := newTierServiceForTest(repo, nil)
	ctx := context.Background()

	// 任意字符（大小写、中文、空格、符号）都可作为标识
	for _, code := range []string{"VIP", "领航 档位-2026", "a.b/c", "1"} {
		threshold := 500.0
		created, err := svc.CreateTier(ctx, UserTierInput{
			Code:         &code,
			Name:         tierStrPtr("任意标识"),
			TriggerType:  tierStrPtr(UserTierTriggerConsumption),
			ThresholdUSD: &threshold,
		})
		require.NoError(t, err, "code=%q 应被接受", code)
		require.Equal(t, code, created.Code)
	}

	// 仍然非空、仍然受列宽限制（64 字符）
	empty := "  "
	_, err := svc.CreateTier(ctx, UserTierInput{
		Code: &empty, Name: tierStrPtr("空标识"), TriggerType: tierStrPtr(UserTierTriggerConsumption), ThresholdUSD: thresholdPtr(500),
	})
	require.ErrorIs(t, err, ErrUserTierInvalidConfig)

	tooLong := strings.Repeat("长", userTierCodeMaxRunes+1)
	_, err = svc.CreateTier(ctx, UserTierInput{
		Code: &tooLong, Name: tierStrPtr("超长标识"), TriggerType: tierStrPtr(UserTierTriggerConsumption), ThresholdUSD: thresholdPtr(500),
	})
	require.ErrorIs(t, err, ErrUserTierInvalidConfig)

	// 首充档仍必须使用固定标识（首充发放按它查找，改名会静默停发）
	_, err = svc.CreateTier(ctx, UserTierInput{
		Code: tierStrPtr("first-recharge-custom"), Name: tierStrPtr("首充档"), TriggerType: tierStrPtr(UserTierTriggerFirstRecharge),
	})
	require.ErrorIs(t, err, ErrUserTierFirstRechargeCodeLocked)
}

func balanceUser(id int64, email string) *tierUserLookupStub {
	return &tierUserLookupStub{
		users: map[string]User{strings.ToLower(strings.TrimSpace(email)): {ID: id, Email: email, Username: "tester"}},
	}
}

func newTierServiceWithLookup(repo *tierRepoStub, lookup UserTierUserLookup) (*UserTierService, *tierUserLookupStub) {
	stub, _ := lookup.(*tierUserLookupStub)
	return NewUserTierService(repo, &tierCreditRepoStub{}, nil, nil, nil, nil, lookup), stub
}

// setSortOrder 直接改档位顺序并保持列表按 (sort_order, id) 稳定有序，
// 用来模拟管理员在配置页换序（服务实现一律以该顺序推导覆盖范围）。
func (s *tierRepoStub) setSortOrder(id int64, order int) {
	for i := range s.tiers {
		if s.tiers[i].ID == id {
			s.tiers[i].SortOrder = order
		}
	}
	sort.SliceStable(s.tiers, func(i, j int) bool { return s.tiers[i].SortOrder < s.tiers[j].SortOrder })
}

func tierStrPtr(v string) *string { return &v }

func thresholdPtr(v float64) *float64 { return &v }

//go:build integration

// Package tierverify 用真实 PostgreSQL 验证用户等级功能的关键语义（事务、唯一约束幂等、
// 并发、冲突规则、回滚与阈值边界）——这些是 sqlmock/内存 stub 覆盖不到的部分。
//
// 与 internal/repository 的集成测试基座（testcontainers）分开成独立包，是为了不触发那套
// TestMain 的容器编排：本包只需要一个可写 PostgreSQL。
//
// 运行方式：
//
//	docker run -d --rm -e POSTGRES_PASSWORD=... -e POSTGRES_USER=tierv -e POSTGRES_DB=tierv -p 127.0.0.1:15499:5432 postgres:18-alpine
//	TIER_VERIFY_DSN="host=127.0.0.1 port=15499 user=tierv password=... dbname=tierv sslmode=disable" \
//	  go test -tags integration ./internal/tierverify
//
// TIER_VERIFY_DSN 未设置时全部用例跳过（不依赖 Docker 的机器上不会误报失败）。
package tierverify

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/service"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func openHarness(t *testing.T) (*sql.DB, *dbent.Client) {
	t.Helper()
	dsn := os.Getenv("TIER_VERIFY_DSN")
	if dsn == "" {
		t.Skip("TIER_VERIFY_DSN not set")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	require.NoError(t, db.Ping())

	// 空库跑全量迁移（含 241_add_user_tiers），随后再跑一次验证幂等
	require.NoError(t, repository.ApplyMigrations(context.Background(), db))
	require.NoError(t, repository.ApplyMigrations(context.Background(), db))

	// 每个用例独立：清空等级相关表（共享同一库）
	_, err = db.Exec(`
		TRUNCATE user_tier_rate_overlays, user_tier_effects, user_tier_awards, user_tier_benefits, user_tiers RESTART IDENTITY CASCADE
	`)
	require.NoError(t, err)

	drv, err := entsql.Open(dialect.Postgres, dsn)
	require.NoError(t, err)
	client := dbent.NewClient(dbent.Driver(drv))
	t.Cleanup(func() {
		_ = client.Close()
		_ = db.Close()
	})
	return db, client
}

var seedSeq int64

func seedUserAndGroup(t *testing.T, db *sql.DB) (int64, int64) {
	t.Helper()
	ctx := context.Background()
	seedSeq++
	suffix := fmt.Sprintf("%d-%d", time.Now().UnixNano(), seedSeq)

	var groupID int64
	require.NoError(t, db.QueryRowContext(ctx, `
		INSERT INTO groups (name, rate_multiplier, created_at, updated_at)
		VALUES ('tier-verify-group-' || $1, 1.2, NOW(), NOW()) RETURNING id
	`, suffix).Scan(&groupID))

	var userID int64
	require.NoError(t, db.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, role, status, balance, total_recharged, created_at, updated_at)
		VALUES ('tier-verify-' || $1 || '@example.test', 'x', 'user', 'active', 0, 0, NOW(), NOW()) RETURNING id
	`, suffix).Scan(&userID))

	// 累计消费只算 redeem 来源的 amount - remaining_amount
	_, err := db.ExecContext(ctx, `
		INSERT INTO user_balance_credits (user_id, email, source_type, source_id, source_code, amount, remaining_amount, status, created_at, updated_at)
		VALUES ($1, 'tier-verify-' || $2 || '@example.test', 'redeem', '1', 'CODE', 1500, 100, 'active', NOW(), NOW())
	`, userID, suffix)
	require.NoError(t, err)
	return userID, groupID
}

func newService(t *testing.T, db *sql.DB, client *dbent.Client) *service.UserTierService {
	t.Helper()
	repo := repository.NewUserTierRepository(client, db)
	creditRepo := repository.NewBalanceCreditRepository(client, db)
	return service.NewUserTierService(repo, creditRepo, client, nil, nil, nil)
}

func seedTiers(t *testing.T, svc *service.UserTierService, groupID int64) {
	t.Helper()
	ctx := context.Background()
	code := "consume_500"
	name := "常客"
	trigger := service.UserTierTriggerConsumption
	threshold := 500.0
	benefits := []service.UserTierBenefitInput{
		{BenefitType: service.UserTierBenefitBalanceCredit, Amount: 50, ValidityDays: 3, Enabled: true},
	}
	_, err := svc.CreateTier(ctx, service.UserTierInput{
		Code: &code, Name: &name, TriggerType: &trigger, ThresholdUSD: &threshold, Benefits: &benefits,
	})
	require.NoError(t, err)

	highCode := "consume_1400"
	highName := "企业专线"
	highThreshold := 1400.0
	highBenefits := []service.UserTierBenefitInput{
		{BenefitType: service.UserTierBenefitBalanceCredit, Amount: 150, ValidityDays: 30, Enabled: true},
		{BenefitType: service.UserTierBenefitGroupRate, GroupID: groupID, RateMultiplier: 0.9, Enabled: true},
	}
	_, err = svc.CreateTier(ctx, service.UserTierInput{
		Code: &highCode, Name: &highName, TriggerType: &trigger, ThresholdUSD: &highThreshold, Benefits: &highBenefits,
	})
	require.NoError(t, err)
}

func tierList(t *testing.T, svc *service.UserTierService) []service.UserTier {
	t.Helper()
	tiers, err := svc.ListTiers(context.Background(), true)
	require.NoError(t, err)
	return tiers
}

func findTier(t *testing.T, tiers []service.UserTier, code string) service.UserTier {
	t.Helper()
	for _, tier := range tiers {
		if tier.Code == code {
			return tier
		}
	}
	t.Fatalf("tier %s not found", code)
	return service.UserTier{}
}

// ---------- 1. 迁移与配置 CRUD ----------

func TestVerify_MigrationAndTierCRUD(t *testing.T) {
	db, client := openHarness(t)
	svc := newService(t, db, client)
	_, groupID := seedUserAndGroup(t, db)
	seedTiers(t, svc, groupID)

	tiers := tierList(t, svc)
	require.Len(t, tiers, 2)
	require.Equal(t, 0, tiers[0].SortOrder)
	require.Equal(t, 1, tiers[1].SortOrder)

	// 重排（真实的 unnest 批量 UPDATE）
	require.NoError(t, svc.ReorderTiers(context.Background(), []int64{tiers[1].ID, tiers[0].ID}))
	reordered := tierList(t, svc)
	require.Equal(t, tiers[1].Code, reordered[0].Code)

	// 权益参数校验：group_id 不存在必须被拒，且不落库
	badCode := "bad_group"
	badName := "坏分组"
	trigger := service.UserTierTriggerConsumption
	zero := 0.0
	bad := []service.UserTierBenefitInput{{BenefitType: service.UserTierBenefitGroupRate, GroupID: 999999, RateMultiplier: 0.9, Enabled: true}}
	_, err := svc.CreateTier(context.Background(), service.UserTierInput{
		Code: &badCode, Name: &badName, TriggerType: &trigger, ThresholdUSD: &zero, Benefits: &bad,
	})
	require.ErrorIs(t, err, service.ErrUserTierInvalidBenefit)

	var badCount int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM user_tiers WHERE code = 'bad_group'`).Scan(&badCount))
	require.Zero(t, badCount)

	// 唯一 code 冲突 → 领域错误
	dup := tiers[0].Code
	_, err = svc.CreateTier(context.Background(), service.UserTierInput{Code: &dup, Name: &badName, TriggerType: &trigger, ThresholdUSD: &zero})
	require.ErrorIs(t, err, service.ErrUserTierCodeExists)
}

// ---------- 2. 领取：真事务 + 唯一约束幂等 ----------

func TestVerify_ClaimFlowOnRealDB(t *testing.T) {
	db, client := openHarness(t)
	svc := newService(t, db, client)
	userID, groupID := seedUserAndGroup(t, db)
	seedTiers(t, svc, groupID)
	tiers := tierList(t, svc)
	low := findTier(t, tiers, "consume_500")
	high := findTier(t, tiers, "consume_1400")
	ctx := context.Background()

	// 累计消费 = 1500-100 = 1400：两档都达标
	view, err := svc.GetUserTierView(ctx, userID)
	require.NoError(t, err)
	require.InDelta(t, 1400, view.ConsumedAmount, 1e-9)
	require.Equal(t, 2, view.ClaimableCount)

	// 逐档补领（不是只有最高档能领）
	for _, tier := range []service.UserTier{low, high} {
		result, err := svc.ClaimTier(ctx, userID, tier.ID)
		require.NoError(t, err)
		require.False(t, result.AlreadyClaimed)
	}

	var balance, totalRecharged float64
	require.NoError(t, db.QueryRow(`SELECT balance, total_recharged FROM users WHERE id = $1`, userID).Scan(&balance, &totalRecharged))
	require.InDelta(t, 200, balance, 1e-9)
	require.Zero(t, totalRecharged, "等级奖励不得累加 total_recharged")

	var tierRewardRows int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM user_balance_credits WHERE user_id = $1 AND source_type = 'tier_reward'`, userID).Scan(&tierRewardRows))
	require.Equal(t, 2, tierRewardRows)

	// 倍率写进覆盖层，且不触碰手工倍率表
	var overlayRate float64
	require.NoError(t, db.QueryRow(`SELECT rate_multiplier FROM user_tier_rate_overlays WHERE user_id = $1 AND group_id = $2 AND status = 'active'`, userID, groupID).Scan(&overlayRate))
	require.InDelta(t, 0.9, overlayRate, 1e-9)
	var manualRows int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM user_group_rate_multipliers WHERE user_id = $1`, userID).Scan(&manualRows))
	require.Zero(t, manualRows)

	// 计费热路径的读取合并：等级倍率生效
	rateRepo := repository.NewUserGroupRateRepository(db)
	effective, err := rateRepo.GetByUserAndGroup(ctx, userID, groupID)
	require.NoError(t, err)
	require.NotNil(t, effective)
	require.InDelta(t, 0.9, *effective, 1e-9)

	// 幂等：重复领取返回已领取，且账本/overlay/effect 均不新增
	result, err := svc.ClaimTier(ctx, userID, high.ID)
	require.NoError(t, err)
	require.True(t, result.AlreadyClaimed)
	require.Empty(t, result.Applied)

	var awards, effects, credits int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM user_tier_awards WHERE user_id = $1`, userID).Scan(&awards))
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM user_tier_effects WHERE user_id = $1`, userID).Scan(&effects))
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM user_balance_credits WHERE user_id = $1 AND source_type = 'tier_reward'`, userID).Scan(&credits))
	require.Equal(t, 2, awards)
	require.Equal(t, 3, effects) // 2 档额度 + 1 项倍率
	require.Equal(t, 2, credits)
	require.NoError(t, db.QueryRow(`SELECT balance FROM users WHERE id = $1`, userID).Scan(&balance))
	require.InDelta(t, 200, balance, 1e-9)
}

// ---------- 3. 并发领取：唯一约束兜底 ----------

func TestVerify_ConcurrentClaimOnRealDB(t *testing.T) {
	db, client := openHarness(t)
	svc := newService(t, db, client)
	userID, groupID := seedUserAndGroup(t, db)
	seedTiers(t, svc, groupID)
	tiers := tierList(t, svc)
	low := findTier(t, tiers, "consume_500")
	ctx := context.Background()

	const workers = 2
	var wg sync.WaitGroup
	errorsByWorker := make([]error, workers)
	alreadyClaimed := make([]bool, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			result, err := svc.ClaimTier(ctx, userID, low.ID)
			errorsByWorker[index] = err
			if result != nil {
				alreadyClaimed[index] = result.AlreadyClaimed
			}
		}(i)
	}
	wg.Wait()
	for _, err := range errorsByWorker {
		require.NoError(t, err)
	}
	// 恰好一次真实发放：一个 created，另一个（或并发冲突后）得到已领取
	require.Equal(t, 1, countTrue(alreadyClaimed))

	var awards, credits, overlays int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM user_tier_awards WHERE user_id = $1`, userID).Scan(&awards))
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM user_balance_credits WHERE user_id = $1 AND source_type = 'tier_reward'`, userID).Scan(&credits))
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM user_tier_rate_overlays WHERE user_id = $1`, userID).Scan(&overlays))
	require.Equal(t, 1, awards)
	require.Equal(t, 1, credits)
	var balance float64
	require.NoError(t, db.QueryRow(`SELECT balance FROM users WHERE id = $1`, userID).Scan(&balance))
	require.InDelta(t, 50, balance, 1e-9)
}

func countTrue(values []bool) int {
	n := 0
	for _, v := range values {
		if v {
			n++
		}
	}
	return n
}

// ---------- 4. 手工倍率冲突规则 ----------

func TestVerify_ManualRateWinsOverTierOverlay(t *testing.T) {
	db, client := openHarness(t)
	svc := newService(t, db, client)
	userID, groupID := seedUserAndGroup(t, db)
	seedTiers(t, svc, groupID)
	tiers := tierList(t, svc)
	high := findTier(t, tiers, "consume_1400")
	ctx := context.Background()

	// 该用户已有手工倍率 0.16
	_, err := db.ExecContext(ctx, `
		INSERT INTO user_group_rate_multipliers (user_id, group_id, rate_multiplier, created_at, updated_at)
		VALUES ($1, $2, 0.16, NOW(), NOW())
	`, userID, groupID)
	require.NoError(t, err)

	result, err := svc.ClaimTier(ctx, userID, high.ID)
	require.NoError(t, err)

	// 倍率不写覆盖层；额度照发
	var overlays int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM user_tier_rate_overlays WHERE user_id = $1`, userID).Scan(&overlays))
	require.Zero(t, overlays)
	var balance float64
	require.NoError(t, db.QueryRow(`SELECT balance FROM users WHERE id = $1`, userID).Scan(&balance))
	require.InDelta(t, 150, balance, 1e-9)

	// effect detail 记录跳过原因
	var detail string
	require.NoError(t, db.QueryRow(`
		SELECT detail::text FROM user_tier_effects
		WHERE user_id = $1 AND benefit_type = 'group_rate'
	`, userID).Scan(&detail))
	require.Contains(t, detail, service.UserTierSkipManualRateMultiplier)

	// 手工值仍然是生效倍率
	rateRepo := repository.NewUserGroupRateRepository(db)
	effective, err := rateRepo.GetByUserAndGroup(ctx, userID, groupID)
	require.NoError(t, err)
	require.InDelta(t, 0.16, *effective, 1e-9)

	require.Len(t, result.Applied, 2)
}

// ---------- 5. 失败回滚（半成品防护） ----------

func TestVerify_ClaimRollbackLeavesNoHalfState(t *testing.T) {
	db, client := openHarness(t)
	svc := newService(t, db, client)
	userID, groupID := seedUserAndGroup(t, db)
	seedTiers(t, svc, groupID)
	tiers := tierList(t, svc)
	low := findTier(t, tiers, "consume_500")
	ctx := context.Background()

	// 制造必然失败的发放：把 user_balance_credits 的 amount>0 约束用作触发器——
	// 删除目标用户，使 ApplyTierRewardBalance 影响 0 行（返回 ErrUserNotFound），从而整事务回滚。
	// 为保留 redeem 账本行（口径依据），改为让 users 行不可见：软删除。
	_, err := db.ExecContext(ctx, `UPDATE users SET deleted_at = NOW() WHERE id = $1`, userID)
	require.NoError(t, err)

	_, err = svc.ClaimTier(ctx, userID, low.ID)
	require.Error(t, err)

	var awards, effects, credits int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM user_tier_awards WHERE user_id = $1`, userID).Scan(&awards))
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM user_tier_effects WHERE user_id = $1`, userID).Scan(&effects))
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM user_balance_credits WHERE user_id = $1 AND source_type = 'tier_reward'`, userID).Scan(&credits))
	require.Zero(t, awards, "失败必须整体回滚：不留下已获得的等级")
	require.Zero(t, effects)
	require.Zero(t, credits)

	// 恢复后可重试成功
	_, err = db.ExecContext(ctx, `UPDATE users SET deleted_at = NULL WHERE id = $1`, userID)
	require.NoError(t, err)
	retry, err := svc.ClaimTier(ctx, userID, low.ID)
	require.NoError(t, err)
	require.False(t, retry.AlreadyClaimed)
	var balance float64
	require.NoError(t, db.QueryRow(`SELECT balance FROM users WHERE id = $1`, userID).Scan(&balance))
	require.InDelta(t, 50, balance, 1e-9)
}

// ---------- 6. 首充单源读取（真库） ----------

func TestVerify_FirstRechargeSingleSource(t *testing.T) {
	db, client := openHarness(t)
	svc := newService(t, db, client)

	// 未导入首充档时：返回错误（调用方据此回退 settings）
	_, err := svc.ResolveFirstRechargeGrant(context.Background())
	require.ErrorIs(t, err, service.ErrUserTierNotFound)

	code := service.UserTierCodeFirstRecharge
	name := "新朋友"
	trigger := service.UserTierTriggerFirstRecharge
	enabled := false
	benefits := []service.UserTierBenefitInput{{BenefitType: service.UserTierBenefitBalanceCredit, Amount: 5, ValidityDays: 3, Enabled: true}}
	_, err = svc.CreateTier(context.Background(), service.UserTierInput{
		Code: &code, Name: &name, TriggerType: &trigger, Enabled: &enabled, Benefits: &benefits,
	})
	require.NoError(t, err)

	grant, err := svc.ResolveFirstRechargeGrant(context.Background())
	require.NoError(t, err)
	require.False(t, grant.Enabled)
	require.InDelta(t, 5, grant.Amount, 1e-9)
	require.Equal(t, 3, grant.ValidityDays)

	// 阈值约束：首充档不允许带阈值（数据库 CHECK 兜底）
	badThreshold := 100.0
	badCode := "first_recharge_bad"
	_, err = svc.CreateTier(context.Background(), service.UserTierInput{
		Code: &badCode, Name: &name, TriggerType: &trigger, ThresholdUSD: &badThreshold,
	})
	require.NoError(t, err, "服务层会清空首充档阈值后再落库")
	var stored *float64
	require.NoError(t, db.QueryRow(`SELECT threshold_usd FROM user_tiers WHERE code = $1`, badCode).Scan(&stored))
	require.Nil(t, stored)
}

// ---------- 7. 停用档位与删除保护 ----------

func TestVerify_DisabledTierAndDeleteProtection(t *testing.T) {
	db, client := openHarness(t)
	svc := newService(t, db, client)
	userID, groupID := seedUserAndGroup(t, db)
	seedTiers(t, svc, groupID)
	tiers := tierList(t, svc)
	low := findTier(t, tiers, "consume_500")
	ctx := context.Background()

	_, err := svc.ClaimTier(ctx, userID, low.ID)
	require.NoError(t, err)

	// 已产生授予 → 只能停用
	require.ErrorIs(t, svc.DeleteTier(ctx, low.ID), service.ErrUserTierHasAwards)
	disabled, err := svc.SetTierEnabled(ctx, low.ID, false)
	require.NoError(t, err)
	require.False(t, disabled.Enabled)

	// 停用档位不再出现在用户端视图与可领取列表
	view, err := svc.GetUserTierView(ctx, userID)
	require.NoError(t, err)
	for _, state := range view.Tiers {
		require.NotEqual(t, low.Code, state.Code)
	}

	// 未产生授予的档位可以删除
	spareCode := "spare"
	spareName := "备用"
	trigger := service.UserTierTriggerConsumption
	threshold := 2000.0
	spare, err := svc.CreateTier(ctx, service.UserTierInput{Code: &spareCode, Name: &spareName, TriggerType: &trigger, ThresholdUSD: &threshold})
	require.NoError(t, err)
	require.NoError(t, svc.DeleteTier(ctx, spare.ID))

	var remaining int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM user_tiers WHERE code = 'spare'`).Scan(&remaining))
	require.Zero(t, remaining)
}

// ---------- 8. 阈值边界（真库视图） ----------

func TestVerify_ThresholdBoundariesOnRealDB(t *testing.T) {
	db, client := openHarness(t)
	svc := newService(t, db, client)
	userID, groupID := seedUserAndGroup(t, db)
	seedTiers(t, svc, groupID)

	// 把 redeem 账本的已消费额精确调到 500（amount 1500 / remaining 1000）
	_, err := db.ExecContext(context.Background(), `UPDATE user_balance_credits SET remaining_amount = 1000 WHERE user_id = $1`, userID)
	require.NoError(t, err)

	view, err := svc.GetUserTierView(context.Background(), userID)
	require.NoError(t, err)
	require.InDelta(t, 500, view.ConsumedAmount, 1e-9)
	require.Equal(t, 1, view.ClaimableCount, "恰好等于阈值必须视为达成")
	require.Nil(t, view.CurrentTier)
	require.NotNil(t, view.NextTier)
	require.InDelta(t, 900, *view.RemainingToNext, 1e-9)
	require.Equal(t, groupID, groupID) // 保持 groupID 被使用
	fmt.Println("threshold boundary verified")
}

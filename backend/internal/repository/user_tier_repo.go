package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// userTierSQL 只依赖 Exec/Query，故 *sql.DB 与 ent.Tx（经 dbent.TxFromContext）都能作为执行器，
// 等级与权益、授予与发放记录因此可以在同一个 Ent 事务内写入。
type userTierSQL interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type userTierRepository struct {
	client *dbent.Client
	db     *sql.DB
}

// NewUserTierRepository 创建用户等级仓储（raw SQL，与 user_group_rate_repo.go 同风格）
func NewUserTierRepository(client *dbent.Client, db *sql.DB) service.UserTierRepository {
	return &userTierRepository{client: client, db: db}
}

func (r *userTierRepository) execer(ctx context.Context) (userTierSQL, error) {
	if r != nil && r.client != nil {
		if tx := dbent.TxFromContext(ctx); tx != nil {
			return tx.Client(), nil
		}
	}
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("user tier repository db is nil")
	}
	return r.db, nil
}

const userTierColumns = `t.id, t.code, t.name, t.description, t.sort_order, t.trigger_type,
	t.threshold_usd, t.enabled, t.created_at, t.updated_at,
	(SELECT count(*) FROM user_tier_awards a WHERE a.tier_code = t.code) AS award_count`

func scanUserTier(scan func(dest ...any) error) (*service.UserTier, error) {
	var tier service.UserTier
	var threshold sql.NullFloat64
	if err := scan(
		&tier.ID, &tier.Code, &tier.Name, &tier.Description, &tier.SortOrder, &tier.TriggerType,
		&threshold, &tier.Enabled, &tier.CreatedAt, &tier.UpdatedAt, &tier.AwardCount,
	); err != nil {
		return nil, err
	}
	if threshold.Valid {
		v := threshold.Float64
		tier.ThresholdUSD = &v
	}
	return &tier, nil
}

func scanUserTierBenefit(scan func(dest ...any) error) (*service.UserTierBenefit, error) {
	var (
		benefit   service.UserTierBenefit
		rawParams []byte
	)
	if err := scan(&benefit.ID, &benefit.TierID, &benefit.BenefitType, &rawParams, &benefit.Enabled, &benefit.SortOrder, &benefit.CreatedAt, &benefit.UpdatedAt); err != nil {
		return nil, err
	}
	if err := applyBenefitParams(&benefit, rawParams); err != nil {
		return nil, err
	}
	return &benefit, nil
}

// applyBenefitParams 按 benefit_type 把 jsonb 参数解析为强类型（参数校验在服务层做，此处只做解析）
func applyBenefitParams(benefit *service.UserTierBenefit, raw []byte) error {
	if len(raw) == 0 {
		return nil
	}
	switch benefit.BenefitType {
	case service.UserTierBenefitBalanceCredit:
		var params service.UserTierBalanceCreditParams
		if err := json.Unmarshal(raw, &params); err != nil {
			return fmt.Errorf("parse balance_credit params (benefit %d): %w", benefit.ID, err)
		}
		benefit.BalanceCredit = &params
	case service.UserTierBenefitGroupRate:
		var params service.UserTierGroupRateParams
		if err := json.Unmarshal(raw, &params); err != nil {
			return fmt.Errorf("parse group_rate params (benefit %d): %w", benefit.ID, err)
		}
		benefit.GroupRate = &params
	}
	return nil
}

func benefitParamsJSON(input service.UserTierBenefitInput) ([]byte, error) {
	switch input.BenefitType {
	case service.UserTierBenefitBalanceCredit:
		return json.Marshal(service.UserTierBalanceCreditParams{
			Amount:       input.Amount,
			ValidityDays: input.ValidityDays,
		})
	case service.UserTierBenefitGroupRate:
		return json.Marshal(service.UserTierGroupRateParams{
			GroupID:        input.GroupID,
			RateMultiplier: input.RateMultiplier,
		})
	default:
		return nil, service.ErrUserTierInvalidBenefit
	}
}

// loadBenefits 为给定等级批量装载权益（按 sort_order 稳定排序）
func (r *userTierRepository) loadBenefits(ctx context.Context, exec userTierSQL, tierIDs []int64) (map[int64][]service.UserTierBenefit, error) {
	result := make(map[int64][]service.UserTierBenefit, len(tierIDs))
	if len(tierIDs) == 0 {
		return result, nil
	}
	rows, err := exec.QueryContext(ctx, `
		SELECT id, tier_id, benefit_type, params, enabled, sort_order, created_at, updated_at
		FROM user_tier_benefits
		WHERE tier_id = ANY($1)
		ORDER BY tier_id, sort_order, id
	`, pq.Array(tierIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		benefit, err := scanUserTierBenefit(rows.Scan)
		if err != nil {
			return nil, err
		}
		result[benefit.TierID] = append(result[benefit.TierID], *benefit)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *userTierRepository) ListTiers(ctx context.Context, includeDisabled bool) ([]service.UserTier, error) {
	exec, err := r.execer(ctx)
	if err != nil {
		return nil, err
	}
	query := `SELECT ` + userTierColumns + ` FROM user_tiers t`
	if !includeDisabled {
		query += ` WHERE t.enabled`
	}
	query += ` ORDER BY t.sort_order, t.id`

	rows, err := exec.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var tiers []service.UserTier
	var ids []int64
	for rows.Next() {
		tier, err := scanUserTier(rows.Scan)
		if err != nil {
			return nil, err
		}
		tiers = append(tiers, *tier)
		ids = append(ids, tier.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	benefits, err := r.loadBenefits(ctx, exec, ids)
	if err != nil {
		return nil, err
	}
	for i := range tiers {
		tiers[i].Benefits = benefits[tiers[i].ID]
	}
	return tiers, nil
}

func (r *userTierRepository) GetTierByID(ctx context.Context, id int64) (*service.UserTier, error) {
	if id <= 0 {
		return nil, service.ErrUserTierNotFound
	}
	exec, err := r.execer(ctx)
	if err != nil {
		return nil, err
	}
	tier, err := scanUserTier(func(dest ...any) error {
		return scanSingleRow(ctx, exec, `SELECT `+userTierColumns+` FROM user_tiers t WHERE t.id = $1`, []any{id}, dest...)
	})
	if err == sql.ErrNoRows {
		return nil, service.ErrUserTierNotFound
	}
	if err != nil {
		return nil, err
	}
	benefits, err := r.loadBenefits(ctx, exec, []int64{tier.ID})
	if err != nil {
		return nil, err
	}
	tier.Benefits = benefits[tier.ID]
	return tier, nil
}

func (r *userTierRepository) GetTierByCode(ctx context.Context, code string) (*service.UserTier, error) {
	exec, err := r.execer(ctx)
	if err != nil {
		return nil, err
	}
	tier, err := scanUserTier(func(dest ...any) error {
		return scanSingleRow(ctx, exec, `SELECT `+userTierColumns+` FROM user_tiers t WHERE t.code = $1`, []any{code}, dest...)
	})
	if err == sql.ErrNoRows {
		return nil, service.ErrUserTierNotFound
	}
	if err != nil {
		return nil, err
	}
	benefits, err := r.loadBenefits(ctx, exec, []int64{tier.ID})
	if err != nil {
		return nil, err
	}
	tier.Benefits = benefits[tier.ID]
	return tier, nil
}

// CreateTier 等级与权益同事务写入，避免出现「等级存了权益没存」的中间态。
func (r *userTierRepository) CreateTier(ctx context.Context, tier *service.UserTier) error {
	if r.client == nil {
		return fmt.Errorf("user tier repository client is nil")
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var tierID int64
	err = scanSingleRow(ctx, tx.Client(), `
		INSERT INTO user_tiers (code, name, description, sort_order, trigger_type, threshold_usd, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id
	`, []any{tier.Code, tier.Name, tier.Description, tier.SortOrder, tier.TriggerType, tier.ThresholdUSD, tier.Enabled}, &tierID)
	if err != nil {
		return translatePersistenceError(err, nil, service.ErrUserTierCodeExists)
	}
	tier.ID = tierID

	for i := range tier.Benefits {
		params, err := benefitParamsJSON(service.UserTierBenefitInput{
			BenefitType:    tier.Benefits[i].BenefitType,
			Amount:         balanceAmountOf(tier.Benefits[i]),
			ValidityDays:   balanceValidityDaysOf(tier.Benefits[i]),
			GroupID:        groupIDOf(tier.Benefits[i]),
			RateMultiplier: groupRateOf(tier.Benefits[i]),
		})
		if err != nil {
			return err
		}
		var benefitID int64
		if err := scanSingleRow(ctx, tx.Client(), `
			INSERT INTO user_tier_benefits (tier_id, benefit_type, params, enabled, sort_order, created_at, updated_at)
			VALUES ($1, $2, $3::jsonb, $4, $5, NOW(), NOW())
			RETURNING id
		`, []any{tierID, tier.Benefits[i].BenefitType, string(params), tier.Benefits[i].Enabled, tier.Benefits[i].SortOrder}, &benefitID); err != nil {
			return err
		}
		tier.Benefits[i].ID = benefitID
		tier.Benefits[i].TierID = tierID
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

// UpdateTier 等级与权益同事务写入。Benefits 非 nil 时整体替换该档权益列表：
// 保留输入中带 ID 的既有行（benefit_key 因此稳定，已发 effect 不受影响），删除未出现的行，新增 ID 为 0 的行。
func (r *userTierRepository) UpdateTier(ctx context.Context, tier *service.UserTier) error {
	if r.client == nil {
		return fmt.Errorf("user tier repository client is nil")
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// code 必须一起写：服务层在无授予记录时允许改名，漏写会让「改名成功」变成静默不生效
	// （响应体返回新 code、库里仍是旧 code）。改名的前置校验见 UserTierService.UpdateTier。
	if _, err := tx.Client().ExecContext(ctx, `
		UPDATE user_tiers
		SET name = $2, description = $3, trigger_type = $4, threshold_usd = $5, enabled = $6, code = $7, updated_at = NOW()
		WHERE id = $1
	`, tier.ID, tier.Name, tier.Description, tier.TriggerType, tier.ThresholdUSD, tier.Enabled, tier.Code); err != nil {
		return err
	}

	if tier.Benefits != nil {
		keepIDs := make([]int64, 0, len(tier.Benefits))
		for i := range tier.Benefits {
			params, err := benefitParamsJSON(service.UserTierBenefitInput{
				BenefitType:    tier.Benefits[i].BenefitType,
				Amount:         balanceAmountOf(tier.Benefits[i]),
				ValidityDays:   balanceValidityDaysOf(tier.Benefits[i]),
				GroupID:        groupIDOf(tier.Benefits[i]),
				RateMultiplier: groupRateOf(tier.Benefits[i]),
			})
			if err != nil {
				return err
			}
			if tier.Benefits[i].ID > 0 {
				keepIDs = append(keepIDs, tier.Benefits[i].ID)
				if _, err := tx.Client().ExecContext(ctx, `
					UPDATE user_tier_benefits
					SET benefit_type = $3, params = $4::jsonb, enabled = $5, sort_order = $6, updated_at = NOW()
					WHERE id = $1 AND tier_id = $2
				`, tier.Benefits[i].ID, tier.ID, tier.Benefits[i].BenefitType, string(params), tier.Benefits[i].Enabled, tier.Benefits[i].SortOrder); err != nil {
					return err
				}
				continue
			}
			if err := scanSingleRow(ctx, tx.Client(), `
				INSERT INTO user_tier_benefits (tier_id, benefit_type, params, enabled, sort_order, created_at, updated_at)
				VALUES ($1, $2, $3::jsonb, $4, $5, NOW(), NOW())
				RETURNING id
			`, []any{tier.ID, tier.Benefits[i].BenefitType, string(params), tier.Benefits[i].Enabled, tier.Benefits[i].SortOrder}, &tier.Benefits[i].ID); err != nil {
				return err
			}
			keepIDs = append(keepIDs, tier.Benefits[i].ID)
		}
		if len(keepIDs) == 0 {
			if _, err := tx.Client().ExecContext(ctx, `DELETE FROM user_tier_benefits WHERE tier_id = $1`, tier.ID); err != nil {
				return err
			}
		} else if _, err := tx.Client().ExecContext(ctx,
			`DELETE FROM user_tier_benefits WHERE tier_id = $1 AND id <> ALL($2)`, tier.ID, pq.Array(keepIDs)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *userTierRepository) ReorderTiers(ctx context.Context, orderedIDs []int64) error {
	if len(orderedIDs) == 0 {
		return nil
	}
	exec, err := r.execer(ctx)
	if err != nil {
		return err
	}
	// 用 int64 承载序号：pq.Array 对 []int32 的支持未在 lib/pq 中显式保证，交由 SQL 侧转型
	orders := make([]int64, len(orderedIDs))
	for i := range orderedIDs {
		orders[i] = int64(i)
	}
	result, err := exec.ExecContext(ctx, `
		UPDATE user_tiers AS t
		SET sort_order = data.sort_order, updated_at = NOW()
		FROM unnest($1::bigint[], $2::bigint[]) AS data(id, sort_order)
		WHERE t.id = data.id
	`, pq.Array(orderedIDs), pq.Array(orders))
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if int(affected) != len(orderedIDs) {
		return fmt.Errorf("reorder tiers: %d of %d ids matched", affected, len(orderedIDs))
	}
	return nil
}

// DeleteTier 只允许删除从未产生授予记录的等级；已产生授予的等级只能停用。
func (r *userTierRepository) DeleteTier(ctx context.Context, id int64) error {
	exec, err := r.execer(ctx)
	if err != nil {
		return err
	}
	var code string
	if err := scanSingleRow(ctx, exec, `SELECT code FROM user_tiers WHERE id = $1`, []any{id}, &code); err != nil {
		if err == sql.ErrNoRows {
			return service.ErrUserTierNotFound
		}
		return err
	}
	count, err := r.CountAwardsByTierCode(ctx, code)
	if err != nil {
		return err
	}
	if count > 0 {
		return service.ErrUserTierHasAwards
	}
	// 指派同样以档位为锚：删掉配置会让已指派用户的阶梯位置失去依据（tier_id 被置空后
	// 只能退回顺序快照），因此有指派的档位也只能停用。
	assigned, err := r.CountAssignmentsByTierID(ctx, id)
	if err != nil {
		return err
	}
	if assigned > 0 {
		return service.ErrUserTierHasAssignments
	}
	_, err = exec.ExecContext(ctx, `DELETE FROM user_tiers WHERE id = $1`, id)
	return err
}

func (r *userTierRepository) CountAwardsByTierID(ctx context.Context, tierID int64) (int64, error) {
	exec, err := r.execer(ctx)
	if err != nil {
		return 0, err
	}
	var code string
	if err := scanSingleRow(ctx, exec, `SELECT code FROM user_tiers WHERE id = $1`, []any{tierID}, &code); err != nil {
		if err == sql.ErrNoRows {
			return 0, service.ErrUserTierNotFound
		}
		return 0, err
	}
	return r.CountAwardsByTierCode(ctx, code)
}

func (r *userTierRepository) CountAwardsByTierCode(ctx context.Context, code string) (int64, error) {
	exec, err := r.execer(ctx)
	if err != nil {
		return 0, err
	}
	var count int64
	if err := scanSingleRow(ctx, exec, `SELECT count(*) FROM user_tier_awards WHERE tier_code = $1`, []any{code}, &count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *userTierRepository) GroupExists(ctx context.Context, groupID int64) (bool, error) {
	exec, err := r.execer(ctx)
	if err != nil {
		return false, err
	}
	var exists bool
	if err := scanSingleRow(ctx, exec, `SELECT EXISTS(SELECT 1 FROM groups WHERE id = $1)`, []any{groupID}, &exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (r *userTierRepository) GroupName(ctx context.Context, groupID int64) (string, error) {
	exec, err := r.execer(ctx)
	if err != nil {
		return "", err
	}
	var name string
	if err := scanSingleRow(ctx, exec, `SELECT name FROM groups WHERE id = $1`, []any{groupID}, &name); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return name, nil
}

// GetUserConsumedAmount 累计消费额：只算兑换额度消费（额度账本是终身账，不受日志保留影响）。
//
// 必须减去 expired_amount：额度到期作废时到期任务把 remaining_amount 置 0，若只算
// amount - remaining_amount，未使用的部分会被整额算成「已消费」——用户凭一张过期作废的
// 兑换码即可达标领钱（1500 档 = 赠 150 美元 + 0.9 倍率）。expired_amount 记的正是
// 「到期那一刻仍未使用的余额」，减去它得到的才是真实消费额。
// 非 expired 行该列恒为 NULL，行为与加列前逐位一致。
func (r *userTierRepository) GetUserConsumedAmount(ctx context.Context, userID int64) (float64, error) {
	if userID <= 0 {
		return 0, nil
	}
	exec, err := r.execer(ctx)
	if err != nil {
		return 0, err
	}
	var consumed float64
	if err := scanSingleRow(ctx, exec, `
		SELECT COALESCE(sum(amount - remaining_amount - COALESCE(expired_amount, 0)), 0)
		FROM user_balance_credits
		WHERE user_id = $1 AND source_type = 'redeem'
	`, []any{userID}, &consumed); err != nil {
		return 0, err
	}
	return consumed, nil
}

// GetFirstRechargeCredit 读取既有首充奖励账本记录。
// 首充档的「已发放」状态由账本本身判定，因此 433 名已发用户零回填（设计第 6、9 节）。
func (r *userTierRepository) GetFirstRechargeCredit(ctx context.Context, userID int64) (*service.UserTierFirstRechargeCredit, error) {
	if userID <= 0 {
		return nil, nil
	}
	exec, err := r.execer(ctx)
	if err != nil {
		return nil, err
	}
	var (
		credit    service.UserTierFirstRechargeCredit
		expiresAt sql.NullTime
	)
	err = scanSingleRow(ctx, exec, `
		SELECT amount, created_at, expires_at
		FROM user_balance_credits
		WHERE user_id = $1 AND source_type = 'first_recharge_bonus'
		ORDER BY id
		LIMIT 1
	`, []any{userID}, &credit.Amount, &credit.CreatedAt, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		v := expiresAt.Time
		credit.ExpiresAt = &v
	}
	return &credit, nil
}

const userTierAwardColumns = `id, user_id, tier_id, tier_code, tier_name_snapshot, threshold_snapshot, achieved_at, source, detail`

func scanUserTierAward(scan func(dest ...any) error) (*service.UserTierAward, error) {
	var (
		award     service.UserTierAward
		tierID    sql.NullInt64
		threshold sql.NullFloat64
		rawDetail []byte
	)
	if err := scan(&award.ID, &award.UserID, &tierID, &award.TierCode, &award.TierNameSnapshot, &threshold, &award.AchievedAt, &award.Source, &rawDetail); err != nil {
		return nil, err
	}
	if tierID.Valid {
		v := tierID.Int64
		award.TierID = &v
	}
	if threshold.Valid {
		v := threshold.Float64
		award.ThresholdSnapshot = &v
	}
	award.Detail = decodeJSONMap(rawDetail)
	return &award, nil
}

func (r *userTierRepository) ListUserAwards(ctx context.Context, userID int64) ([]service.UserTierAward, error) {
	if userID <= 0 {
		return nil, nil
	}
	exec, err := r.execer(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := exec.QueryContext(ctx, `
		SELECT `+userTierAwardColumns+`
		FROM user_tier_awards
		WHERE user_id = $1
		ORDER BY achieved_at, id
	`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var awards []service.UserTierAward
	for rows.Next() {
		award, err := scanUserTierAward(rows.Scan)
		if err != nil {
			return nil, err
		}
		awards = append(awards, *award)
	}
	return awards, rows.Err()
}

const userTierEffectColumns = `id, award_id, user_id, benefit_key, benefit_type, amount, group_id, rate_multiplier,
	status, attempts, last_error, detail, applied_at`

func scanUserTierEffect(scan func(dest ...any) error) (*service.UserTierEffect, error) {
	var (
		effect  service.UserTierEffect
		amount  sql.NullFloat64
		groupID sql.NullInt64
		rate    sql.NullFloat64
		applied sql.NullTime
		rawJSON []byte
	)
	if err := scan(&effect.ID, &effect.AwardID, &effect.UserID, &effect.BenefitKey, &effect.BenefitType,
		&amount, &groupID, &rate, &effect.Status, &effect.Attempts, &effect.LastError, &rawJSON, &applied); err != nil {
		return nil, err
	}
	if amount.Valid {
		v := amount.Float64
		effect.Amount = &v
	}
	if groupID.Valid {
		v := groupID.Int64
		effect.GroupID = &v
	}
	if rate.Valid {
		v := rate.Float64
		effect.RateMultiplier = &v
	}
	if applied.Valid {
		v := applied.Time
		effect.AppliedAt = &v
	}
	effect.Detail = decodeJSONMap(rawJSON)
	return &effect, nil
}

func (r *userTierRepository) ListUserEffects(ctx context.Context, userID int64) ([]service.UserTierEffect, error) {
	if userID <= 0 {
		return nil, nil
	}
	exec, err := r.execer(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := exec.QueryContext(ctx, `
		SELECT `+userTierEffectColumns+`
		FROM user_tier_effects
		WHERE user_id = $1
		ORDER BY created_at, id
	`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var effects []service.UserTierEffect
	for rows.Next() {
		effect, err := scanUserTierEffect(rows.Scan)
		if err != nil {
			return nil, err
		}
		effects = append(effects, *effect)
	}
	return effects, rows.Err()
}

func (r *userTierRepository) ListUserRateOverlays(ctx context.Context, userID int64) ([]service.UserTierRateOverlay, error) {
	if userID <= 0 {
		return nil, nil
	}
	exec, err := r.execer(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := exec.QueryContext(ctx, `
		SELECT id, user_id, group_id, effect_id, rate_multiplier, status, created_at
		FROM user_tier_rate_overlays
		WHERE user_id = $1 AND status = 'active'
		ORDER BY group_id, id
	`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var overlays []service.UserTierRateOverlay
	for rows.Next() {
		var o service.UserTierRateOverlay
		if err := rows.Scan(&o.ID, &o.UserID, &o.GroupID, &o.EffectID, &o.RateMultiplier, &o.Status, &o.CreatedAt); err != nil {
			return nil, err
		}
		overlays = append(overlays, o)
	}
	return overlays, rows.Err()
}

// EnsureAward 幂等创建授予记录：唯一约束 (user_id, tier_code) 是幂等锚点。
func (r *userTierRepository) EnsureAward(ctx context.Context, award *service.UserTierAward) (int64, bool, error) {
	exec, err := r.execer(ctx)
	if err != nil {
		return 0, false, err
	}
	detail, err := json.Marshal(orEmptyJSONMap(award.Detail))
	if err != nil {
		return 0, false, err
	}
	var id int64
	err = scanSingleRow(ctx, exec, `
		INSERT INTO user_tier_awards (user_id, tier_id, tier_code, tier_name_snapshot, threshold_snapshot, achieved_at, source, detail, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), $6, $7::jsonb, NOW(), NOW())
		ON CONFLICT (user_id, tier_code) DO NOTHING
		RETURNING id
	`, []any{award.UserID, award.TierID, award.TierCode, award.TierNameSnapshot, award.ThresholdSnapshot, award.Source, string(detail)}, &id)
	if err == nil {
		return id, true, nil
	}
	if err != sql.ErrNoRows {
		return 0, false, err
	}
	// 冲突：读取既有授予记录，返回既有状态（重复领取不是错误）
	if err := scanSingleRow(ctx, exec, `
		SELECT id FROM user_tier_awards WHERE user_id = $1 AND tier_code = $2
	`, []any{award.UserID, award.TierCode}, &id); err != nil {
		return 0, false, err
	}
	return id, false, nil
}

// EnsureEffect 幂等创建权益发放记录：唯一约束 (award_id, benefit_key) 是幂等锚点。
func (r *userTierRepository) EnsureEffect(ctx context.Context, effect *service.UserTierEffect) (*service.UserTierEffect, bool, error) {
	exec, err := r.execer(ctx)
	if err != nil {
		return nil, false, err
	}
	detail, err := json.Marshal(orEmptyJSONMap(effect.Detail))
	if err != nil {
		return nil, false, err
	}
	var id int64
	err = scanSingleRow(ctx, exec, `
		INSERT INTO user_tier_effects (award_id, user_id, benefit_key, benefit_type, amount, group_id, rate_multiplier, status, attempts, detail, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', 0, $8::jsonb, NOW(), NOW())
		ON CONFLICT (award_id, benefit_key) DO NOTHING
		RETURNING id
	`, []any{effect.AwardID, effect.UserID, effect.BenefitKey, effect.BenefitType, effect.Amount, effect.GroupID, effect.RateMultiplier, string(detail)}, &id)
	created := err == nil
	if err != nil && err != sql.ErrNoRows {
		return nil, false, err
	}

	stored, err := scanUserTierEffect(func(dest ...any) error {
		return scanSingleRow(ctx, exec, `
			SELECT `+userTierEffectColumns+`
			FROM user_tier_effects
			WHERE award_id = $1 AND benefit_key = $2
		`, []any{effect.AwardID, effect.BenefitKey}, dest...)
	})
	if err != nil {
		return nil, false, err
	}
	return stored, created, nil
}

func (r *userTierRepository) MarkEffectApplied(ctx context.Context, effectID int64, detail map[string]any) error {
	exec, err := r.execer(ctx)
	if err != nil {
		return err
	}
	rawDetail, err := json.Marshal(orEmptyJSONMap(detail))
	if err != nil {
		return err
	}
	_, err = exec.ExecContext(ctx, `
		UPDATE user_tier_effects
		SET status = 'applied', attempts = attempts + 1, last_error = '', detail = $2::jsonb, applied_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, effectID, string(rawDetail))
	return err
}

// 刻意没有 MarkEffectFailed：任一权益失败即整事务回滚，回滚后 effect 行不存在，
// 没有可标记为 failed 的对象。失败可见面是 service 层的结构化日志（见 claim）。

func (r *userTierRepository) HasManualRateMultiplier(ctx context.Context, userID, groupID int64) (bool, error) {
	exec, err := r.execer(ctx)
	if err != nil {
		return false, err
	}
	var exists bool
	if err := scanSingleRow(ctx, exec, `
		SELECT EXISTS(
			SELECT 1 FROM user_group_rate_multipliers
			WHERE user_id = $1 AND group_id = $2 AND rate_multiplier IS NOT NULL
		)
	`, []any{userID, groupID}, &exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (r *userTierRepository) UpsertRateOverlay(ctx context.Context, overlay service.UserTierRateOverlay) error {
	exec, err := r.execer(ctx)
	if err != nil {
		return err
	}
	_, err = exec.ExecContext(ctx, `
		INSERT INTO user_tier_rate_overlays (user_id, group_id, effect_id, rate_multiplier, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'active', NOW(), NOW())
		ON CONFLICT (user_id, group_id, effect_id)
		DO UPDATE SET rate_multiplier = EXCLUDED.rate_multiplier, status = 'active', updated_at = NOW()
	`, overlay.UserID, overlay.GroupID, overlay.EffectID, overlay.RateMultiplier)
	return err
}

// ApplyTierRewardBalance 只增加余额。刻意不走 users.UpdateBalance：后者会累加 users.total_recharged，
// 而等级奖励按设计不计入充值额（同 ApplyRedeemBalanceAdjustment 的写法）。
func (r *userTierRepository) ApplyTierRewardBalance(ctx context.Context, userID int64, amount float64) error {
	if amount <= 0 {
		return nil
	}
	exec, err := r.execer(ctx)
	if err != nil {
		return err
	}
	result, err := exec.ExecContext(ctx, `
		UPDATE users SET balance = balance + $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
	`, amount, userID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrUserNotFound
	}
	return nil
}

// ---------- 等级手工指派（user_tier_assignments） ----------

const userTierAssignmentColumns = `user_id, tier_id, tier_code, tier_name_snapshot, sort_order_snapshot,
	source, note, assigned_by, assigned_at, updated_at`

func scanUserTierAssignment(scan func(dest ...any) error) (*service.UserTierAssignment, error) {
	var (
		assignment service.UserTierAssignment
		tierID     sql.NullInt64
		assignedBy sql.NullInt64
	)
	if err := scan(&assignment.UserID, &tierID, &assignment.TierCode, &assignment.TierNameSnapshot,
		&assignment.SortOrderSnapshot, &assignment.Source, &assignment.Note, &assignedBy,
		&assignment.AssignedAt, &assignment.UpdatedAt); err != nil {
		return nil, err
	}
	if tierID.Valid {
		v := tierID.Int64
		assignment.TierID = &v
	}
	if assignedBy.Valid {
		v := assignedBy.Int64
		assignment.AssignedBy = &v
	}
	return &assignment, nil
}

// GetAssignment 读取用户当前指派；无指派返回 (nil, nil)——「没有指派」是正常状态而不是错误。
func (r *userTierRepository) GetAssignment(ctx context.Context, userID int64) (*service.UserTierAssignment, error) {
	if userID <= 0 {
		return nil, nil
	}
	exec, err := r.execer(ctx)
	if err != nil {
		return nil, err
	}
	assignment, err := scanUserTierAssignment(func(dest ...any) error {
		return scanSingleRow(ctx, exec, `
			SELECT `+userTierAssignmentColumns+`
			FROM user_tier_assignments
			WHERE user_id = $1
		`, []any{userID}, dest...)
	})
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return assignment, nil
}

// UpsertAssignment 幂等写入指派：user_id 是主键即幂等锚点，重指派覆盖全部字段
// （含 source/note 与新的阶梯位置），assigned_at 更新时间戳但 created_at 保留首次指派时刻。
func (r *userTierRepository) UpsertAssignment(ctx context.Context, assignment *service.UserTierAssignment) (*service.UserTierAssignment, error) {
	if assignment == nil || assignment.UserID <= 0 {
		return nil, service.ErrUserNotFound
	}
	exec, err := r.execer(ctx)
	if err != nil {
		return nil, err
	}
	source := assignment.Source
	if source == "" {
		source = service.UserTierAssignmentSourceAdmin
	}
	stored, err := scanUserTierAssignment(func(dest ...any) error {
		return scanSingleRow(ctx, exec, `
			INSERT INTO user_tier_assignments
				(user_id, tier_id, tier_code, tier_name_snapshot, sort_order_snapshot, source, note,
				 assigned_by, assigned_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW(), NOW())
			ON CONFLICT (user_id) DO UPDATE SET
				tier_id             = EXCLUDED.tier_id,
				tier_code           = EXCLUDED.tier_code,
				tier_name_snapshot  = EXCLUDED.tier_name_snapshot,
				sort_order_snapshot = EXCLUDED.sort_order_snapshot,
				source              = EXCLUDED.source,
				note                = EXCLUDED.note,
				assigned_by         = EXCLUDED.assigned_by,
				assigned_at         = NOW(),
				updated_at          = NOW()
			RETURNING `+userTierAssignmentColumns+`
		`, []any{assignment.UserID, assignment.TierID, assignment.TierCode, assignment.TierNameSnapshot,
			assignment.SortOrderSnapshot, source, assignment.Note, assignment.AssignedBy},
			dest...)
	})
	if err != nil {
		return nil, err
	}
	return stored, nil
}

// DeleteAssignment 取消指派（幂等：无既有行时返回 false, nil）。
// 刻意不回收任何已发权益：取消指派只影响「还没领的档位」是否可领。
func (r *userTierRepository) DeleteAssignment(ctx context.Context, userID int64) (bool, error) {
	if userID <= 0 {
		return false, nil
	}
	exec, err := r.execer(ctx)
	if err != nil {
		return false, err
	}
	result, err := exec.ExecContext(ctx, `DELETE FROM user_tier_assignments WHERE user_id = $1`, userID)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *userTierRepository) CountAssignmentsByTierID(ctx context.Context, tierID int64) (int64, error) {
	if tierID <= 0 {
		return 0, nil
	}
	exec, err := r.execer(ctx)
	if err != nil {
		return 0, err
	}
	var count int64
	if err := scanSingleRow(ctx, exec, `SELECT count(*) FROM user_tier_assignments WHERE tier_id = $1`, []any{tierID}, &count); err != nil {
		return 0, err
	}
	return count, nil
}

func balanceAmountOf(b service.UserTierBenefit) float64 {
	if b.BalanceCredit == nil {
		return 0
	}
	return b.BalanceCredit.Amount
}

func balanceValidityDaysOf(b service.UserTierBenefit) int {
	if b.BalanceCredit == nil {
		return 0
	}
	return b.BalanceCredit.ValidityDays
}

func groupIDOf(b service.UserTierBenefit) int64 {
	if b.GroupRate == nil {
		return 0
	}
	return b.GroupRate.GroupID
}

func groupRateOf(b service.UserTierBenefit) float64 {
	if b.GroupRate == nil {
		return 0
	}
	return b.GroupRate.RateMultiplier
}

func orEmptyJSONMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

func decodeJSONMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{"raw": string(raw)}
	}
	return out
}

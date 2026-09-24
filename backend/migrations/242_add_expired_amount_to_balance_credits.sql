-- 额度账本补记「到期作废金额」，修正用户等级消费口径的一处失真。
--
-- 背景：到期任务把 remaining_amount 置 0 并只留 expired_at，未使用的部分因此不可再区分。
-- 用户等级的累计消费口径 `sum(amount - remaining_amount)` 会把「到期作废」整额算成
-- 「已消费」——一张兑换后从未使用、到期作废的 1500 美元额度，足以让用户达标领 150 美元
-- 赠送与 0.9 分组倍率。本迁移记录到期那一刻仍未使用的余额，口径改为
-- `amount - remaining_amount - COALESCE(expired_amount, 0)`，得到真实消费额。
--
-- 只加列，不改既有列、不动既有行：非 expired 行该列恒为 NULL，口径与加列前逐位一致。
--
-- 回滚：口径 SQL 改回 `amount - remaining_amount` 即可；本列留着无害（不删列）。
--
-- 注意（不可回填）：本迁移之前已经 expired 的行，其「到期时未使用金额」当时未被记录，
-- remaining_amount 已被置 0，无法从任何现存字段还原。这些行保持 expired_amount = NULL，
-- 因而仍会被口径当作「已全额消费」。生产核对：迁移时该表 expired 行为 0，故无实际影响。

ALTER TABLE user_balance_credits
    ADD COLUMN IF NOT EXISTS expired_amount NUMERIC(20, 8);

COMMENT ON COLUMN user_balance_credits.expired_amount IS
    '到期作废时仍未使用的余额（仅 status=expired 且由本迁移之后的到期任务处理的行有值）；'
    '消费口径需减去它，避免把「到期作废」算成「已消费」';

-- 同步修正发放状态的文档口径：任一权益失败即整事务回滚，不存在可标记的 failed 行，
-- 故 'failed' 不再是一条会被写入的状态（保留取值以免约束变更，语义上仅历史遗留）。
COMMENT ON COLUMN user_tier_effects.status IS
    'pending 待发 / applied 已发 / failed 历史遗留（现行实现为失败整体回滚，不写 failed）';

-- 等级手工指派（按邮箱）：管理员把指定用户配置到指定等级，纯新增一张表，不改任何既有表结构。
--
-- 设计要点（见 docs/plan/sub2api-user-tier-manual-assignment-20260924.md）：
--   1. 指派只是「抬高达成下限」的一条配置，不写 user_tier_awards / user_tier_effects /
--      user_tier_rate_overlays；权益仍由用户按既有领取链路手动领取。
--      因此不存在「等级已获得但权益未发」的中间态，也不产生误发额度的路径。
--   2. user_id 作主键：一个用户当前只有一条指派，重指派即覆盖（ON CONFLICT DO UPDATE），
--      天然幂等；不需要历史版本表，审计信息就在本行。
--   3. tier_id 可为空（与 user_tier_awards.tier_id 同语义：配置删除后保留快照）。
--      服务层另有一道守卫：已有用户指派的等级只能停用，不能物理删除。
--   4. 判定「该等级及之前的等级」用档位顺序 (sort_order, id)，与 ListTiers 的排序同源；
--      因此除 tier_id 外再留 sort_order_snapshot，作为 tier_id 不可用时的兜底。
--   5. source 区分来源：admin 管理端人工配置 / historical_20260924 历史累计一次性批量指派，
--      使一次性批量可被精确回滚（DELETE ... WHERE source = 'historical_20260924'）。
--
-- 回滚：保留该表（回退代码后成为无引用表，指派不生效也不丢失），不提供破坏性 DROP。

CREATE TABLE IF NOT EXISTS user_tier_assignments (
    user_id             BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    tier_id             BIGINT REFERENCES user_tiers(id) ON DELETE SET NULL,
    tier_code           VARCHAR(64)  NOT NULL,
    tier_name_snapshot  VARCHAR(128) NOT NULL DEFAULT '',
    sort_order_snapshot INTEGER      NOT NULL DEFAULT 0,
    source              VARCHAR(32)  NOT NULL DEFAULT 'admin',
    note                TEXT         NOT NULL DEFAULT '',
    assigned_by         BIGINT,
    assigned_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT user_tier_assignments_source_check
        CHECK (source IN ('admin', 'historical_20260924'))
);

-- 删除守卫与运营核对用：查「哪些用户被指派到该档位」。
CREATE INDEX IF NOT EXISTS idx_user_tier_assignments_tier ON user_tier_assignments (tier_id);

COMMENT ON TABLE user_tier_assignments IS
    '用户等级手工指派（一人一条，重指派即覆盖；只抬高达成下限，不发放权益）';
COMMENT ON COLUMN user_tier_assignments.tier_id IS
    '目标档位；可空（配置被删除后仍保留快照）。已有指派的档位只能停用，不能删除';
COMMENT ON COLUMN user_tier_assignments.tier_code IS
    '目标档位标识快照（等级标识可被修改，故以 tier_id 为准，此处仅作快照与展示）';
COMMENT ON COLUMN user_tier_assignments.sort_order_snapshot IS
    '指派时刻的档位顺序；仅在 tier_id 不可用时作为阶梯位置兜底';
COMMENT ON COLUMN user_tier_assignments.source IS
    '来源：admin 管理端人工配置 / historical_20260924 历史累计一次性批量指派';
COMMENT ON COLUMN user_tier_assignments.assigned_by IS
    '指派的管理员用户 ID；脚本批量写入时为 NULL';

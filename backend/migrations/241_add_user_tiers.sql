-- 用户等级与权益体系（低侵入方案）：五张新表，不改任何既有表结构。
--
-- 设计要点（见 docs/plan/sub2api-user-tier-system-20260924.md）：
--   1. 档位与权益全部是管理端数据，代码里只有「权益类型」与「触发类型」两层枚举，
--      不出现任何具体档位常量、称呼或阈值。
--   2. 已获得的等级写快照（user_tier_awards），配置阈值/名称/排序变化不影响历史等级，
--      因此「只升不降」由数据模型保证，不需要额外列。
--   3. 一次性发放的幂等锚点是 user_tier_effects 的唯一约束；
--      余额账本行（user_balance_credits）与倍率覆盖行只是实际效果，不单独决定是否已发放。
--   4. 等级倍率只写覆盖层，读取时手工倍率（user_group_rate_multipliers）优先，
--      等级权益不写入、不覆盖既有倍率表。
--
-- 回滚：保留全部新表（关闭入口即可停止新发放），不提供破坏性 DROP。

CREATE TABLE IF NOT EXISTS user_tiers (
    id            BIGSERIAL PRIMARY KEY,
    code          VARCHAR(64)  NOT NULL,
    name          VARCHAR(128) NOT NULL,
    description   TEXT         NOT NULL DEFAULT '',
    sort_order    INTEGER      NOT NULL DEFAULT 0,
    trigger_type  VARCHAR(32)  NOT NULL,
    threshold_usd NUMERIC(20, 8),
    enabled       BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT user_tiers_trigger_type_check
        CHECK (trigger_type IN ('first_recharge', 'consumption')),
    CONSTRAINT user_tiers_threshold_check
        CHECK (
            (trigger_type = 'consumption' AND threshold_usd IS NOT NULL AND threshold_usd >= 0)
            OR (trigger_type = 'first_recharge' AND threshold_usd IS NULL)
        )
);

-- code 是稳定标识，用于幂等与展示，且一旦产生授予记录不可修改。
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_tiers_code ON user_tiers (code);

-- 排序不设唯一约束（避免换序时的两阶段更新），但读取一律 ORDER BY sort_order, id，
-- 保证「下一等级」的判定稳定。
CREATE INDEX IF NOT EXISTS idx_user_tiers_sort_order ON user_tiers (sort_order, id);

COMMENT ON TABLE user_tiers IS '用户等级定义（可无限增删；有授予记录的等级只能停用）';
COMMENT ON COLUMN user_tiers.code IS '稳定标识（如 first_recharge / consume_500），用于幂等与展示';
COMMENT ON COLUMN user_tiers.name IS '等级称呼（面向用户的展示名）';
COMMENT ON COLUMN user_tiers.trigger_type IS '触发类型：first_recharge 首充 / consumption 累计消费额';
COMMENT ON COLUMN user_tiers.threshold_usd IS '消费额门槛（仅 consumption 档必填，first_recharge 档为 NULL）';

-- 一档多项权益。params 用 JSONB 而不是每类一张表：类型集合后续要扩展（RPM/并发等），
-- jsonb + 类型注册表让新增类型不必改表结构；代价是参数校验必须在服务层按类型显式做。
CREATE TABLE IF NOT EXISTS user_tier_benefits (
    id           BIGSERIAL PRIMARY KEY,
    tier_id      BIGINT      NOT NULL REFERENCES user_tiers(id) ON DELETE CASCADE,
    benefit_type VARCHAR(32) NOT NULL,
    params       JSONB       NOT NULL DEFAULT '{}'::jsonb,
    enabled      BOOLEAN     NOT NULL DEFAULT TRUE,
    sort_order   INTEGER     NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_tier_benefits_type_check
        CHECK (benefit_type IN ('balance_credit', 'group_rate'))
);

CREATE INDEX IF NOT EXISTS idx_user_tier_benefits_tier ON user_tier_benefits (tier_id, sort_order, id);

COMMENT ON TABLE user_tier_benefits IS '每档权益定义（一档可多项）';
COMMENT ON COLUMN user_tier_benefits.params IS 'balance_credit: {amount, validity_days}；group_rate: {group_id, rate_multiplier}';

-- 已获得等级快照。当前等级取本表中该用户已获得等级的最高档。
CREATE TABLE IF NOT EXISTS user_tier_awards (
    id                 BIGSERIAL PRIMARY KEY,
    user_id            BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tier_id            BIGINT       REFERENCES user_tiers(id) ON DELETE SET NULL,
    tier_code          VARCHAR(64)  NOT NULL,
    tier_name_snapshot VARCHAR(128) NOT NULL DEFAULT '',
    threshold_snapshot NUMERIC(20, 8),
    achieved_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    source             VARCHAR(32)  NOT NULL DEFAULT 'claim',
    detail             JSONB        NOT NULL DEFAULT '{}'::jsonb,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT user_tier_awards_source_check
        CHECK (source IN ('claim', 'first_recharge', 'manual_20260924', 'backfill'))
);

-- 幂等锚点之一：同一用户同一档位只允许一条授予记录。
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_tier_awards_user_tier
    ON user_tier_awards (user_id, tier_code);

CREATE INDEX IF NOT EXISTS idx_user_tier_awards_user ON user_tier_awards (user_id, achieved_at DESC);

COMMENT ON TABLE user_tier_awards IS '已获得等级快照（配置变化不影响历史等级，只升不降）';
COMMENT ON COLUMN user_tier_awards.tier_id IS '可空：配置被停用/删除后仍保留历史快照';
COMMENT ON COLUMN user_tier_awards.source IS '授予来源：claim 手动领取 / first_recharge 首充自动 / manual_20260924 历史手工授予 / backfill 回填';

-- 权益发放状态、幂等与重试。唯一约束 (award_id, benefit_key) 是一次性发放的幂等锚点。
CREATE TABLE IF NOT EXISTS user_tier_effects (
    id              BIGSERIAL PRIMARY KEY,
    award_id        BIGINT      NOT NULL REFERENCES user_tier_awards(id) ON DELETE CASCADE,
    user_id         BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    benefit_key     VARCHAR(64) NOT NULL,
    benefit_type    VARCHAR(32) NOT NULL,
    amount          NUMERIC(20, 8),
    group_id        BIGINT,
    rate_multiplier NUMERIC(20, 8),
    status          VARCHAR(16) NOT NULL DEFAULT 'pending',
    attempts        INTEGER     NOT NULL DEFAULT 0,
    last_error      TEXT        NOT NULL DEFAULT '',
    detail          JSONB       NOT NULL DEFAULT '{}'::jsonb,
    applied_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_tier_effects_status_check
        CHECK (status IN ('pending', 'applied', 'failed')),
    CONSTRAINT user_tier_effects_type_check
        CHECK (benefit_type IN ('balance_credit', 'group_rate'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_tier_effects_award_benefit
    ON user_tier_effects (award_id, benefit_key);

CREATE INDEX IF NOT EXISTS idx_user_tier_effects_user ON user_tier_effects (user_id, created_at DESC);

COMMENT ON TABLE user_tier_effects IS '权益发放状态/幂等/重试；实发金额与倍率在此留快照';
COMMENT ON COLUMN user_tier_effects.benefit_key IS '稳定权益键（benefit:<benefit_id>），同档位内唯一';
COMMENT ON COLUMN user_tier_effects.status IS 'pending 待发 / applied 已发 / failed 失败可重试';

-- 等级倍率覆盖层：只被等级权益写入，读取时手工 user_group_rate_multipliers 优先。
CREATE TABLE IF NOT EXISTS user_tier_rate_overlays (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id        BIGINT         NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    effect_id       BIGINT         NOT NULL REFERENCES user_tier_effects(id) ON DELETE CASCADE,
    rate_multiplier NUMERIC(20, 8) NOT NULL,
    status          VARCHAR(16)    NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    CONSTRAINT user_tier_rate_overlays_status_check
        CHECK (status IN ('active', 'revoked')),
    CONSTRAINT user_tier_rate_overlays_rate_check
        CHECK (rate_multiplier > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_tier_rate_overlays_unique
    ON user_tier_rate_overlays (user_id, group_id, effect_id);

-- 计费读取路径用：只查有效覆盖，按倍率升序取最低（同一分组多条时让利用户）。
CREATE INDEX IF NOT EXISTS idx_user_tier_rate_overlays_lookup
    ON user_tier_rate_overlays (user_id, group_id, rate_multiplier)
    WHERE status = 'active';

COMMENT ON TABLE user_tier_rate_overlays IS '等级倍率覆盖层（不写入、不覆盖 user_group_rate_multipliers）';

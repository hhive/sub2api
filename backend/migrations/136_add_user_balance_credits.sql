-- 用户余额批次账本：支持每笔余额独立有效期和日终结算

INSERT INTO settings (key, value, updated_at)
VALUES
    ('balance_credit_validity_days', '0', NOW())
ON CONFLICT (key) DO NOTHING;

CREATE TABLE IF NOT EXISTS user_balance_credits (
    id               BIGSERIAL PRIMARY KEY,
    user_id          BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source_type      VARCHAR(32) NOT NULL,
    source_id        VARCHAR(128) NOT NULL DEFAULT '',
    source_code      VARCHAR(128) NOT NULL DEFAULT '',
    amount           DECIMAL(20,8) NOT NULL,
    remaining_amount DECIMAL(20,8) NOT NULL,
    settled_until_date DATE,
    expires_at       TIMESTAMPTZ,
    expired_at       TIMESTAMPTZ,
    status           VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_balance_credits_amount_positive CHECK (amount > 0),
    CONSTRAINT user_balance_credits_remaining_non_negative CHECK (remaining_amount >= 0),
    CONSTRAINT user_balance_credits_remaining_not_exceed_amount CHECK (remaining_amount <= amount)
);

ALTER TABLE user_balance_credits ADD COLUMN IF NOT EXISTS source_type VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE user_balance_credits ADD COLUMN IF NOT EXISTS source_id VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE user_balance_credits ADD COLUMN IF NOT EXISTS source_code VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE user_balance_credits ADD COLUMN IF NOT EXISTS remaining_amount DECIMAL(20,8) NOT NULL DEFAULT 0;
ALTER TABLE user_balance_credits ADD COLUMN IF NOT EXISTS settled_until_date DATE;
ALTER TABLE user_balance_credits ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;
ALTER TABLE user_balance_credits ADD COLUMN IF NOT EXISTS expired_at TIMESTAMPTZ;
ALTER TABLE user_balance_credits ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active';
ALTER TABLE user_balance_credits ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_user_balance_credits_daily_settlement
    ON user_balance_credits(user_id, status, settled_until_date, expires_at, id)
    WHERE status = 'active' AND remaining_amount > 0;
CREATE INDEX IF NOT EXISTS idx_user_balance_credits_user_status_expires
    ON user_balance_credits(user_id, status, expires_at, id);
CREATE INDEX IF NOT EXISTS idx_user_balance_credits_expiry_scan
    ON user_balance_credits(status, expires_at, id)
    WHERE status = 'active' AND remaining_amount > 0 AND expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_user_balance_credits_source
    ON user_balance_credits(source_type, source_id);

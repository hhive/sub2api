INSERT INTO settings (key, value, updated_at)
VALUES
    ('first_recharge_bonus_enabled', 'false', NOW()),
    ('first_recharge_bonus_amount', '5', NOW()),
    ('first_recharge_bonus_validity_days', '3', NOW())
ON CONFLICT (key) DO NOTHING;

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_balance_credits_first_recharge_bonus_once
    ON user_balance_credits(user_id, source_type)
    WHERE source_type = 'first_recharge_bonus';

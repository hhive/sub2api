-- 余额流水冗余用户邮箱，支持管理端全局查询避免每次关联 users。

ALTER TABLE user_balance_credits
    ADD COLUMN IF NOT EXISTS email VARCHAR(255) NOT NULL DEFAULT '';

UPDATE user_balance_credits ubc
SET email = u.email
FROM users u
WHERE ubc.user_id = u.id
  AND ubc.email = '';

CREATE INDEX IF NOT EXISTS idx_user_balance_credits_email
    ON user_balance_credits(email);


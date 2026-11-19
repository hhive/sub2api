ALTER TABLE redeem_codes
    ADD COLUMN IF NOT EXISTS balance_credit_validity_days INTEGER;

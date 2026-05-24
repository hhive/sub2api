ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS rate_correction_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0;

COMMENT ON COLUMN groups.rate_correction_multiplier IS '分组隐藏修正倍率，仅管理员可见；余额/订阅实际扣费倍率 = 用户可见费用倍率 * 修正倍率，生图独立倍率不叠加该修正。';

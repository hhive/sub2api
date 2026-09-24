package service

import (
	"context"
	"fmt"
)

// IsUserTierEnabled 用户等级体系总开关（settings.user_tier_enabled）。
//
// opt-out 语义：**未配置或空值 = 开启**，只有显式 false/0/off/disabled 才算关闭。
// 这样既有部署升级后入口不会突然消失，开关也可以随时删掉（回到「开启」）。
//
// 读失败时返回 error，由调用方决定策略（本服务内统一「记日志 + 视为开启」，避免一次
// 读抖动把用户端入口整页打掉）。
func (s *SettingService) IsUserTierEnabled(ctx context.Context) (bool, error) {
	if s == nil || s.settingRepo == nil {
		return true, nil
	}
	value, err := s.settingRepo.GetValue(ctx, SettingKeyUserTierEnabled)
	if err != nil {
		return true, err
	}
	return !isFalseSettingValue(value), nil
}

// SetUserTierEnabled 写入总开关并立即失效依赖设置的缓存（与设置页保存后的收尾一致）。
func (s *SettingService) SetUserTierEnabled(ctx context.Context, enabled bool) error {
	if s == nil || s.settingRepo == nil {
		return fmt.Errorf("setting service is not configured")
	}
	value := "false"
	if enabled {
		value = "true"
	}
	if err := s.settingRepo.Set(ctx, SettingKeyUserTierEnabled, value); err != nil {
		return err
	}
	if s.onUpdate != nil {
		s.onUpdate()
	}
	return nil
}

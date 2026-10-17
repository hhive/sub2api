package service

import (
	"context"
	"errors"
	"time"
)

// GPT-6.1 Sol 长上下文计费开关（SettingKeyOpenAIGPT61SolLongContextBillingEnabled）的进程内缓存。
//
// 该开关在 OpenAI 用量计费热路径上每个请求读取一次，因此禁止在热路径上直接访问 DB：
//   - 缓存新鲜：直接返回。
//   - 缓存过期：先返回旧值，后台经 singleflight 刷新（stale-while-revalidate）。
//   - 冷缓存（进程刚起/首次调用）：同步读一次（独立超时），不让猜测值参与计费。
//
// 默认 false = 不应用官方长上下文阶梯；键缺失、DB 不可用或读取超时都按 false 处理，
// 与默认值同向，避免「读不到就按收费」的隐性反转。
type cachedOpenAIGPT61SolLongContextBillingEnabled struct {
	enabled   bool
	expiresAt int64 // unix nano
	// generation 为该条目对应的设置写入代次：刷新只在代次未变时落缓存，
	// 避免「在途刷新读到旧值、却在管理端保存之后才写缓存」把新值覆盖回旧值。
	generation uint64
}

const (
	openAIGPT61SolLongContextBillingEnabledRefreshKey = "openai_gpt61sol_long_context_billing_enabled"
	openAIGPT61SolLongContextBillingEnabledCacheTTL   = 60 * time.Second
	openAIGPT61SolLongContextBillingEnabledErrorTTL   = 5 * time.Second
	openAIGPT61SolLongContextBillingEnabledDBTimeout  = 5 * time.Second
)

// GetOpenAIGPT61SolLongContextBillingEnabled 返回非订阅 OpenAI 账号的 GPT-6.1 Sol 是否应用
// 官方价卡长上下文阶梯。订阅账号本就豁免、其它模型与渠道自定义区间价不受本开关影响，
// 判断位置见 openAIGPT61SolAPILongContextExempt（OpenAI 用量计费的 modelLongContextExempt）。
func (s *SettingService) GetOpenAIGPT61SolLongContextBillingEnabled(ctx context.Context) bool {
	if s == nil {
		return false
	}
	cached, _ := s.gpt61SolLongContextBillingEnabledCache.Load().(*cachedOpenAIGPT61SolLongContextBillingEnabled)
	now := time.Now().UnixNano()
	if cached != nil && now < cached.expiresAt {
		return cached.enabled
	}
	if cached == nil {
		// 冷缓存：同步读一次，避免把猜测值算进计费。
		s.refreshOpenAIGPT61SolLongContextBillingEnabled(ctx)
		if refreshed, _ := s.gpt61SolLongContextBillingEnabledCache.Load().(*cachedOpenAIGPT61SolLongContextBillingEnabled); refreshed != nil {
			return refreshed.enabled
		}
		return false
	}
	// 过期：先返回旧值，后台刷新（DoChan 去重并发刷新，结果经 atomic 缓存可见）。
	s.gpt61SolLongContextBillingEnabledSF.DoChan(openAIGPT61SolLongContextBillingEnabledRefreshKey, func() (any, error) {
		s.refreshOpenAIGPT61SolLongContextBillingEnabled(context.Background())
		return nil, nil
	})
	return cached.enabled
}

// refreshOpenAIGPT61SolLongContextBillingEnabled 从 DB 读取最新值写入缓存。
// 真错误时保持旧值并改用短 TTL 以便尽快重试；键缺失按默认 false。
// 读前记录写入代次、读后仅在代次未变时落缓存：与设置写入路径竞争时宁可丢弃本次结果
// （下一次读取会重新刷新），也不允许用陈旧 DB 值覆盖刚保存的新值。
func (s *SettingService) refreshOpenAIGPT61SolLongContextBillingEnabled(ctx context.Context) {
	if s == nil || s.settingRepo == nil {
		return
	}
	dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), openAIGPT61SolLongContextBillingEnabledDBTimeout)
	defer cancel()

	generation := s.gpt61SolLongContextBillingEnabledGen.Load()

	enabled := false
	ttl := openAIGPT61SolLongContextBillingEnabledCacheTTL
	raw, err := s.settingRepo.GetValue(dbCtx, SettingKeyOpenAIGPT61SolLongContextBillingEnabled)
	switch {
	case err == nil:
		enabled = raw == "true"
	case errors.Is(err, ErrSettingNotFound):
		enabled = false // 键缺失 = 默认不应用官方长上下文阶梯
	default:
		if prior, _ := s.gpt61SolLongContextBillingEnabledCache.Load().(*cachedOpenAIGPT61SolLongContextBillingEnabled); prior != nil {
			enabled = prior.enabled
		}
		ttl = openAIGPT61SolLongContextBillingEnabledErrorTTL
	}

	// 期间有更新的设置写入（代次已前进）：丢弃本次结果，不用陈旧值覆盖新值。
	if s.gpt61SolLongContextBillingEnabledGen.Load() != generation {
		return
	}

	s.gpt61SolLongContextBillingEnabledCache.Store(&cachedOpenAIGPT61SolLongContextBillingEnabled{
		enabled:    enabled,
		expiresAt:  time.Now().Add(ttl).UnixNano(),
		generation: generation,
	})
}

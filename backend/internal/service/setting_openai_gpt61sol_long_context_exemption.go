package service

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

// GPT-6.1 Sol 长上下文豁免设置（SettingKeyOpenAIGPT61SolLongContextBillingEnabled +
// SettingKeyOpenAIGPT61SolLongContextBillingAccountIDs）的进程内缓存快照。
//
// 该判据在 OpenAI 用量计费热路径上每个请求读取一次，因此禁止在热路径上直接访问 DB：
//   - 缓存新鲜：直接返回。
//   - 缓存过期：先返回旧值，后台经 singleflight 刷新（stale-while-revalidate）。
//   - 冷缓存（进程刚起/首次调用）：同步读一次（独立超时），不让猜测值参与计费。
//
// 语义（与旧开关相反，见计划 §1）：enabled = true 表示豁免；enabled = false 表示照常按
// 官方价卡长上下文阶梯收费。accountIDs 为空表示全部非订阅 OpenAI 账号都豁免。
//
// 失败方向（计划 §3）：键缺失 → 默认（false / 空名单）；名单 JSON 非法或含非法 ID →
// 整体按 false（不豁免）并改用短 TTL 重试；DB 真错误 → 保留上一份快照 + 短 TTL；
// 冷缓存 + 错误 → false。任何情况下都不允许「配置损坏 = 全员豁免」。
type openAIGPT61SolLongContextExemption struct {
	enabled    bool
	accountIDs map[int64]struct{}
}

type cachedOpenAIGPT61SolLongContextExemption struct {
	exemption openAIGPT61SolLongContextExemption
	expiresAt int64 // unix nano
	// generation 为该条目对应的设置写入代次：刷新只在代次未变时落缓存，
	// 避免「在途刷新读到旧值、却在管理端保存之后才写缓存」把新值覆盖回旧值。
	generation uint64
}

const (
	openAIGPT61SolLongContextExemptionRefreshKey = "openai_gpt61sol_long_context_exemption"
	openAIGPT61SolLongContextExemptionCacheTTL   = 60 * time.Second
	openAIGPT61SolLongContextExemptionErrorTTL   = 5 * time.Second
	openAIGPT61SolLongContextExemptionDBTimeout  = 5 * time.Second

	// OpenAIGPT61SolLongContextBillingAccountIDsMax 名单条数上限；超过由管理端回 400。
	OpenAIGPT61SolLongContextBillingAccountIDsMax = 2000
)

// GetOpenAIGPT61SolLongContextExemption 返回非订阅 OpenAI 账号的 GPT-6.1 Sol 长上下文豁免快照。
// 订阅账号本就豁免、其它模型与渠道自定义区间价不受本设置影响，判断位置见
// openAIGPT61SolAPILongContextExempt（OpenAI 用量计费的 modelLongContextExempt）。
func (s *SettingService) GetOpenAIGPT61SolLongContextExemption(ctx context.Context) openAIGPT61SolLongContextExemption {
	if s == nil {
		return openAIGPT61SolLongContextExemption{}
	}
	cached, _ := s.gpt61SolLongContextExemptionCache.Load().(*cachedOpenAIGPT61SolLongContextExemption)
	now := time.Now().UnixNano()
	if cached != nil && now < cached.expiresAt {
		return cached.exemption
	}
	if cached == nil {
		// 冷缓存：同步读一次，避免把猜测值算进计费。
		s.refreshOpenAIGPT61SolLongContextExemption(ctx)
		if refreshed, _ := s.gpt61SolLongContextExemptionCache.Load().(*cachedOpenAIGPT61SolLongContextExemption); refreshed != nil {
			return refreshed.exemption
		}
		return openAIGPT61SolLongContextExemption{}
	}
	// 过期：先返回旧值，后台刷新（DoChan 去重并发刷新，结果经 atomic 缓存可见）。
	s.gpt61SolLongContextExemptionSF.DoChan(openAIGPT61SolLongContextExemptionRefreshKey, func() (any, error) {
		s.refreshOpenAIGPT61SolLongContextExemption(context.Background())
		return nil, nil
	})
	return cached.exemption
}

// refreshOpenAIGPT61SolLongContextExemption 从 DB 读取两个键（开关 + 账号名单）写入缓存。
// DB 真错误时保持旧快照并改用短 TTL 以便尽快重试；键缺失按默认值。
// 读前记录写入代次、读后仅在代次未变时落缓存：与设置写入路径竞争时宁可丢弃本次结果
// （下一次读取会重新刷新），也不允许用陈旧 DB 值覆盖刚保存的新值。
func (s *SettingService) refreshOpenAIGPT61SolLongContextExemption(ctx context.Context) {
	if s == nil || s.settingRepo == nil {
		return
	}
	dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), openAIGPT61SolLongContextExemptionDBTimeout)
	defer cancel()

	generation := s.gpt61SolLongContextExemptionGen.Load()

	exemption := openAIGPT61SolLongContextExemption{}
	ttl := openAIGPT61SolLongContextExemptionCacheTTL

	rawEnabled, enabledErr := s.settingRepo.GetValue(dbCtx, SettingKeyOpenAIGPT61SolLongContextBillingEnabled)
	rawIDs, idsErr := s.settingRepo.GetValue(dbCtx, SettingKeyOpenAIGPT61SolLongContextBillingAccountIDs)

	switch {
	case isSettingReadFailure(enabledErr) || isSettingReadFailure(idsErr):
		// DB 真错误：保留上一份快照 + 短 TTL，不得静默翻成默认值。
		if prior, _ := s.gpt61SolLongContextExemptionCache.Load().(*cachedOpenAIGPT61SolLongContextExemption); prior != nil {
			exemption = prior.exemption
		}
		ttl = openAIGPT61SolLongContextExemptionErrorTTL
	default:
		ids, err := parseOpenAIGPT61SolLongContextBillingAccountIDs(rawIDs)
		if err != nil {
			// 配置损坏：整体按不豁免（false），不允许损坏退化成「全员豁免」。
			logger.L().With(
				zap.String("component", "service.setting"),
				zap.String("setting_key", SettingKeyOpenAIGPT61SolLongContextBillingAccountIDs),
			).Warn("openai_gpt61sol_long_context.account_ids_invalid", zap.Error(err))
			exemption = openAIGPT61SolLongContextExemption{}
			ttl = openAIGPT61SolLongContextExemptionErrorTTL
		} else {
			exemption = openAIGPT61SolLongContextExemption{
				enabled:    rawEnabled == "true", // 键缺失/非 "true" 都按 false，与默认值同向
				accountIDs: ids,
			}
		}
	}

	// 期间有更新的设置写入（代次已前进）：丢弃本次结果，不用陈旧值覆盖新值。
	if s.gpt61SolLongContextExemptionGen.Load() != generation {
		return
	}

	s.gpt61SolLongContextExemptionCache.Store(&cachedOpenAIGPT61SolLongContextExemption{
		exemption:  exemption,
		expiresAt:  time.Now().Add(ttl).UnixNano(),
		generation: generation,
	})
}

// isSettingReadFailure 区分「键不存在」与真正的读取错误：前者按默认值处理，
// 后者才需要保留旧快照并缩短重试间隔。
func isSettingReadFailure(err error) bool {
	return err != nil && !errors.Is(err, ErrSettingNotFound)
}

// parseOpenAIGPT61SolLongContextBillingAccountIDs 严格解析账号名单 JSON 为集合。
// 空串/null 视为空名单；非法 JSON、非整数或非正整数 ID 一律返回错误（fail closed）。
func parseOpenAIGPT61SolLongContextBillingAccountIDs(raw string) (map[int64]struct{}, error) {
	ids, err := ParseOpenAIGPT61SolLongContextBillingAccountIDs(raw)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return openAIGPT61SolLongContextAccountIDSet(ids), nil
}

// openAIGPT61SolLongContextAccountIDSet 把（已归一化的）ID 切片转成判据用的集合；
// 空输入返回 nil，调用方据此区分「全员豁免」与「仅名单内豁免」。
func openAIGPT61SolLongContextAccountIDSet(ids []int64) map[int64]struct{} {
	if len(ids) == 0 {
		return nil
	}
	set := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set
}

// ParseOpenAIGPT61SolLongContextBillingAccountIDs 严格解析账号名单 JSON 为升序切片，
// 供设置视图回填；空串/null 返回空切片，非法 JSON 或非法 ID 返回错误。
func ParseOpenAIGPT61SolLongContextBillingAccountIDs(raw string) ([]int64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "null" {
		return []int64{}, nil
	}
	var ids []int64
	if err := json.Unmarshal([]byte(trimmed), &ids); err != nil {
		return nil, err
	}
	if err := ValidateOpenAIGPT61SolLongContextBillingAccountIDs(ids); err != nil {
		return nil, err
	}
	return NormalizeOpenAIGPT61SolLongContextBillingAccountIDs(ids), nil
}

// ValidateOpenAIGPT61SolLongContextBillingAccountIDs 校验账号名单请求值：
// ID 必须为正整数、条数不超过上限。失败由管理端回 400，不静默丢弃。
func ValidateOpenAIGPT61SolLongContextBillingAccountIDs(ids []int64) error {
	if len(ids) > OpenAIGPT61SolLongContextBillingAccountIDsMax {
		return errors.New("too many account ids")
	}
	for _, id := range ids {
		if id <= 0 {
			return errors.New("account id must be a positive integer")
		}
	}
	return nil
}

// NormalizeOpenAIGPT61SolLongContextBillingAccountIDs 归一化账号名单：丢弃非正整数、去重、升序。
// nil/空输入返回空切片（显式写库为 "[]"，即全部账号豁免）。
func NormalizeOpenAIGPT61SolLongContextBillingAccountIDs(ids []int64) []int64 {
	out := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}

// MarshalOpenAIGPT61SolLongContextBillingAccountIDs 归一化后序列化写库；空名单写 "[]"。
func MarshalOpenAIGPT61SolLongContextBillingAccountIDs(ids []int64) string {
	raw, err := json.Marshal(NormalizeOpenAIGPT61SolLongContextBillingAccountIDs(ids))
	if err != nil {
		// []int64 不可能序列化失败；防御性返回空名单（配合关闭的开关即不豁免）。
		return "[]"
	}
	return string(raw)
}

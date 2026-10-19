//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

// gpt61SolExemptionSettingRepoStub 支持写入的设置仓储桩：settingRepoStub.SetMultiple 会 panic。
type gpt61SolExemptionSettingRepoStub struct {
	settingRepoStub
}

func (s *gpt61SolExemptionSettingRepoStub) SetMultiple(ctx context.Context, updates map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.values == nil {
		s.values = make(map[string]string, len(updates))
	}
	for key, value := range updates {
		s.values[key] = value
	}
	return nil
}

func TestSettingService_GetOpenAIGPT61SolLongContextExemption(t *testing.T) {
	enabledKey := SettingKeyOpenAIGPT61SolLongContextBillingEnabled
	idsKey := SettingKeyOpenAIGPT61SolLongContextBillingAccountIDs

	for _, tt := range []struct {
		name        string
		values      map[string]string
		err         error
		wantEnabled bool
		wantIDs     []int64
	}{
		{name: "缺键默认不豁免且名单为空", values: map[string]string{}},
		{name: "显式 true 且名单非空", values: map[string]string{enabledKey: "true", idsKey: "[7,3,7]"}, wantEnabled: true, wantIDs: []int64{3, 7}},
		{name: "显式 true 名单为空 = 全员豁免", values: map[string]string{enabledKey: "true", idsKey: "[]"}, wantEnabled: true},
		// 开关关闭时快照仍带解析后的名单（判据在 enabled=false 时短路），断言集合照旧核对。
		{name: "显式 false 时开关关闭", values: map[string]string{enabledKey: "false", idsKey: "[3]"}, wantIDs: []int64{3}},
		{name: "非布尔值按默认不豁免", values: map[string]string{enabledKey: "yes", idsKey: "[3]"}, wantIDs: []int64{3}},
		{name: "名单 JSON 非法整体按不豁免", values: map[string]string{enabledKey: "true", idsKey: "[3,"}},
		{name: "名单含非正整数整体按不豁免", values: map[string]string{enabledKey: "true", idsKey: "[3,-1]"}},
		{name: "名单含非整数整体按不豁免", values: map[string]string{enabledKey: "true", idsKey: `["3"]`}},
		{name: "名单为 null 视为空名单", values: map[string]string{enabledKey: "true", idsKey: "null"}, wantEnabled: true},
		{name: "DB 错误冷缓存按不豁免", values: map[string]string{enabledKey: "true", idsKey: "[3]"}, err: errors.New("db down")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewSettingService(&settingRepoStub{values: tt.values, err: tt.err}, nil)
			got := svc.GetOpenAIGPT61SolLongContextExemption(context.Background())
			require.Equal(t, tt.wantEnabled, got.enabled)
			require.Equal(t, len(tt.wantIDs), len(got.accountIDs))
			for _, id := range tt.wantIDs {
				_, ok := got.accountIDs[id]
				require.True(t, ok, "名单必须包含 %d", id)
			}
		})
	}

	// nil 接收者：计费路径可能缺设置服务，必须按缺省处理而不是 panic。
	var nilSvc *SettingService
	require.False(t, nilSvc.GetOpenAIGPT61SolLongContextExemption(context.Background()).enabled)
}

func TestSettingService_GPT61SolLongContextExemptionCache(t *testing.T) {
	ctx := context.Background()
	repo := &settingRepoStub{values: map[string]string{
		SettingKeyOpenAIGPT61SolLongContextBillingEnabled:    "true",
		SettingKeyOpenAIGPT61SolLongContextBillingAccountIDs: "[3]",
	}}
	svc := NewSettingService(repo, nil)

	require.True(t, svc.GetOpenAIGPT61SolLongContextExemption(ctx).enabled)
	require.True(t, svc.GetOpenAIGPT61SolLongContextExemption(ctx).enabled)
	require.Equal(t, 2, repo.getValueCalls, "冷读两键一次；TTL 内重复读取不得回查 DB")
}

func TestSettingService_GPT61SolLongContextExemptionExpiredServesStaleThenRefreshes(t *testing.T) {
	ctx := context.Background()
	repo := &settingRepoStub{values: map[string]string{
		SettingKeyOpenAIGPT61SolLongContextBillingEnabled: "true",
	}}
	svc := NewSettingService(repo, nil)
	svc.gpt61SolLongContextExemptionCache.Store(&cachedOpenAIGPT61SolLongContextExemption{
		exemption: openAIGPT61SolLongContextExemption{enabled: false},
		expiresAt: time.Now().Add(-time.Second).UnixNano(),
	})

	require.False(t, svc.GetOpenAIGPT61SolLongContextExemption(ctx).enabled, "过期后先返回旧值")

	svc.refreshOpenAIGPT61SolLongContextExemption(ctx)
	require.True(t, svc.GetOpenAIGPT61SolLongContextExemption(ctx).enabled, "刷新后取到新值")
}

func TestSettingService_GPT61SolLongContextExemptionRefreshFailureKeepsPrior(t *testing.T) {
	ctx := context.Background()
	// 仓储直接报错：旧快照必须保留，不得静默翻成默认值（会改变计费方向）。
	repo := &settingRepoStub{err: errors.New("db down")}
	svc := NewSettingService(repo, nil)
	svc.gpt61SolLongContextExemptionCache.Store(&cachedOpenAIGPT61SolLongContextExemption{
		exemption: openAIGPT61SolLongContextExemption{
			enabled:    true,
			accountIDs: map[int64]struct{}{7: {}},
		},
		expiresAt: time.Now().Add(-time.Second).UnixNano(),
	})

	require.True(t, svc.GetOpenAIGPT61SolLongContextExemption(ctx).enabled, "过期后先返回旧快照")
	svc.refreshOpenAIGPT61SolLongContextExemption(ctx)

	cached, _ := svc.gpt61SolLongContextExemptionCache.Load().(*cachedOpenAIGPT61SolLongContextExemption)
	require.NotNil(t, cached)
	require.True(t, cached.exemption.enabled, "刷新失败必须保持旧快照")
	_, ok := cached.exemption.accountIDs[7]
	require.True(t, ok, "刷新失败必须保持旧名单")
	require.Less(t, cached.expiresAt, time.Now().Add(openAIGPT61SolLongContextExemptionCacheTTL).UnixNano(),
		"刷新失败必须改用短 TTL 尽快重试")
}

func TestSettingService_GPT61SolLongContextExemptionInvalidConfigUsesShortTTL(t *testing.T) {
	ctx := context.Background()
	repo := &settingRepoStub{values: map[string]string{
		SettingKeyOpenAIGPT61SolLongContextBillingEnabled:    "true",
		SettingKeyOpenAIGPT61SolLongContextBillingAccountIDs: "not-json",
	}}
	svc := NewSettingService(repo, nil)

	require.False(t, svc.GetOpenAIGPT61SolLongContextExemption(ctx).enabled,
		"配置损坏必须整体按不豁免，禁止退化成全员豁免")

	cached, _ := svc.gpt61SolLongContextExemptionCache.Load().(*cachedOpenAIGPT61SolLongContextExemption)
	require.NotNil(t, cached)
	require.Less(t, cached.expiresAt, time.Now().Add(openAIGPT61SolLongContextExemptionCacheTTL).UnixNano(),
		"配置损坏必须改用短 TTL 尽快重试")
}

// gpt61SolExemptionRacingSettingRepoStub 在读取途中触发回调，用于确定性复现
// 「刷新在途、期间管理端保存」的交错。
type gpt61SolExemptionRacingSettingRepoStub struct {
	settingRepoStub
	onGetValue func()
}

func (s *gpt61SolExemptionRacingSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if s.onGetValue != nil {
		s.onGetValue()
	}
	return s.settingRepoStub.GetValue(ctx, key)
}

func TestSettingService_GPT61SolLongContextExemptionStaleRefreshKeepsNewerWrite(t *testing.T) {
	ctx := context.Background()
	// DB 中仍是旧值（不豁免）；刷新读到它之前，管理端保存路径已落库新值并递增代次。
	racing := &gpt61SolExemptionRacingSettingRepoStub{
		settingRepoStub: settingRepoStub{values: map[string]string{
			SettingKeyOpenAIGPT61SolLongContextBillingEnabled: "false",
		}},
	}
	svc := NewSettingService(racing, nil)
	racing.onGetValue = func() {
		generation := svc.gpt61SolLongContextExemptionGen.Add(1)
		svc.gpt61SolLongContextExemptionCache.Store(&cachedOpenAIGPT61SolLongContextExemption{
			exemption:  openAIGPT61SolLongContextExemption{enabled: true},
			expiresAt:  time.Now().Add(time.Minute).UnixNano(),
			generation: generation,
		})
	}

	svc.refreshOpenAIGPT61SolLongContextExemption(ctx)

	cached, _ := svc.gpt61SolLongContextExemptionCache.Load().(*cachedOpenAIGPT61SolLongContextExemption)
	require.NotNil(t, cached)
	require.True(t, cached.exemption.enabled, "在途刷新不得用陈旧 DB 值覆盖更新的写入")
}

func TestSettingService_UpdateSettingsGPT61SolLongContextExemption(t *testing.T) {
	ctx := context.Background()
	enabledKey := SettingKeyOpenAIGPT61SolLongContextBillingEnabled
	idsKey := SettingKeyOpenAIGPT61SolLongContextBillingAccountIDs
	repo := &gpt61SolExemptionSettingRepoStub{settingRepoStub: settingRepoStub{values: map[string]string{}}}
	svc := NewSettingService(repo, &config.Config{})

	require.NoError(t, svc.UpdateSettings(ctx, &SystemSettings{
		OpenAIGPT61SolLongContextBillingEnabled:    true,
		OpenAIGPT61SolLongContextBillingAccountIDs: []int64{5, 3, 5, -2},
	}))
	require.Equal(t, "true", repo.values[enabledKey])
	require.Equal(t, "[3,5]", repo.values[idsKey], "写库必须归一化：丢弃非正整数、去重、升序")
	got := svc.GetOpenAIGPT61SolLongContextExemption(ctx)
	require.True(t, got.enabled, "写后缓存必须立即生效")
	require.Equal(t, map[int64]struct{}{3: {}, 5: {}}, got.accountIDs)

	// 空数组 = 清空名单（开关仍打开时即全员豁免）。
	require.NoError(t, svc.UpdateSettings(ctx, &SystemSettings{
		OpenAIGPT61SolLongContextBillingEnabled: true,
	}))
	require.Equal(t, "[]", repo.values[idsKey])
	require.Empty(t, svc.GetOpenAIGPT61SolLongContextExemption(ctx).accountIDs, "写后缓存必须立即生效")
}

func TestSettingService_UpdateSettingsGPT61SolLongContextExemptionOmittingKeepsStoredIDs(t *testing.T) {
	ctx := context.Background()
	idsKey := SettingKeyOpenAIGPT61SolLongContextBillingAccountIDs
	repo := &gpt61SolExemptionSettingRepoStub{settingRepoStub: settingRepoStub{values: map[string]string{
		idsKey: "[9]",
	}}}
	svc := NewSettingService(repo, &config.Config{})

	require.NoError(t, svc.UpdateSettingsOmitting(ctx, &SystemSettings{
		OpenAIGPT61SolLongContextBillingEnabled: true,
	}, OmittedSettingKeys{idsKey: struct{}{}}))

	require.Equal(t, "[9]", repo.values[idsKey], "省略名单字段必须保持库中现值")
	got := svc.GetOpenAIGPT61SolLongContextExemption(ctx)
	require.True(t, got.enabled)
	_, ok := got.accountIDs[9]
	require.True(t, ok, "省略字段后缓存必须保留库中名单")
}

func TestParseSettings_GPT61SolLongContextExemptionDefaults(t *testing.T) {
	svc := NewSettingService(&settingRepoStub{}, &config.Config{})
	parsed := svc.parseSettings(map[string]string{})
	require.False(t, parsed.OpenAIGPT61SolLongContextBillingEnabled, "键缺失必须解析为默认不豁免")
	require.Empty(t, parsed.OpenAIGPT61SolLongContextBillingAccountIDs)

	parsed = svc.parseSettings(map[string]string{
		SettingKeyOpenAIGPT61SolLongContextBillingEnabled:    "true",
		SettingKeyOpenAIGPT61SolLongContextBillingAccountIDs: "[7,3]",
	})
	require.True(t, parsed.OpenAIGPT61SolLongContextBillingEnabled)
	require.Equal(t, []int64{3, 7}, parsed.OpenAIGPT61SolLongContextBillingAccountIDs)

	// 非法 JSON 只回退为空名单供展示，不改动开关解析（计费侧由缓存按不豁免处理），
	// 但必须置派生 invalid 标志，供管理端保存路径拒绝把损坏值回写成空名单。
	parsed = svc.parseSettings(map[string]string{
		SettingKeyOpenAIGPT61SolLongContextBillingEnabled:    "true",
		SettingKeyOpenAIGPT61SolLongContextBillingAccountIDs: "{",
	})
	require.True(t, parsed.OpenAIGPT61SolLongContextBillingEnabled)
	require.Empty(t, parsed.OpenAIGPT61SolLongContextBillingAccountIDs)
	require.True(t, parsed.OpenAIGPT61SolLongContextBillingAccountIDsInvalid)

	parsed = svc.parseSettings(map[string]string{
		SettingKeyOpenAIGPT61SolLongContextBillingAccountIDs: "[7]",
	})
	require.False(t, parsed.OpenAIGPT61SolLongContextBillingAccountIDsInvalid, "合法名单不得标记 invalid")
	require.Equal(t, []int64{7}, parsed.OpenAIGPT61SolLongContextBillingAccountIDs)

	// 非法 ID（非正整数）同样按损坏处理。
	parsed = svc.parseSettings(map[string]string{
		SettingKeyOpenAIGPT61SolLongContextBillingAccountIDs: "[0]",
	})
	require.True(t, parsed.OpenAIGPT61SolLongContextBillingAccountIDsInvalid)
}

// 损坏态下省略该键的写入必须保留库中损坏值，且写后快照仍按不豁免 fail closed
// —— 否则「开关打开 + 空名单」会被缓存成全员豁免（计划 §3 禁止的少收费方向）。
func TestSettingService_UpdateSettingsGPT61SolLongContextExemptionCorruptListStaysFailClosed(t *testing.T) {
	ctx := context.Background()
	enabledKey := SettingKeyOpenAIGPT61SolLongContextBillingEnabled
	idsKey := SettingKeyOpenAIGPT61SolLongContextBillingAccountIDs
	const corrupt = "{not-json"
	repo := &gpt61SolExemptionSettingRepoStub{settingRepoStub: settingRepoStub{values: map[string]string{
		enabledKey: "true",
		idsKey:     corrupt,
	}}}
	svc := NewSettingService(repo, &config.Config{})

	stored, err := svc.GetAllSettings(ctx)
	require.NoError(t, err)
	require.True(t, stored.OpenAIGPT61SolLongContextBillingAccountIDsInvalid)

	// 管理端在损坏态下省略该键（setting_handler_update.go 的守卫）。
	require.NoError(t, svc.UpdateSettingsOmitting(ctx, stored, OmittedSettingKeys{idsKey: struct{}{}}))
	require.Equal(t, corrupt, repo.values[idsKey], "损坏值不得被回写成空名单")

	got := svc.GetOpenAIGPT61SolLongContextExemption(ctx)
	require.False(t, got.enabled, "损坏态下写后快照必须 fail closed")
	require.Empty(t, got.accountIDs)
}

func TestParseOpenAIGPT61SolLongContextBillingAccountIDs(t *testing.T) {
	for _, tt := range []struct {
		name    string
		raw     string
		want    []int64
		wantErr bool
	}{
		{name: "空串", raw: "", want: []int64{}},
		{name: "空白", raw: "  ", want: []int64{}},
		{name: "null", raw: "null", want: []int64{}},
		{name: "空数组", raw: "[]", want: []int64{}},
		{name: "去重升序", raw: "[5,3,5]", want: []int64{3, 5}},
		{name: "非法 JSON", raw: "[1,", wantErr: true},
		{name: "非整数", raw: `[1.5]`, wantErr: true},
		{name: "字符串 ID", raw: `["1"]`, wantErr: true},
		{name: "零 ID", raw: "[0]", wantErr: true},
		{name: "负 ID", raw: "[-1]", wantErr: true},
		{name: "对象", raw: `{"a":1}`, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseOpenAIGPT61SolLongContextBillingAccountIDs(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestValidateOpenAIGPT61SolLongContextBillingAccountIDs(t *testing.T) {
	require.NoError(t, ValidateOpenAIGPT61SolLongContextBillingAccountIDs(nil))
	require.NoError(t, ValidateOpenAIGPT61SolLongContextBillingAccountIDs([]int64{1, 2}))
	require.Error(t, ValidateOpenAIGPT61SolLongContextBillingAccountIDs([]int64{0}))
	require.Error(t, ValidateOpenAIGPT61SolLongContextBillingAccountIDs([]int64{-1}))

	over := make([]int64, OpenAIGPT61SolLongContextBillingAccountIDsMax+1)
	for i := range over {
		over[i] = int64(i + 1)
	}
	require.Error(t, ValidateOpenAIGPT61SolLongContextBillingAccountIDs(over))
	require.NoError(t, ValidateOpenAIGPT61SolLongContextBillingAccountIDs(over[:len(over)-1]))
}

func TestMarshalOpenAIGPT61SolLongContextBillingAccountIDs(t *testing.T) {
	require.Equal(t, "[]", MarshalOpenAIGPT61SolLongContextBillingAccountIDs(nil))
	require.Equal(t, "[]", MarshalOpenAIGPT61SolLongContextBillingAccountIDs([]int64{}))
	require.Equal(t, "[3,5]", MarshalOpenAIGPT61SolLongContextBillingAccountIDs([]int64{5, 3, 0}))
}

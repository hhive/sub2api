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

func TestSettingService_GetOpenAIGPT61SolLongContextBillingEnabled(t *testing.T) {
	for _, tt := range []struct {
		name   string
		values map[string]string
		err    error
		want   bool
	}{
		{name: "缺键默认不应用", values: map[string]string{}, want: false},
		{name: "显式 true", values: map[string]string{SettingKeyOpenAIGPT61SolLongContextBillingEnabled: "true"}, want: true},
		{name: "显式 false", values: map[string]string{SettingKeyOpenAIGPT61SolLongContextBillingEnabled: "false"}, want: false},
		{name: "非布尔值按默认不应用", values: map[string]string{SettingKeyOpenAIGPT61SolLongContextBillingEnabled: "yes"}, want: false},
		{name: "DB 错误按默认不应用", values: map[string]string{SettingKeyOpenAIGPT61SolLongContextBillingEnabled: "true"}, err: errors.New("db down"), want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewSettingService(&settingRepoStub{values: tt.values, err: tt.err}, nil)
			require.Equal(t, tt.want, svc.GetOpenAIGPT61SolLongContextBillingEnabled(context.Background()))
		})
	}

	// nil 接收者：计费路径可能缺设置服务，必须按缺省处理而不是 panic。
	var nilSvc *SettingService
	require.False(t, nilSvc.GetOpenAIGPT61SolLongContextBillingEnabled(context.Background()))
}

func TestSettingService_GPT61SolLongContextBillingEnabledCache(t *testing.T) {
	ctx := context.Background()
	repo := &settingRepoStub{values: map[string]string{SettingKeyOpenAIGPT61SolLongContextBillingEnabled: "true"}}
	svc := NewSettingService(repo, nil)

	require.True(t, svc.GetOpenAIGPT61SolLongContextBillingEnabled(ctx))
	require.True(t, svc.GetOpenAIGPT61SolLongContextBillingEnabled(ctx))
	require.Equal(t, 1, repo.getValueCalls, "TTL 内重复读取不得回查 DB")
}

func TestSettingService_GPT61SolLongContextBillingEnabledRefreshFailureKeepsPrior(t *testing.T) {
	ctx := context.Background()
	// 仓储直接报错：旧值必须保留，不得静默翻成默认值。
	repo := &settingRepoStub{
		values: map[string]string{SettingKeyOpenAIGPT61SolLongContextBillingEnabled: "true"},
		err:    errors.New("db down"),
	}
	svc := NewSettingService(repo, nil)
	svc.gpt61SolLongContextBillingEnabledCache.Store(&cachedOpenAIGPT61SolLongContextBillingEnabled{
		enabled:   true,
		expiresAt: time.Now().Add(-time.Second).UnixNano(),
	})

	require.True(t, svc.GetOpenAIGPT61SolLongContextBillingEnabled(ctx), "过期后先返回旧值")
	svc.refreshOpenAIGPT61SolLongContextBillingEnabled(ctx)

	cached, _ := svc.gpt61SolLongContextBillingEnabledCache.Load().(*cachedOpenAIGPT61SolLongContextBillingEnabled)
	require.NotNil(t, cached)
	require.True(t, cached.enabled, "刷新失败必须保持旧值")
	require.Less(t, cached.expiresAt, time.Now().Add(openAIGPT61SolLongContextBillingEnabledCacheTTL).UnixNano(),
		"刷新失败必须改用短 TTL 尽快重试")
}

// gpt61SolRacingSettingRepoStub 在读取途中触发回调，用于确定性复现
// 「刷新在途、期间管理端保存」的交错。
type gpt61SolRacingSettingRepoStub struct {
	settingRepoStub
	onGetValue func()
}

func (s *gpt61SolRacingSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if s.onGetValue != nil {
		s.onGetValue()
	}
	return s.settingRepoStub.GetValue(ctx, key)
}

func TestSettingService_GPT61SolLongContextBillingEnabledStaleRefreshKeepsNewerWrite(t *testing.T) {
	ctx := context.Background()
	// DB 中仍是旧值 false；刷新读到它之前，管理端保存路径已落库新值并递增代次。
	racing := &gpt61SolRacingSettingRepoStub{
		settingRepoStub: settingRepoStub{values: map[string]string{
			SettingKeyOpenAIGPT61SolLongContextBillingEnabled: "false",
		}},
	}
	svc := NewSettingService(racing, nil)
	racing.onGetValue = func() {
		generation := svc.gpt61SolLongContextBillingEnabledGen.Add(1)
		svc.gpt61SolLongContextBillingEnabledCache.Store(&cachedOpenAIGPT61SolLongContextBillingEnabled{
			enabled:    true,
			expiresAt:  time.Now().Add(time.Minute).UnixNano(),
			generation: generation,
		})
	}

	svc.refreshOpenAIGPT61SolLongContextBillingEnabled(ctx)

	cached, _ := svc.gpt61SolLongContextBillingEnabledCache.Load().(*cachedOpenAIGPT61SolLongContextBillingEnabled)
	require.NotNil(t, cached)
	require.True(t, cached.enabled, "在途刷新不得用陈旧 DB 值覆盖更新的写入")
}

// gpt61SolSettingRepoStub 支持写入的设置仓储桩：settingRepoStub.SetMultiple 会 panic。
type gpt61SolSettingRepoStub struct {
	settingRepoStub
}

func (s *gpt61SolSettingRepoStub) SetMultiple(ctx context.Context, updates map[string]string) error {
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

func TestSettingService_UpdateSettingsGPT61SolLongContextBilling(t *testing.T) {
	ctx := context.Background()
	repo := &gpt61SolSettingRepoStub{settingRepoStub: settingRepoStub{values: map[string]string{}}}
	svc := NewSettingService(repo, &config.Config{})

	require.NoError(t, svc.UpdateSettings(ctx, &SystemSettings{OpenAIGPT61SolLongContextBillingEnabled: true}))
	require.Equal(t, "true", repo.values[SettingKeyOpenAIGPT61SolLongContextBillingEnabled])
	require.True(t, svc.GetOpenAIGPT61SolLongContextBillingEnabled(ctx), "写后缓存必须立即生效")

	require.NoError(t, svc.UpdateSettings(ctx, &SystemSettings{OpenAIGPT61SolLongContextBillingEnabled: false}))
	require.Equal(t, "false", repo.values[SettingKeyOpenAIGPT61SolLongContextBillingEnabled])
	require.False(t, svc.GetOpenAIGPT61SolLongContextBillingEnabled(ctx), "写后缓存必须立即生效")
}

func TestParseSettings_GPT61SolLongContextBillingDefault(t *testing.T) {
	svc := NewSettingService(&settingRepoStub{}, &config.Config{})
	require.False(t, svc.parseSettings(map[string]string{}).OpenAIGPT61SolLongContextBillingEnabled,
		"键缺失必须解析为默认不应用")
	require.True(t, svc.parseSettings(map[string]string{
		SettingKeyOpenAIGPT61SolLongContextBillingEnabled: "true",
	}).OpenAIGPT61SolLongContextBillingEnabled)
}

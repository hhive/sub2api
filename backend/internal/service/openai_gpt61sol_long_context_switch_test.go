//go:build unit

package service

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// newGPT61SolSwitchSettingService 构造只带 GPT-6.1 Sol 长上下文计费开关的设置服务。
// 未装配（nil）表示走缺省 = 不应用官方长上下文阶梯。
func newGPT61SolSwitchSettingService(enabled bool) *SettingService {
	return NewSettingService(&settingRepoStub{values: map[string]string{
		SettingKeyOpenAIGPT61SolLongContextBillingEnabled: strconv.FormatBool(enabled),
	}}, nil)
}

func TestOpenAIGPT61SolAPILongContextExempt(t *testing.T) {
	apiKeyAccount := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	oauthAccount := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	setupTokenAccount := &Account{Platform: PlatformOpenAI, Type: AccountTypeSetupToken}
	grokAccount := &Account{Platform: PlatformGrok, Type: AccountTypeAPIKey}

	for _, tt := range []struct {
		name           string
		account        *Account
		result         *OpenAIForwardResult
		billingEnabled bool
		want           bool
	}{
		{name: "开关关闭时 apikey 账号豁免", account: apiKeyAccount, result: &OpenAIForwardResult{Model: "gpt-6.1-sol"}, want: true},
		{name: "开关开启时不豁免", account: apiKeyAccount, result: &OpenAIForwardResult{Model: "gpt-6.1-sol"}, billingEnabled: true},
		{name: "别名与 effort 后缀仍识别", account: apiKeyAccount, result: &OpenAIForwardResult{Model: "openai/GPT_6.1_SOL-max"}, want: true},
		{name: "压缩后缀仍识别", account: apiKeyAccount, result: &OpenAIForwardResult{Model: "gpt-6.1-sol-openai-compact"}, want: true},
		{name: "映射到目标仍识别", account: apiKeyAccount, result: &OpenAIForwardResult{Model: "custom", UpstreamModel: "gpt-6.1-sol"}, want: true},
		{name: "映射离开目标不豁免", account: apiKeyAccount, result: &OpenAIForwardResult{Model: "gpt-6.1-sol", UpstreamModel: "gpt-6-sol"}},
		{name: "其它模型不豁免", account: apiKeyAccount, result: &OpenAIForwardResult{Model: "gpt-6-sol"}},
		{name: "相似名称不豁免", account: apiKeyAccount, result: &OpenAIForwardResult{Model: "gpt-6.1-sol-preview"}},
		{name: "订阅账号由订阅豁免处理", account: oauthAccount, result: &OpenAIForwardResult{Model: "gpt-6.1-sol"}},
		{name: "订阅访问令牌由订阅豁免处理", account: setupTokenAccount, result: &OpenAIForwardResult{Model: "gpt-6.1-sol"}},
		{name: "其它平台不豁免", account: grokAccount, result: &OpenAIForwardResult{Model: "gpt-6.1-sol"}},
		{name: "空账号不豁免", result: &OpenAIForwardResult{Model: "gpt-6.1-sol"}},
		{name: "空结果不豁免", account: apiKeyAccount},
	} {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, openAIGPT61SolAPILongContextExempt(tt.account, tt.result, tt.billingEnabled))
		})
	}
}

// 覆盖真实扣款：同一账号/分组门槛下，只有全局开关变化时客户价必须跟着变。
func TestOpenAIGatewayServiceRecordUsage_GPT61SolAPISwitch(t *testing.T) {
	apiKeyAccount := func() *Account {
		return &Account{ID: 3, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
			Extra: map[string]any{openAILongContextBillingEnabledKey: true}}
	}
	oauthAccount := func() *Account {
		return &Account{ID: 3, Platform: PlatformOpenAI, Type: AccountTypeOAuth,
			Extra: map[string]any{openAILongContextBillingEnabledKey: true}}
	}

	for _, tt := range []struct {
		name     string
		settings *SettingService
		account  *Account
		model    string
		wantLong bool
	}{
		{name: "开关关闭不触发", settings: newGPT61SolSwitchSettingService(false), account: apiKeyAccount(), model: "gpt-6.1-sol"},
		{name: "未装配设置服务按缺省不触发", account: apiKeyAccount(), model: "gpt-6.1-sol"},
		{name: "开关开启按官方阶梯收费", settings: newGPT61SolSwitchSettingService(true), account: apiKeyAccount(), model: "gpt-6.1-sol", wantLong: true},
		{name: "开关关闭不影响其它模型", settings: newGPT61SolSwitchSettingService(false), account: apiKeyAccount(), model: "gpt-6-sol", wantLong: true},
		{name: "订阅账号不受开关影响", settings: newGPT61SolSwitchSettingService(true), account: oauthAccount(), model: "gpt-6.1-sol"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
			userRepo := &openAIRecordUsageUserRepoStub{}
			svc := newOpenAIRecordUsageServiceForTest(usageRepo, userRepo, &openAIRecordUsageSubRepoStub{}, nil)
			svc.settingService = tt.settings
			apiKey := openAIRecordUsageAPIKeyWithGroup(svc, 1, true)
			apiKey.GroupID = i64p(1)
			apiKey.Group.RateMultiplier = 1

			err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
				Result: &OpenAIForwardResult{RequestID: tt.name, Model: tt.model, Duration: time.Second,
					Usage: OpenAIUsage{InputTokens: 300000, OutputTokens: 2000}},
				APIKey: apiKey, User: &User{ID: 2}, Account: tt.account,
			})
			require.NoError(t, err)

			wantInput, wantOutput := 300000*2e-6, 2000*10e-6
			if tt.wantLong {
				wantInput *= 2
				wantOutput *= 1.5
			}
			log := usageRepo.lastLog
			require.NotNil(t, log)
			require.Equal(t, tt.wantLong, log.LongContextBillingApplied)
			require.InDelta(t, wantInput, log.InputCost, 1e-12)
			require.InDelta(t, wantOutput, log.OutputCost, 1e-12)
			require.InDelta(t, wantInput+wantOutput, log.TotalCost, 1e-12)
		})
	}
}

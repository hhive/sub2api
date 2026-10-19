//go:build unit

package service

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// newGPT61SolExemptionSettingService 构造只带 GPT-6.1 Sol 长上下文豁免设置的设置服务。
// enabled = true 表示豁免（开关打开），false 表示照常按官方阶梯收费。
// 未装配（nil）设置服务表示走缺省 = 不豁免。
func newGPT61SolExemptionSettingService(enabled bool, ids []int64) *SettingService {
	return NewSettingService(&settingRepoStub{values: map[string]string{
		SettingKeyOpenAIGPT61SolLongContextBillingEnabled:    strconv.FormatBool(enabled),
		SettingKeyOpenAIGPT61SolLongContextBillingAccountIDs: MarshalOpenAIGPT61SolLongContextBillingAccountIDs(ids),
	}}, nil)
}

func gpt61SolExemption(enabled bool, ids ...int64) openAIGPT61SolLongContextExemption {
	return openAIGPT61SolLongContextExemption{enabled: enabled, accountIDs: openAIGPT61SolLongContextAccountIDSet(ids)}
}

func TestOpenAIGPT61SolAPILongContextExempt(t *testing.T) {
	apiKeyAccount := &Account{ID: 3, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	otherAPIKeyAccount := &Account{ID: 4, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	oauthAccount := &Account{ID: 5, Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	setupTokenAccount := &Account{ID: 6, Platform: PlatformOpenAI, Type: AccountTypeSetupToken}
	grokAccount := &Account{ID: 7, Platform: PlatformGrok, Type: AccountTypeAPIKey}

	for _, tt := range []struct {
		name      string
		account   *Account
		result    *OpenAIForwardResult
		exemption openAIGPT61SolLongContextExemption
		want      bool
	}{
		{name: "开关关闭时不豁免（默认）", account: apiKeyAccount, result: &OpenAIForwardResult{Model: "gpt-6.1-sol"}},
		{name: "开关关闭时名单被忽略", account: apiKeyAccount, result: &OpenAIForwardResult{Model: "gpt-6.1-sol"}, exemption: gpt61SolExemption(false, 3)},
		{name: "开关打开且名单为空 = 全员豁免", account: apiKeyAccount, result: &OpenAIForwardResult{Model: "gpt-6.1-sol"}, exemption: gpt61SolExemption(true), want: true},
		{name: "开关打开且命中名单", account: apiKeyAccount, result: &OpenAIForwardResult{Model: "gpt-6.1-sol"}, exemption: gpt61SolExemption(true, 3, 9), want: true},
		{name: "开关打开但未命中名单", account: otherAPIKeyAccount, result: &OpenAIForwardResult{Model: "gpt-6.1-sol"}, exemption: gpt61SolExemption(true, 3)},
		{name: "别名与 effort 后缀仍识别", account: apiKeyAccount, result: &OpenAIForwardResult{Model: "openai/GPT_6.1_SOL-max"}, exemption: gpt61SolExemption(true, 3), want: true},
		{name: "压缩后缀仍识别", account: apiKeyAccount, result: &OpenAIForwardResult{Model: "gpt-6.1-sol-openai-compact"}, exemption: gpt61SolExemption(true), want: true},
		{name: "映射到目标仍识别", account: apiKeyAccount, result: &OpenAIForwardResult{Model: "custom", UpstreamModel: "gpt-6.1-sol"}, exemption: gpt61SolExemption(true, 3), want: true},
		{name: "映射离开目标不豁免", account: apiKeyAccount, result: &OpenAIForwardResult{Model: "gpt-6.1-sol", UpstreamModel: "gpt-6-sol"}, exemption: gpt61SolExemption(true)},
		{name: "其它模型不豁免", account: apiKeyAccount, result: &OpenAIForwardResult{Model: "gpt-6-sol"}, exemption: gpt61SolExemption(true)},
		{name: "相似名称不豁免", account: apiKeyAccount, result: &OpenAIForwardResult{Model: "gpt-6.1-sol-preview"}, exemption: gpt61SolExemption(true)},
		{name: "订阅账号由订阅豁免处理", account: oauthAccount, result: &OpenAIForwardResult{Model: "gpt-6.1-sol"}, exemption: gpt61SolExemption(true)},
		{name: "订阅访问令牌由订阅豁免处理", account: setupTokenAccount, result: &OpenAIForwardResult{Model: "gpt-6.1-sol"}, exemption: gpt61SolExemption(true)},
		{name: "其它平台不豁免", account: grokAccount, result: &OpenAIForwardResult{Model: "gpt-6.1-sol"}, exemption: gpt61SolExemption(true)},
		{name: "空账号不豁免", result: &OpenAIForwardResult{Model: "gpt-6.1-sol"}, exemption: gpt61SolExemption(true)},
		{name: "空结果不豁免", account: apiKeyAccount, exemption: gpt61SolExemption(true)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, openAIGPT61SolAPILongContextExempt(tt.account, tt.result, tt.exemption))
		})
	}
}

// 覆盖真实扣款：同一账号/分组门槛下，开关与名单变化时客户价与账号成本统计必须同步变化。
func TestOpenAIGatewayServiceRecordUsage_GPT61SolAPIExemptionSwitch(t *testing.T) {
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
		{name: "开关关闭不豁免", settings: newGPT61SolExemptionSettingService(false, nil), account: apiKeyAccount(), model: "gpt-6.1-sol", wantLong: true},
		{name: "开关关闭时名单被忽略", settings: newGPT61SolExemptionSettingService(false, []int64{3}), account: apiKeyAccount(), model: "gpt-6.1-sol", wantLong: true},
		{name: "未装配设置服务按缺省不豁免", account: apiKeyAccount(), model: "gpt-6.1-sol", wantLong: true},
		{name: "开关打开且名单为空则豁免", settings: newGPT61SolExemptionSettingService(true, nil), account: apiKeyAccount(), model: "gpt-6.1-sol"},
		{name: "开关打开且命中名单则豁免", settings: newGPT61SolExemptionSettingService(true, []int64{3, 9}), account: apiKeyAccount(), model: "gpt-6.1-sol"},
		{name: "开关打开但未命中名单照常收费", settings: newGPT61SolExemptionSettingService(true, []int64{9}), account: apiKeyAccount(), model: "gpt-6.1-sol", wantLong: true},
		{name: "开关打开不影响其它模型", settings: newGPT61SolExemptionSettingService(true, nil), account: apiKeyAccount(), model: "gpt-6-sol", wantLong: true},
		{name: "订阅账号不受开关影响", settings: newGPT61SolExemptionSettingService(true, []int64{9}), account: oauthAccount(), model: "gpt-6.1-sol"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
			userRepo := &openAIRecordUsageUserRepoStub{}
			svc := newOpenAIRecordUsageServiceForTest(usageRepo, userRepo, &openAIRecordUsageSubRepoStub{}, nil)
			svc.settingService = tt.settings
			apiKey := openAIRecordUsageAPIKeyWithGroup(svc, 1, true)
			apiKey.GroupID = i64p(1)
			apiKey.Group.RateMultiplier = 1.1
			svc.channelService = newTestChannelServiceForStats(t, &Channel{ID: 1, Status: StatusActive}, 1, PlatformOpenAI)

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
			wantTotal := wantInput + wantOutput
			log := usageRepo.lastLog
			require.NotNil(t, log)
			require.Equal(t, tt.wantLong, log.LongContextBillingApplied)
			require.InDelta(t, wantInput, log.InputCost, 1e-12)
			require.InDelta(t, wantOutput, log.OutputCost, 1e-12)
			require.InDelta(t, wantTotal, log.TotalCost, 1e-12)
			require.InDelta(t, wantTotal*1.1, log.ActualCost, 1e-12)
			require.Equal(t, 1, userRepo.deductCalls)
			require.InDelta(t, wantTotal*1.1, userRepo.lastAmount, 1e-12)
			// 账号成本统计与客户价共用同一豁免判据，两侧必须一致。
			require.NotNil(t, log.AccountStatsCost)
			require.InDelta(t, wantTotal, *log.AccountStatsCost, 1e-12)
		})
	}
}

// 影子账号一律由订阅豁免覆盖，账号名单对其无意义。
//
// 证明力边界：resolveCredentialAccount 强制影子母账号必须是 OpenAI OAuth
// （见 credential_shadow.go:29-31），而订阅豁免对 OAuth 账号先短路，所以本用例**无法**
// 在「子账号 ID 是否命中名单」上做出可区分断言 —— 把判据改成按子账号判定，结论同样成立。
// 它守的是「影子账号按母账号解析后计费、且不因名单设置崩掉/误收费」。
func TestOpenAIGatewayServiceRecordUsage_GPT61SolShadowAccountCoveredBySubscriptionExemption(t *testing.T) {
	parentID := int64(4)
	parent := &Account{ID: parentID, Platform: PlatformOpenAI, Type: AccountTypeOAuth,
		Extra: map[string]any{openAILongContextBillingEnabledKey: true}}
	child := &Account{ID: 3, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		ParentAccountID: &parentID, QuotaDimension: QuotaDimensionSpark,
		Extra: map[string]any{openAILongContextBillingEnabledKey: true}}

	usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
	svc := newOpenAIRecordUsageServiceForTest(usageRepo, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil)
	// 名单里放的是子账号 ID；计费按母账号（订阅豁免）判定，因此仍然豁免。
	svc.settingService = newGPT61SolExemptionSettingService(true, []int64{3})
	accountRepo := &openAIRecordUsageAccountRepoStub{account: parent}
	svc.accountRepo = accountRepo

	err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
		Result: &OpenAIForwardResult{RequestID: "shadow", Model: "gpt-6.1-sol", Duration: time.Second,
			Usage: OpenAIUsage{InputTokens: 300000, OutputTokens: 2000}},
		APIKey: openAIRecordUsageAPIKeyWithGroup(svc, 1, true), User: &User{ID: 2}, Account: child,
	})
	require.NoError(t, err)
	require.Equal(t, 1, accountRepo.calls, "影子账号必须解析到母账号后再计费")
	require.False(t, usageRepo.lastLog.LongContextBillingApplied)
	require.InDelta(t, 300000*2e-6+2000*10e-6, usageRepo.lastLog.TotalCost, 1e-12)
}

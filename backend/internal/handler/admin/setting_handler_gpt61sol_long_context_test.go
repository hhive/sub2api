package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 覆盖 GPT-6.1 Sol 长上下文豁免设置的完整往返：请求省略必须保持现值（否则每次保存都会把
// 开关与名单重置成默认值），显式值必须写入并归一化，非法值必须 400 且不写库，
// 响应必须回带两个字段供前端回填。
func TestSettingHandler_GPT61SolLongContextExemptionRoundTrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enabledKey := service.SettingKeyOpenAIGPT61SolLongContextBillingEnabled
	idsKey := service.SettingKeyOpenAIGPT61SolLongContextBillingAccountIDs

	for _, tt := range []struct {
		name     string
		initial  map[string]string
		body     string
		stored   string
		value    any
		storedID string
		idValue  any
		status   int
	}{
		{
			name: "absent keeps defaults off", body: `{}`,
			stored: "false", value: false, storedID: "[]", idValue: []any{}, status: http.StatusOK,
		},
		{
			name: "enable switch only", body: `{"openai_gpt61sol_long_context_billing_enabled":true}`,
			stored: "true", value: true, storedID: "[]", idValue: []any{}, status: http.StatusOK,
		},
		{
			name: "disable is explicit", initial: map[string]string{enabledKey: "true"},
			body:   `{"openai_gpt61sol_long_context_billing_enabled":false}`,
			stored: "false", value: false, storedID: "[]", idValue: []any{}, status: http.StatusOK,
		},
		{
			name:    "omitted preserves stored switch and ids",
			initial: map[string]string{enabledKey: "true", idsKey: "[9]"},
			body:    `{}`,
			stored:  "true", value: true, storedID: "[9]", idValue: []any{float64(9)}, status: http.StatusOK,
		},
		{
			name:   "account ids normalized on write",
			body:   `{"openai_gpt61sol_long_context_billing_account_ids":[7,3,7]}`,
			stored: "false", value: false, storedID: "[3,7]", idValue: []any{float64(3), float64(7)}, status: http.StatusOK,
		},
		{
			name:    "empty array clears stored ids",
			initial: map[string]string{idsKey: "[9]"}, body: `{"openai_gpt61sol_long_context_billing_account_ids":[]}`,
			stored: "false", value: false, storedID: "[]", idValue: []any{}, status: http.StatusOK,
		},
		{
			name:    "null is equivalent to omitted",
			initial: map[string]string{idsKey: "[9]"}, body: `{"openai_gpt61sol_long_context_billing_account_ids":null}`,
			stored: "false", value: false, storedID: "[9]", idValue: []any{float64(9)}, status: http.StatusOK,
		},
		{
			name: "string is rejected", body: `{"openai_gpt61sol_long_context_billing_enabled":"yes"}`,
			status: http.StatusBadRequest,
		},
		{
			name: "zero id is rejected", body: `{"openai_gpt61sol_long_context_billing_account_ids":[0]}`,
			status: http.StatusBadRequest,
		},
		{
			name: "negative id is rejected", body: `{"openai_gpt61sol_long_context_billing_account_ids":[3,-1]}`,
			status: http.StatusBadRequest,
		},
		{
			name: "too many ids are rejected",
			body: fmt.Sprintf(`{"openai_gpt61sol_long_context_billing_account_ids":%s}`,
				mustJSONInt64Slice(make([]int64, service.OpenAIGPT61SolLongContextBillingAccountIDsMax+1))),
			status: http.StatusBadRequest,
		},
		{
			name: "non-array ids are rejected", body: `{"openai_gpt61sol_long_context_billing_account_ids":"3"}`,
			status: http.StatusBadRequest,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &settingHandlerRepoStub{values: tt.initial}
			svc := service.NewSettingService(repo, &config.Config{})
			handler := NewSettingHandler(svc, nil, nil, nil, nil, nil, nil, nil)

			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewBufferString(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")
			handler.UpdateSettings(c)

			require.Equal(t, tt.status, rec.Code, rec.Body.String())
			if tt.status != http.StatusOK {
				require.Empty(t, repo.lastUpdates)
				return
			}
			require.Equal(t, tt.stored, repo.values[enabledKey])
			require.Equal(t, tt.storedID, repo.values[idsKey])

			var body struct {
				Data map[string]any `json:"data"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.Equal(t, tt.value, body.Data[enabledKey])
			require.Equal(t, tt.idValue, body.Data[idsKey])
		})
	}
}

// 库中名单损坏时保存任意设置不得把「损坏」洗成空名单：空名单 + 开关打开 = 全员豁免，
// 是少收费方向，而计费侧对同一损坏按不豁免 fail closed。
func TestSettingHandler_GPT61SolLongContextExemptionCorruptListIsNotLaundered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enabledKey := service.SettingKeyOpenAIGPT61SolLongContextBillingEnabled
	idsKey := service.SettingKeyOpenAIGPT61SolLongContextBillingAccountIDs
	const corrupt = "{not-json"

	newCorruptHandler := func(t *testing.T) (*SettingHandler, *settingHandlerRepoStub) {
		t.Helper()
		repo := &settingHandlerRepoStub{values: map[string]string{enabledKey: "true", idsKey: corrupt}}
		svc := service.NewSettingService(repo, &config.Config{})
		return NewSettingHandler(svc, nil, nil, nil, nil, nil, nil, nil), repo
	}

	t.Run("omitted field keeps the corrupt value", func(t *testing.T) {
		handler, repo := newCorruptHandler(t)

		// 请求体只改别的设置（旧客户端/脚本/前端表单回传空名单的同效路径）。
		rec := doUpdateSettings(t, handler, map[string]any{"registration_enabled": false}, nil)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, corrupt, repo.values[idsKey], "损坏值不得被回写成空名单")
		require.NotContains(t, repo.lastUpdates, idsKey, "损坏态下该键必须从写入中剔除")
		// 计费侧仍为不豁免由 service 包内的
		// TestSettingService_UpdateSettingsGPT61SolLongContextExemptionCorruptListStaysFailClosed 断言
		// （快照字段不导出，handler 包内无法观测）。
	})

	t.Run("explicit empty array keeps the corrupt value", func(t *testing.T) {
		handler, repo := newCorruptHandler(t)

		// 显式 [] 也被保留：运营者须先写一次非空名单让库值恢复可解析，再清空。
		rec := doUpdateSettings(t, handler, map[string]any{
			"openai_gpt61sol_long_context_billing_account_ids": []int64{},
		}, nil)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, corrupt, repo.values[idsKey])
	})

	t.Run("explicit non-empty list repairs the stored value", func(t *testing.T) {
		handler, repo := newCorruptHandler(t)

		rec := doUpdateSettings(t, handler, map[string]any{
			"openai_gpt61sol_long_context_billing_account_ids": []int64{5, 3},
		}, nil)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, "[3,5]", repo.values[idsKey], "显式非空名单是损坏态下唯一的覆盖途径")
		require.Contains(t, rec.Body.String(), `"openai_gpt61sol_long_context_billing_account_ids":[3,5]`)
	})
}

// GET 必须回带两个字段，供管理端回填开关与账号选择器。
func TestSettingHandler_GPT61SolLongContextExemptionGet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{values: map[string]string{
		service.SettingKeyOpenAIGPT61SolLongContextBillingEnabled:    "true",
		service.SettingKeyOpenAIGPT61SolLongContextBillingAccountIDs: "[7,3]",
	}}
	handler := NewSettingHandler(service.NewSettingService(repo, &config.Config{}), nil, nil, nil, nil, nil, nil, nil)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)
	handler.GetSettings(c)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, true, body.Data[service.SettingKeyOpenAIGPT61SolLongContextBillingEnabled])
	require.Equal(t, []any{float64(3), float64(7)}, body.Data[service.SettingKeyOpenAIGPT61SolLongContextBillingAccountIDs])
}

// 库中名单损坏时 GET 仍必须 200，并把名单回退成空数组作为展示占位（损坏值不暴露给管理端）；
// 计费侧对同一损坏按不豁免 fail closed，保存路径不得把损坏值回写成空名单（见上面的 Update 用例）。
func TestSettingHandler_GPT61SolLongContextExemptionGetCorruptList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{values: map[string]string{
		service.SettingKeyOpenAIGPT61SolLongContextBillingEnabled:    "true",
		service.SettingKeyOpenAIGPT61SolLongContextBillingAccountIDs: "{not-json",
	}}
	handler := NewSettingHandler(service.NewSettingService(repo, &config.Config{}), nil, nil, nil, nil, nil, nil, nil)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)
	handler.GetSettings(c)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, []any{}, body.Data[service.SettingKeyOpenAIGPT61SolLongContextBillingAccountIDs])
}

// 审计 changed 必须命中名单键（增删账号都要留下痕迹）。
func TestDiffSettings_GPT61SolLongContextExemptionAccountIDs(t *testing.T) {
	for _, tt := range []struct {
		name   string
		before []int64
		after  []int64
		want   bool
	}{
		{name: "名单变化", before: []int64{3}, after: []int64{3, 9}, want: true},
		{name: "名单清空", before: []int64{3}, after: []int64{}, want: true},
		{name: "名单不变", before: []int64{3}, after: []int64{3}},
		{name: "都为空", before: nil, after: []int64{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			changed := diffSettings(
				&service.SystemSettings{OpenAIGPT61SolLongContextBillingAccountIDs: tt.before},
				&service.SystemSettings{OpenAIGPT61SolLongContextBillingAccountIDs: tt.after},
				nil, nil, UpdateSettingsRequest{},
			)
			require.Equal(t, tt.want, containsString(changed, service.SettingKeyOpenAIGPT61SolLongContextBillingAccountIDs), "%v", changed)
		})
	}

	// 开关本身的审计命中不能因为新增名单条目而丢失。
	changed := diffSettings(
		&service.SystemSettings{OpenAIGPT61SolLongContextBillingEnabled: false},
		&service.SystemSettings{OpenAIGPT61SolLongContextBillingEnabled: true},
		nil, nil, UpdateSettingsRequest{},
	)
	require.True(t, containsString(changed, service.SettingKeyOpenAIGPT61SolLongContextBillingEnabled), "%v", changed)
}

func mustJSONInt64Slice(ids []int64) string {
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	raw, err := json.Marshal(ids)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

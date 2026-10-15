//go:build unit

package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// useKeyConfigBody 构造一个 use_key_config 请求体；codexOverrides 覆盖 codex 块中的字段。
func useKeyConfigBody(version any, tabs any, codexOverrides map[string]any) map[string]any {
	codex := map[string]any{
		"provider_name":    "OpenAI",
		"model":            "gpt-5.5",
		"review_model":     "",
		"reasoning_effort": "",
	}
	for key, value := range codexOverrides {
		codex[key] = value
	}
	return map[string]any{
		"use_key_config": map[string]any{
			"version": version,
			"enabled": true,
			"platforms": map[string]any{
				"openai": map[string]any{"tabs": tabs, "codex": codex},
			},
		},
	}
}

// 非法 use_key_config 必须在写入前显式 400：这些值会被前端拼进 config.toml 的双引号内，
// 放行等于把注入面存进设置表。
func TestUpdateSettingsUseKeyConfigRejectsInvalid(t *testing.T) {
	cases := []struct {
		name string
		body map[string]any
	}{
		{"version must be 1", useKeyConfigBody(2, []string{"codex"}, nil)},
		{"unknown tab", useKeyConfigBody(1, []string{"codex", "vscode"}, nil)},
		{"model with double quote", useKeyConfigBody(1, []string{"codex"}, map[string]any{"model": `gpt"5.5`})},
		{"model with newline", useKeyConfigBody(1, []string{"codex"}, map[string]any{"model": "gpt-5.5\nrm -rf /"})},
		{"model with backslash", useKeyConfigBody(1, []string{"codex"}, map[string]any{"model": `gpt\5.5`})},
		{"provider name with newline", useKeyConfigBody(1, []string{"codex"}, map[string]any{"provider_name": "Open\nAI"})},
		{"review model with double quote", useKeyConfigBody(1, []string{"codex"}, map[string]any{"review_model": `gpt"5.5`})},
		{"provider name too long", useKeyConfigBody(1, []string{"codex"}, map[string]any{"provider_name": strings.Repeat("p", 65)})},
		{"model too long", useKeyConfigBody(1, []string{"codex"}, map[string]any{"model": strings.Repeat("m", 129)})},
		{"review model too long", useKeyConfigBody(1, []string{"codex"}, map[string]any{"review_model": strings.Repeat("r", 129)})},
		{"reasoning effort not in enum", useKeyConfigBody(1, []string{"codex"}, map[string]any{"reasoning_effort": "ultra"})},
		{"not an object", map[string]any{"use_key_config": "version=1"}},
		{"array instead of object", map[string]any{"use_key_config": []any{1, 2}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newStepUpSwitchTestHandler(t, map[string]string{service.SettingKeyUseKeyConfig: ""})

			rec := doUpdateSettings(t, h, tc.body, nil)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.Equal(t, "", repo.values[service.SettingKeyUseKeyConfig], "非法配置不得落库")
		})
	}
}

// 合法配置落库为规范化 JSON：tabs 去重，空值字段保留为空串（表示回退前端硬编码默认）。
func TestUpdateSettingsUseKeyConfigAcceptsValidAndNormalizes(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{})

	rec := doUpdateSettings(t, h, useKeyConfigBody(1,
		[]string{"codex", "codex-ws", "codex", "opencode", "claude"},
		map[string]any{"model": "gpt-5.5-mini", "reasoning_effort": "high"},
	), nil)

	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"version":1,"enabled":true,"platforms":{"openai":{"tabs":["codex","codex-ws","opencode","claude"],"codex":{"provider_id":"","provider_name":"OpenAI","model":"gpt-5.5-mini","review_model":"","reasoning_effort":"high"}}}}`,
		repo.values[service.SettingKeyUseKeyConfig])
}

// 缺省字段是合法值：只给 version/enabled 也能保存，空 tabs 归一化为空数组。
func TestUpdateSettingsUseKeyConfigAcceptsSparseObject(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{})

	rec := doUpdateSettings(t, h, map[string]any{
		"use_key_config": map[string]any{"version": 1, "enabled": false},
	}, nil)

	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"version":1,"enabled":false,"platforms":{}}`, repo.values[service.SettingKeyUseKeyConfig])
}

// 显式 null 表示清空配置（前端「重置为默认」），必须真正清库而不是静默忽略。
func TestUpdateSettingsUseKeyConfigClearedByNull(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeyUseKeyConfig: `{"version":1,"enabled":true,"platforms":{}}`,
	})

	rec := doUpdateSettings(t, h, map[string]any{"use_key_config": nil}, nil)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "", repo.values[service.SettingKeyUseKeyConfig])
}

// 未提交该字段时保持已存值（整文档 PUT 的部分提交语义）。
func TestUpdateSettingsUseKeyConfigKeptWhenOmitted(t *testing.T) {
	stored := `{"version":1,"enabled":true,"platforms":{}}`
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeyUseKeyConfig: stored,
	})

	rec := doUpdateSettings(t, h, map[string]any{"site_name": "Example Gateway"}, nil)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, stored, repo.values[service.SettingKeyUseKeyConfig])
}

// 审计必须把 use_key_config 变化记进 changed 列表。
func TestDiffSettingsReportsUseKeyConfigChange(t *testing.T) {
	before := &service.SystemSettings{}
	after := &service.SystemSettings{UseKeyConfig: `{"version":1,"enabled":true}`}

	require.Contains(t, diffSettings(before, after, nil, nil, UpdateSettingsRequest{}), "use_key_config")
	require.NotContains(t, diffSettings(before, before, nil, nil, UpdateSettingsRequest{}), "use_key_config")
}

// adminSettingsViewData 调用 GET /api/v1/admin/settings 并返回响应 data 的原始字段。
func adminSettingsViewData(t *testing.T, stored map[string]string) map[string]json.RawMessage {
	t.Helper()
	h, _ := newStepUpSwitchTestHandler(t, stored)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)

	h.GetSettings(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var resp struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	return resp.Data
}

// 管理端必须能区分「未配置」与「配置无效」：前者静默回退硬编码，后者要显式提示
// 「配置无效，已回退默认」。两者都返回 use_key_config = null，靠 use_key_config_invalid 分流。
func TestGetSettingsUseKeyConfigInvalidFlag(t *testing.T) {
	const valid = `{"version":1,"enabled":true,"platforms":{"openai":{"tabs":["codex"],"codex":{"provider_id":"","provider_name":"OpenAI","model":"gpt-5.5-mini","review_model":"","reasoning_effort":""}}}}`

	cases := []struct {
		name        string
		stored      string
		wantConfig  string
		wantInvalid string
	}{
		{name: "unset key is not invalid", stored: "", wantConfig: "null", wantInvalid: "false"},
		{name: "unparsable JSON is invalid", stored: "not json", wantConfig: "null", wantInvalid: "true"},
		{name: "unsupported version is invalid", stored: `{"version":2,"enabled":true,"platforms":{}}`, wantConfig: "null", wantInvalid: "true"},
		{name: "applicable config is exposed and valid", stored: valid, wantConfig: valid, wantInvalid: "false"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := adminSettingsViewData(t, map[string]string{service.SettingKeyUseKeyConfig: tc.stored})

			require.JSONEq(t, tc.wantConfig, string(data["use_key_config"]))
			require.JSONEq(t, tc.wantInvalid, string(data["use_key_config_invalid"]))
		})
	}
}

// provider_id 会以 TOML 裸表键 [model_providers.<id>] 的形式拼进客户端 config.toml，
// 因此只允许裸键字符集；这类错误必须与其他字段的「禁引号」报错可区分。
func TestUpdateSettingsUseKeyConfigRejectsInvalidProviderID(t *testing.T) {
	cases := []struct {
		name  string
		value string
	}{
		{"space", "open ai"},
		{"double quote", `open"ai`},
		{"dot", "open.ai"},
		{"chinese", "中转站"},
		{"newline", "open\nai"},
		{"slash", "open/ai"},
		{"bracket", "open[ai]"},
		{"too long", strings.Repeat("p", 65)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newStepUpSwitchTestHandler(t, map[string]string{service.SettingKeyUseKeyConfig: ""})

			rec := doUpdateSettings(t, h, useKeyConfigBody(1, []string{"codex"}, map[string]any{"provider_id": tc.value}), nil)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.Equal(t, "", repo.values[service.SettingKeyUseKeyConfig], "非法 provider_id 不得落库")
		})
	}
}

// provider_id 为合法裸键时按原值落库（前后空白先 trim）。
func TestUpdateSettingsUseKeyConfigAcceptsProviderID(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{})

	rec := doUpdateSettings(t, h, useKeyConfigBody(1, []string{"codex"}, map[string]any{"provider_id": "  sub2api-codex_2  "}), nil)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, repo.values[service.SettingKeyUseKeyConfig], `"provider_id":"sub2api-codex_2"`)
}

// provider_id 空串或字段缺失都是合法值，表示回退前端内置默认。
func TestUpdateSettingsUseKeyConfigAcceptsEmptyProviderID(t *testing.T) {
	for _, overrides := range []map[string]any{{"provider_id": ""}, {}} {
		h, repo := newStepUpSwitchTestHandler(t, map[string]string{})

		rec := doUpdateSettings(t, h, useKeyConfigBody(1, []string{"codex"}, overrides), nil)

		require.Equal(t, http.StatusOK, rec.Code, "overrides=%v", overrides)
		require.Contains(t, repo.values[service.SettingKeyUseKeyConfig], `"provider_id":""`)
	}
}

// PUT 的响应必须与 GET 对同一库状态给出相同判定：部分更新时 use_key_config 会被原样
// 保留，若 PUT 只按请求值判定，会把库中仍然无效的配置报成 valid，前端随即清掉告警
// （SettingsView 用 updated.use_key_config_invalid === true 覆写告警状态）。
func TestUpdateSettingsResponseMatchesGetForUseKeyConfigInvalid(t *testing.T) {
	stored := map[string]string{service.SettingKeyUseKeyConfig: "not json"}

	h, _ := newStepUpSwitchTestHandler(t, stored)
	rec := doUpdateSettings(t, h, map[string]any{"site_name": "Example Gateway"}, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var putResp struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &putResp))

	getData := adminSettingsViewData(t, stored)

	for _, key := range []string{"use_key_config", "use_key_config_invalid"} {
		require.JSONEq(t, string(getData[key]), string(putResp.Data[key]),
			"同一库状态下 PUT 与 GET 的 %s 必须一致", key)
	}
	require.JSONEq(t, "null", string(putResp.Data["use_key_config"]))
	require.JSONEq(t, "true", string(putResp.Data["use_key_config_invalid"]))
}

// 公开端点与管理端视图对同一存库值的可用性判定必须逐例一致，不能出现
// 「公开端返回可用对象、管理端判为无效」这类两端矛盾。
func TestUseKeyConfigUsabilityJudgeAgreesAcrossEndpoints(t *testing.T) {
	cases := []struct {
		name   string
		stored string
	}{
		{"supported version", `{"version":1,"enabled":true,"platforms":{}}`},
		{"unsupported version", `{"version":2,"enabled":true,"platforms":{}}`},
		{"missing version", `{"enabled":true}`},
		{"non-integer version", `{"version":"1","enabled":true}`},
		{"null version", `{"version":null,"enabled":true}`},
		{"not an object", "not json"},
		{"json array", "[]"},
		{"json null", "null"},
		{"empty", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := service.NewSettingService(
				&settingHandlerRepoStub{values: map[string]string{service.SettingKeyUseKeyConfig: tc.stored}},
				&config.Config{},
			)
			publicSettings, err := svc.GetPublicSettings(context.Background())
			require.NoError(t, err)
			// 公开端可用 ⟺ 输出不是字面 null
			publicUsable := string(publicSettings.UseKeyConfig) != "null"

			parsed, invalid := dto.ParseUseKeyConfigWithValidity(tc.stored)
			adminUsable := parsed != nil

			require.Equal(t, publicUsable, adminUsable, "stored=%q 公开端与管理端可用性判定必须一致", tc.stored)

			// 三态语义：invalid 仅当「存了但不可用」时为 true；未配置（"" / null）不是无效。
			trimmed := strings.TrimSpace(tc.stored)
			unset := trimmed == "" || trimmed == "null"
			require.Equal(t, !unset && !publicUsable, invalid, "stored=%q 的 invalid 判定", tc.stored)
		})
	}
}

//go:build unit

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

const useKeyConfigValidJSON = `{"version":1,"enabled":true,"platforms":{"openai":{"tabs":["codex"],"codex":{"provider_name":"OpenAI","model":"gpt-5.5-mini","review_model":"","reasoning_effort":""}}}}`

// 「使用密钥」弹窗配置必须原样出现在公开设置与 SSR 注入里，前端直接读取该对象。
func TestSettingService_GetPublicSettings_ExposesUseKeyConfig(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{
		SettingKeyUseKeyConfig: useKeyConfigValidJSON,
	}}, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.JSONEq(t, useKeyConfigValidJSON, string(settings.UseKeyConfig))

	injection, err := svc.GetPublicSettingsForInjection(context.Background())
	require.NoError(t, err)
	payload, err := json.Marshal(injection)
	require.NoError(t, err)
	require.Contains(t, string(payload), `"model":"gpt-5.5-mini"`)
}

// 未配置 / 非法 / 非对象值一律输出合法 JSON null：前端对该字段直接 JSON.parse，
// 空字符串会让解析炸掉，所以必须显式给 null 而不是空值。
func TestSettingService_GetPublicSettings_UseKeyConfigFallsBackToJSONNull(t *testing.T) {
	for _, raw := range []string{
		"",
		"   ",
		"not json",
		"[]",
		`"str"`,
		"null",
		// version 不受支持时同样必须回退：管理端对同一状态返回 null + invalid，两端判据一致。
		`{"version":2,"enabled":true,"platforms":{}}`,
		`{"enabled":true}`,
		`{"version":"1","enabled":true}`,
	} {
		t.Run("raw="+raw, func(t *testing.T) {
			svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{
				SettingKeyUseKeyConfig: raw,
			}}, &config.Config{})

			settings, err := svc.GetPublicSettings(context.Background())
			require.NoError(t, err)
			require.Equal(t, "null", string(settings.UseKeyConfig))
			require.True(t, json.Valid(settings.UseKeyConfig))

			injection, err := svc.GetPublicSettingsForInjection(context.Background())
			require.NoError(t, err)
			payload, err := json.Marshal(injection)
			require.NoError(t, err)
			require.Contains(t, string(payload), `"use_key_config":null`)
		})
	}
}

// 解析失败必须留下可定位的 warn 日志（含键名与原因），不得静默吞掉。
func TestSettingService_GetPublicSettings_UseKeyConfigLogsInvalidJSON(t *testing.T) {
	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	defer slog.SetDefault(previousLogger)

	svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{
		SettingKeyUseKeyConfig: "{not json",
	}}, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, "null", string(settings.UseKeyConfig))

	logged := logs.String()
	require.Contains(t, logged, "level=WARN")
	require.Contains(t, logged, SettingKeyUseKeyConfig)
}

// version 存在但类型不可读（例如字符串 "1"）时同样回退为字面 null，并留下可定位的
// warn 日志。该分支与「版本不受支持」是两条不同出口，只断言输出为 null 覆盖不到它，
// 而 warn 日志本身是 §3.3 要求的必需行为。
func TestSettingService_GetPublicSettings_UseKeyConfigLogsUnreadableVersion(t *testing.T) {
	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	defer slog.SetDefault(previousLogger)

	svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{
		SettingKeyUseKeyConfig: `{"version":"1","enabled":true}`,
	}}, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, "null", string(settings.UseKeyConfig))
	require.True(t, json.Valid(settings.UseKeyConfig))

	logged := logs.String()
	require.Contains(t, logged, "level=WARN")
	require.Contains(t, logged, SettingKeyUseKeyConfig)
	require.Contains(t, logged, "unreadable version")
}

// 键未配置（库中无该行）时同样输出 null，且不产生解析失败日志。
func TestSettingService_GetPublicSettings_UseKeyConfigUnsetIsQuiet(t *testing.T) {
	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	defer slog.SetDefault(previousLogger)

	svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{}}, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, "null", string(settings.UseKeyConfig))
	require.NotContains(t, logs.String(), SettingKeyUseKeyConfig)
}

// version 不受支持时必须回退为字面 null，并留下可定位的 warn 日志：否则
// GET /api/v1/settings/public 会原样返回一个本 build 无法应用的对象，且服务端无日志可查。
func TestSettingService_GetPublicSettings_UseKeyConfigLogsUnsupportedVersion(t *testing.T) {
	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	defer slog.SetDefault(previousLogger)

	svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{
		SettingKeyUseKeyConfig: `{"version":2,"enabled":true,"platforms":{"openai":{"tabs":["codex"]}}}`,
	}}, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, "null", string(settings.UseKeyConfig))
	require.True(t, json.Valid(settings.UseKeyConfig))

	injection, err := svc.GetPublicSettingsForInjection(context.Background())
	require.NoError(t, err)
	payload, err := json.Marshal(injection)
	require.NoError(t, err)
	require.Contains(t, string(payload), `"use_key_config":null`)

	logged := logs.String()
	require.Contains(t, logged, "level=WARN")
	require.Contains(t, logged, SettingKeyUseKeyConfig)
	require.Contains(t, logged, "version=2")
	require.Contains(t, logged, "supported_version=1")
}

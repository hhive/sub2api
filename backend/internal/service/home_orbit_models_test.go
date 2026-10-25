//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestGetPublicSettingsExposesHomeOrbitModels(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyHomeOrbitModels: `[{"vendor":"openai","label":"GPT-6 Astra"},{"vendor":"deepseek","label":"DeepSeek V4.1 Flash"}]`,
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())

	require.NoError(t, err)
	require.JSONEq(t,
		`[{"vendor":"openai","label":"GPT-6 Astra"},{"vendor":"deepseek","label":"DeepSeek V4.1 Flash"}]`,
		settings.HomeOrbitModels)
}

func TestGetPublicSettingsDefaultsHomeOrbitModelsToEmptyArray(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{}}, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())

	require.NoError(t, err)
	require.JSONEq(t, `[]`, settings.HomeOrbitModels)
}

func TestGetPublicSettingsForInjectionCarriesHomeOrbitModels(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyHomeOrbitModels: `[{"vendor":"claude","label":"Claude Opus 5.5"}]`,
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	payload, err := svc.GetPublicSettingsForInjection(context.Background())

	require.NoError(t, err)
	injected, ok := payload.(*PublicSettingsInjectionPayload)
	require.True(t, ok)
	require.JSONEq(t, `[{"vendor":"claude","label":"Claude Opus 5.5"}]`, string(injected.HomeOrbitModels))
}

func TestGetPublicSettingsForInjectionFallsBackToEmptyHomeOrbitModels(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{}}, &config.Config{})

	payload, err := svc.GetPublicSettingsForInjection(context.Background())

	require.NoError(t, err)
	injected, ok := payload.(*PublicSettingsInjectionPayload)
	require.True(t, ok)
	require.JSONEq(t, `[]`, string(injected.HomeOrbitModels))
}

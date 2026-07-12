//go:build unit

package service

import (
	"context"
	"reflect"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestSettingService_GetPublicSettings_ExposesOnyxMenuConfig(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{
		values: map[string]string{
			SettingKeyOnyxEnabled:               "true",
			SettingKeyOnyxBaseURL:               "https://onyx.example.com",
			SettingKeyOnyxMenuLabel:             "Knowledge Chat",
			SettingKeyOnyxLaunchTokenTTLSeconds: "60",
		},
	}, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.OnyxEnabled)
	require.Equal(t, "Knowledge Chat", settings.OnyxMenuLabel)
	require.Equal(t, "/api/v1/onyx/launch", settings.OnyxLaunchPath)
}

func TestSettingService_GetPublicSettings_ExposesLobeHubMenuConfig(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{
		values: map[string]string{
			SettingKeyLobeHubEnabled:   "true",
			SettingKeyLobeHubMenuLabel: "Lobe Chat",
		},
	}, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.LobeHubEnabled)
	require.Equal(t, "Lobe Chat", settings.LobeHubMenuLabel)
	require.Equal(t, "/api/v1/lobehub/launch", settings.LobeHubLaunchPath)
}

func TestSettingService_GetPublicSettings_DefaultsImagePlaygroundToImageAndVideo(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{}}, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, "图片与视频", settings.ImagePlaygroundMenuLabel)
	require.Equal(t, "/api/v1/image-playground/launch", settings.ImagePlaygroundLaunchPath)
}

func TestPublicSettingsInjectionPayloadKeepsPublicSettingsFields(t *testing.T) {
	publicSettingsType := reflect.TypeOf(PublicSettings{})
	injectionPayloadType := reflect.TypeOf(PublicSettingsInjectionPayload{})

	missing := make([]string, 0)
	for i := 0; i < publicSettingsType.NumField(); i++ {
		field := publicSettingsType.Field(i)
		jsonName := field.Tag.Get("json")
		if jsonName == "" {
			jsonName = field.Name
		}
		if _, ok := injectionPayloadType.FieldByName(field.Name); !ok {
			missing = append(missing, jsonName)
		}
	}

	require.Empty(t, missing)
}

//go:build unit

package service

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

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

func TestSettingService_GetPublicSettings_DefaultsMediaPlaygroundToImageAndVideo(t *testing.T) {
	t.Setenv("MEDIA_PLAYGROUND_EXCHANGE_SECRET", "test-secret")
	svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{}}, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.MediaPlaygroundEnabled)
	require.Equal(t, "图片与视频", settings.MediaPlaygroundMenuLabel)
	require.Equal(t, "/api/v1/media-playground/launch", settings.MediaPlaygroundLaunchPath)

	injection, err := svc.GetPublicSettingsForInjection(context.Background())
	require.NoError(t, err)
	payload, err := json.Marshal(injection)
	require.NoError(t, err)
	require.Contains(t, string(payload), `"media_playground_launch_path":"/api/v1/media-playground/launch"`)
	require.False(t, strings.Contains(string(payload), "image_playground_"))
	require.False(t, strings.Contains(string(payload), "video_playground_"))
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

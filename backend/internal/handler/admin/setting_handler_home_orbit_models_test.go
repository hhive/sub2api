//go:build unit

package admin

import (
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/stretchr/testify/require"
)

func TestUpdateSettingsHomeOrbitModelsTrimsAndPersists(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{})

	rec := doUpdateSettings(t, h, map[string]any{
		"home_orbit_models": []map[string]any{
			{"vendor": " openai ", "label": " GPT-6 Astra "},
			{"vendor": "deepseek", "label": "DeepSeek V4.1 Flash"},
		},
	}, nil)

	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t,
		`[{"vendor":"openai","label":"GPT-6 Astra"},{"vendor":"deepseek","label":"DeepSeek V4.1 Flash"}]`,
		repo.values[service.SettingKeyHomeOrbitModels])
}

func TestUpdateSettingsRejectsTooManyHomeOrbitModels(t *testing.T) {
	h, _ := newStepUpSwitchTestHandler(t, map[string]string{})

	models := make([]map[string]any, 25)
	for i := range models {
		models[i] = map[string]any{"vendor": "openai", "label": "GPT-6 Astra"}
	}

	rec := doUpdateSettings(t, h, map[string]any{"home_orbit_models": models}, nil)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateSettingsRejectsInvalidHomeOrbitModels(t *testing.T) {
	cases := map[string]map[string]any{
		"empty vendor":  {"vendor": "  ", "label": "GPT-6 Astra"},
		"empty label":   {"vendor": "openai", "label": "   "},
		"long vendor":   {"vendor": "v-v-v-v-v-v-v-v-v-v-v-v-v-v-v-v-v-v-v-v-v", "label": "GPT-6 Astra"},
		"long label":    {"vendor": "openai", "label": "This label is definitely longer than twenty-eight characters"},
		"missing label": {"vendor": "openai"},
	}

	for name, model := range cases {
		t.Run(name, func(t *testing.T) {
			h, _ := newStepUpSwitchTestHandler(t, map[string]string{})

			rec := doUpdateSettings(t, h, map[string]any{
				"home_orbit_models": []map[string]any{model},
			}, nil)

			require.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

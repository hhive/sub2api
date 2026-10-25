package dto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseHomeOrbitModels(t *testing.T) {
	models := ParseHomeOrbitModels(`[{"vendor":"openai","label":"GPT-6 Astra"},{"vendor":"qwen","label":"Qwen3.5-Max"}]`)

	require.Equal(t, []HomeOrbitModel{
		{Vendor: "openai", Label: "GPT-6 Astra"},
		{Vendor: "qwen", Label: "Qwen3.5-Max"},
	}, models)
}

func TestParseHomeOrbitModelsFallsBackToEmpty(t *testing.T) {
	for _, raw := range []string{"", "  ", "[]", "not-json", `{"vendor":"openai"}`} {
		require.Empty(t, ParseHomeOrbitModels(raw), "raw=%q", raw)
	}
}

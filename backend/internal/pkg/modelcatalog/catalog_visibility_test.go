package modelcatalog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVisibleOpenAIModelsHideRetiredAndPreviewIDs(t *testing.T) {
	models := []string{
		"gpt-4o-audio-preview", "gpt-4o-realtime-preview", "gpt-5.2",
		"gpt-5.2-2025-12-11", "gpt-5.2-chat-latest", "gpt-5.2-pro",
		"gpt-5.3-codex-spark", "gpt-5.4", "gpt-image-1.5",
		"o4-mini", "codex-auto-review", "openai/gpt-5.4", "openai/gpt-5.6", "models/gpt-5.6", "GPT-5.5", "gpt5.2", "GPT 5.2",
		"gpt-5.5", "gpt-5.6", "gpt-5.6-sol", "gpt-6", "gpt-6-astra", "gpt-6-luna", "gpt-image-2",
		"gpt-7.2-sol", "gpt-image-3.1", "my-custom-model",
	}
	require.Equal(t, []string{
		"gpt-5.5", "gpt-5.6-sol", "gpt-6-astra", "gpt-6-luna", "gpt-image-2", "gpt-7.2-sol", "gpt-image-3.1", "my-custom-model",
	}, FilterVisibleOpenAIModels(models))
	require.False(t, IsVisibleOpenAIModel(""))
	require.False(t, IsVisibleOpenAIModel("gpt-5.6"))
	require.False(t, IsVisibleOpenAIModel("gpt-6"))
	require.True(t, IsVisibleOpenAIModel("gpt-5.6-sol"))
	require.True(t, IsVisibleOpenAIModel("gpt-6-astra"))
}

func TestDiscoveredGPTReleasesDoNotRequireStaticCatalogUpdate(t *testing.T) {
	for _, model := range []string{"gpt-6.1-sol", "gpt-6.2-sol", "gpt-7-sol", "gpt-7.2-codex", "gpt-10-sol", "gpt-image-3", "gpt-image-3.1-flash"} {
		require.True(t, IsVisibleOpenAIModel(model), model)
	}
	for _, model := range []string{"gpt-6", "gpt-6.x-sol", "gpt-6.1/sol", "gpt-6.1-*", "gpt-4o", "gpt-image-1.5", "codex-auto-review"} {
		require.False(t, IsVisibleOpenAIModel(model), model)
	}
	require.NotContains(t, ModelsForPlatform("openai"), "gpt-7-sol", "discovery must not invent static models")
}

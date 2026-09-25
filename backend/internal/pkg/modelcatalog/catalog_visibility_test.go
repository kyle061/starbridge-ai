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
		"gpt-5.5", "gpt-5.6-sol", "gpt-6-luna", "gpt-image-2",
		"my-custom-model",
	}
	require.Equal(t, []string{
		"gpt-5.5", "gpt-5.6-sol", "gpt-6-luna", "gpt-image-2", "my-custom-model",
	}, FilterVisibleOpenAIModels(models))
	require.False(t, IsVisibleOpenAIModel(""))
}

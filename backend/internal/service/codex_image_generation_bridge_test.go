package service

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCodexImageBridgeLiteKeepsProtocolAndExistingInstructions(t *testing.T) {
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	tools := []any{map[string]any{"type": "custom", "name": "exec"}}
	body := map[string]any{"model": "gpt-6-astra", "instructions": "Preserve the user's requested design.", "tools": tools, "parallel_tool_calls": false, "reasoning": map[string]any{"context": "all_turns"}}
	require.True(t, ensureCodexImageGenerationBridge(body, account, true))
	require.Equal(t, tools, body["tools"])
	require.False(t, hasOpenAIImageGenerationTool(body))
	require.NotContains(t, body, "tool_choice")
	require.Equal(t, false, body["parallel_tool_calls"])
	require.Contains(t, body["instructions"], "Preserve the user's requested design.")
	require.Contains(t, body["instructions"], codexImageAPIAvailableMarker)
	require.False(t, ensureCodexImageGenerationBridge(body, account, true), "guidance must be appended only once")
}

func TestCodexImageBridgeLitePreservesClientImageTool(t *testing.T) {
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	body := map[string]any{"model": "gpt-6-astra", "instructions": "Use the client's image tool.", "tools": []any{map[string]any{"type": "function", "name": "image_gen.imagegen"}}}
	require.False(t, ensureCodexImageGenerationBridge(body, account, true))
	require.Equal(t, "Use the client's image tool.", body["instructions"])
}

func TestCodexImageBridgeLiteUsesGroupRouteWithTextOnlyAccount(t *testing.T) {
	account := &Account{
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"model_mapping": map[string]any{"gpt-5.4": "gpt-5.4"}},
	}
	require.Empty(t, codexImageGenerationModel(account))
	body := map[string]any{"model": "gpt-5.4", "input": "Generate a landscape image"}
	require.True(t, ensureCodexImageGenerationBridge(body, account, true))
	require.Contains(t, body["instructions"], codexImageAPIAvailableMarker)
	require.False(t, hasOpenAIImageGenerationTool(body))
	require.False(t, ensureCodexImageGenerationBridge(body, account, false), "text account cannot offer the hosted tool")
}

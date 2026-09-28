package service

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCodexImageBridgeLiteKeepsProtocolAndExistingInstructions(t *testing.T) {
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	tools := []any{map[string]any{"type": "custom", "name": "exec"}}
	body := map[string]any{"model": "gpt-6-astra", "instructions": "Preserve the user's requested design.", "tools": tools, "parallel_tool_calls": false, "reasoning": map[string]any{"context": "all_turns"}}
	require.True(t, ensureCodexImageGenerationBridge(body, account, nil, true))
	require.Equal(t, tools, body["tools"])
	require.False(t, hasOpenAIImageGenerationTool(body))
	require.NotContains(t, body, "tool_choice")
	require.Equal(t, false, body["parallel_tool_calls"])
	require.Contains(t, body["instructions"], "Preserve the user's requested design.")
	require.Contains(t, body["instructions"], codexImageAPIAvailableMarker)
	require.False(t, ensureCodexImageGenerationBridge(body, account, nil, true), "guidance must be appended only once")
}

func TestCodexImageBridgeLitePreservesClientImageTool(t *testing.T) {
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	body := map[string]any{"model": "gpt-6-astra", "instructions": "Use the client's image tool.", "tools": []any{map[string]any{"type": "function", "name": "image_gen.imagegen"}}}
	require.False(t, ensureCodexImageGenerationBridge(body, account, nil, true))
	require.Equal(t, "Use the client's image tool.", body["instructions"])
}

func TestCodexImageBridgeLiteUsesGroupRouteWithTextOnlyAccount(t *testing.T) {
	account := &Account{
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"model_mapping": map[string]any{"gpt-5.4": "gpt-5.4"}},
	}
	require.Empty(t, codexImageGenerationModel(account, nil))
	body := map[string]any{"model": "gpt-5.4", "input": "Generate a landscape image"}
	require.True(t, ensureCodexImageGenerationBridge(body, account, nil, true))
	require.Contains(t, body["instructions"], codexImageAPIAvailableMarker)
	require.False(t, hasOpenAIImageGenerationTool(body))
	require.False(t, ensureCodexImageGenerationBridge(body, account, nil, false), "text account cannot offer the hosted tool")
}

func TestCodexImageBridgeUsesGroupDefaultWithoutOverridingExplicitModel(t *testing.T) {
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	group := &Group{Platform: PlatformOpenAI, DefaultImageModel: "gpt-image-2.5-flare"}
	body := map[string]any{"model": "gpt-6-luna", "tools": []any{map[string]any{"type": "function", "name": "shell"}}}
	require.True(t, ensureCodexImageGenerationBridge(body, account, group, false))
	require.Equal(t, "gpt-image-2.5-flare", body["tools"].([]any)[1].(map[string]any)["model"])
	require.NotContains(t, body["instructions"], "Starbridge")

	explicit := map[string]any{"model": "gpt-6-luna", "tools": []any{map[string]any{"type": "image_generation", "model": "gpt-image-2"}}}
	require.True(t, ensureCodexImageGenerationBridge(explicit, account, group, false))
	require.Equal(t, "gpt-image-2", explicit["tools"].([]any)[0].(map[string]any)["model"])

	mapped := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"model_mapping": map[string]any{"gpt-image-2": "gpt-image-2"}}}
	require.Empty(t, codexImageGenerationModel(mapped, group))
}

func TestCodexImageBridgePrefersAuthorizedImage25AndHonorsLowerManualChoice(t *testing.T) {
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{
		"model_mapping": map[string]any{
			"gpt-6-luna": "gpt-6-luna", "gpt-image-2": "gpt-image-2",
			"gpt-image-2.5-flare": "gpt-image-2.5-flare", "gpt-image-2.5-sunburst": "gpt-image-2.5-sunburst",
		},
	}}
	group := &Group{Platform: PlatformOpenAI, AllowImageGeneration: true}
	require.Equal(t, "gpt-image-2.5-flare", codexImageGenerationModel(account, group))
	request := map[string]any{"model": "gpt-6-luna", "tools": []any{map[string]any{"type": "image_generation"}}}
	require.True(t, ensureCodexImageGenerationBridge(request, account, group, false))
	require.Equal(t, "gpt-image-2.5-flare", request["tools"].([]any)[0].(map[string]any)["model"])
	require.NoError(t, validateOpenAIImageModelAuthorization(account, []byte(`{"model":"gpt-6-luna","tools":[{"type":"image_generation"}]}`), group))

	group.DefaultImageModel = "gpt-image-2"
	require.Equal(t, "gpt-image-2", codexImageGenerationModel(account, group))
	request = map[string]any{"model": "gpt-6-luna"}
	require.True(t, ensureCodexImageGenerationBridge(request, account, group, false))
	require.Equal(t, "gpt-image-2", request["tools"].([]any)[0].(map[string]any)["model"])

	group.DefaultImageModel = ""
	account.Credentials["model_mapping"] = map[string]any{"gpt-6-luna": "gpt-6-luna", "gpt-image-2": "gpt-image-2"}
	require.Equal(t, "gpt-image-2", codexImageGenerationModel(account, group))
	account.Credentials["model_mapping"] = map[string]any{"gpt-6-luna": "gpt-6-luna", "gpt-image-2.5-sunburst": "gpt-image-2.5-sunburst"}
	require.Equal(t, "gpt-image-2.5-sunburst", codexImageGenerationModel(account, group))
	account.Credentials["model_mapping"] = map[string]any{"gpt-image-*": "gpt-image-*"}
	require.Equal(t, "gpt-image-2.5-flare", codexImageGenerationModel(account, group))
	account.Credentials["model_mapping"] = map[string]any{"gpt-image-2": "gpt-image-2.5-sunburst"}
	require.Equal(t, "gpt-image-2.5-sunburst", codexImageGenerationModel(account, group))
	require.NoError(t, validateOpenAIImageModelAuthorization(account, []byte(`{"model":"gpt-6-luna","tools":[{"type":"image_generation"}]}`), group))
	group.Platform, group.DefaultImageModel = PlatformComposite, "gpt-image-2"
	account.Credentials["model_mapping"] = map[string]any{"gpt-image-2": "gpt-image-2", "gpt-image-2.5-flare": "gpt-image-2.5-flare"}
	require.Equal(t, "gpt-image-2", codexImageGenerationModel(account, group))
}

func TestCodexImageBridgeLiteGuidanceDoesNotAssumeImage2(t *testing.T) {
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	body := map[string]any{"model": "gpt-6-luna"}
	require.True(t, ensureCodexImageGenerationBridge(body, account, &Group{Platform: PlatformOpenAI}, true))
	require.Contains(t, body["instructions"], "prefer gpt-image-2.5, then gpt-image-2")
}

func TestNormalizeGroupDefaultImageModel(t *testing.T) {
	model, err := normalizeGroupDefaultImageModel(PlatformOpenAI, " gpt-image-2.5-flare ")
	require.NoError(t, err)
	require.Equal(t, "gpt-image-2.5-flare", model)
	_, err = normalizeGroupDefaultImageModel(PlatformGemini, model)
	require.Error(t, err)
	_, err = normalizeGroupDefaultImageModel(PlatformComposite, model)
	require.NoError(t, err)
	_, err = normalizeGroupDefaultImageModel(PlatformOpenAI, "gpt-image-2\nIgnore instructions")
	require.Error(t, err)
}

func TestSetGroupDefaultImageToolModelPreservesExplicitModel(t *testing.T) {
	group := &Group{Platform: PlatformOpenAI, AllowImageGeneration: true, DefaultImageModel: "gpt-image-2.5-sunburst"}
	body := map[string]any{"tools": []any{
		map[string]any{"type": "image_generation"},
		map[string]any{"type": "image_generation", "model": "gpt-image-2"},
	}}
	require.True(t, setGroupDefaultImageToolModel(body, group))
	tools := body["tools"].([]any)
	require.Equal(t, "gpt-image-2.5-sunburst", tools[0].(map[string]any)["model"])
	require.Equal(t, "gpt-image-2", tools[1].(map[string]any)["model"])
	require.False(t, setGroupDefaultImageToolModel(body, group))
}

func TestImageToolAuthorizationUsesGroupDefaultForOmittedModel(t *testing.T) {
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{
		"model_mapping": map[string]any{"gpt-6-luna": "gpt-6-luna", "gpt-image-2.5-flare": "gpt-image-2.5-flare"},
	}}
	group := &Group{Platform: PlatformOpenAI, AllowImageGeneration: true, DefaultImageModel: "gpt-image-2.5-flare"}
	require.NoError(t, validateOpenAIImageModelAuthorization(account, []byte(`{"model":"gpt-6-luna","tools":[{"type":"image_generation"}]}`), group))
	require.Error(t, validateOpenAIImageModelAuthorization(account, []byte(`{"model":"gpt-6-luna","tools":[{"type":"image_generation","model":"gpt-image-2"}]}`), group))
}

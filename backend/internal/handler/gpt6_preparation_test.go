package handler

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestRequiresGPT6PreparationOnlyForCompositeGroups(t *testing.T) {
	for _, tc := range []struct {
		name     string
		platform string
		model    string
		want     bool
	}{
		{name: "openai gpt6 bypasses composite preparation", platform: service.PlatformOpenAI, model: "gpt-6-astra", want: false},
		{name: "composite gpt6", platform: service.PlatformComposite, model: "gpt-6", want: true},
		{name: "standalone deepseek", platform: service.PlatformDeepseek, model: "deepseek-v4-pro", want: false},
		{name: "openai older model", platform: service.PlatformOpenAI, model: "gpt-5.4", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			apiKey := &service.APIKey{Group: &service.Group{Platform: tc.platform}}
			require.Equal(t, tc.want, requiresGPT6Preparation(apiKey, tc.model))
		})
	}
}

func TestGPT6PreparationBillingMultiplierFollowsAdminPricing(t *testing.T) {
	cfg := &config.Config{}
	cfg.Billing.GPT6Preparation.PreparationMultiplier = 12
	cfg.Billing.RetailPricing.Enabled = true
	cfg.Billing.RetailPricing.LatestMultiplier = 6
	require.Equal(t, 6.0, gpt6PreparationBillingMultiplier(cfg))

	cfg.Billing.RetailPricing.LatestMultiplier = 8
	require.Equal(t, 8.0, gpt6PreparationBillingMultiplier(cfg))

	cfg.Billing.RetailPricing.Enabled = false
	require.Equal(t, 12.0, gpt6PreparationBillingMultiplier(cfg))
}

func TestGPT6PreparationFailureKeepsOriginalRequest(t *testing.T) {
	original := []byte(`{"model":"gpt-6-astra","input":"keep this request unchanged"}`)
	prepared := []byte(`{"model":"gpt-6-astra","input":"rewritten"}`)

	require.Equal(t, original, gpt6PreparedBodyOrOriginal(original, prepared, errors.New("no preparation account")))
	require.Equal(t, prepared, gpt6PreparedBodyOrOriginal(original, prepared, nil))
}

func TestBuildGPT6PreparationBodyKeepsTaskButDisablesTools(t *testing.T) {
	body := []byte(`{"model":"gpt-6","input":"ship it","tools":[{"type":"function"}],"stream":true}`)
	prepared, err := buildGPT6PreparationBody(body, "deepseek-v4-pro", 1200, true)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(prepared, &payload))
	require.Equal(t, "deepseek-v4-pro", payload["model"])
	require.Equal(t, false, payload["stream"])
	require.Nil(t, payload["tools"])
	require.Equal(t, "ship it", payload["input"])

	finalBody := appendGPT6Requirements(body, "objective: ship it")
	require.Contains(t, string(finalBody), "objective: ship it")
}

func TestBuildGPT6PreparationBodyPreservesResponseInstructions(t *testing.T) {
	prepared, err := buildGPT6PreparationBody([]byte(`{"model":"gpt-6","instructions":"use the existing API","input":"ship it"}`), "deepseek-v4-pro", 1200, true)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(prepared, &payload))
	require.Contains(t, payload["instructions"], gpt6PreparationInstruction)
	require.Contains(t, payload["instructions"], "use the existing API")
	require.Equal(t, "ship it", payload["input"])
}

func TestBuildGPT6PreparationChatBodyPrependsAnalystInstruction(t *testing.T) {
	prepared, err := buildGPT6PreparationBody([]byte(`{"model":"gpt-6","messages":[{"role":"user","content":"hello"}]}`), "gpt-5.4-mini", 10, false)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(prepared, &payload))
	messages := payload["messages"].([]any)
	require.Len(t, messages, 2)
	require.Equal(t, "system", messages[0].(map[string]any)["role"])
	require.Equal(t, "hello", messages[1].(map[string]any)["content"])
}

func TestResponsesPreparationBodyToChatPreservesPreparedInput(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","instructions":"analyze","input":"ship it","stream":false,"max_output_tokens":1200}`)
	converted, err := responsesPreparationBodyToChat(body)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(converted, &payload))
	require.Equal(t, "deepseek-v4-pro", payload["model"])
	require.NotEqual(t, true, payload["stream"])
	require.Equal(t, float64(1200), payload["max_completion_tokens"])
	require.Nil(t, payload["max_output_tokens"])
	messages := payload["messages"].([]any)
	require.Len(t, messages, 2)
	require.Equal(t, "system", messages[0].(map[string]any)["role"])
	require.Equal(t, "analyze", messages[0].(map[string]any)["content"])
	require.Equal(t, "user", messages[1].(map[string]any)["role"])
	require.Equal(t, "ship it", messages[1].(map[string]any)["content"])
}

func TestGPT6PreparationRequestIDAlwaysUsesPrivatePrefix(t *testing.T) {
	require.Equal(t, "gpt6-prep:upstream-1", gpt6PreparationRequestID("upstream-1", nil, "gpt-6", "deepseek-v4-pro"))
	require.Equal(t, "gpt6-prep:already-private", gpt6PreparationRequestID("gpt6-prep:already-private", nil, "gpt-6", "deepseek-v4-pro"))
	require.NotEmpty(t, gpt6PreparationRequestID("", []byte("request"), "gpt-6", "deepseek-v4-pro"))
}

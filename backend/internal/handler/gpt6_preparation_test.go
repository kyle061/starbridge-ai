package handler

import (
	"context"
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
	body := []byte(`{"model":"gpt-6","input":"ship it","tools":[{"type":"function"}],"reasoning":{"effort":"high"},"service_tier":"priority","prompt_cache_key":"private","previous_response_id":"resp_1","stream":true}`)
	prepared, err := buildGPT6PreparationBody(body, "deepseek-v4-pro", 1200, true)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(prepared, &payload))
	require.Equal(t, "deepseek-v4-pro", payload["model"])
	require.Equal(t, false, payload["stream"])
	require.Nil(t, payload["tools"])
	require.Nil(t, payload["reasoning"])
	require.Nil(t, payload["service_tier"])
	require.Nil(t, payload["prompt_cache_key"])
	require.Nil(t, payload["previous_response_id"])
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

func TestBuildGPT6PreparationBodyRemovesEmbeddedToolDeclarations(t *testing.T) {
	body := []byte(`{"model":"gpt-6","input":[{"role":"user","content":"ship it"},{"type":"additional_tools","tools":[{"type":"function","name":"exec"}]},{"type":"tool_search_output","call_id":"search_1","tools":[{"type":"function","name":"inspect"}]},{"type":"tool_search_output","call_id":"search_2","output":"found docs","tools":[{"type":"function","name":"read"}]}]}`)
	prepared, err := buildGPT6PreparationBody(body, "deepseek-v4-pro", 1200, true)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(prepared, &payload))
	items := payload["input"].([]any)
	require.Len(t, items, 2)
	require.Equal(t, "ship it", items[0].(map[string]any)["content"])
	require.Equal(t, "found docs", items[1].(map[string]any)["output"])
	require.Nil(t, items[1].(map[string]any)["tools"])

	converted, err := responsesPreparationBodyToChat(prepared)
	require.NoError(t, err)
	require.NotContains(t, string(converted), `"tools"`)
}

func TestBuildGPT6PreparationChatBodyPrependsAnalystInstruction(t *testing.T) {
	prepared, err := buildGPT6PreparationBody([]byte(`{"model":"gpt-6","messages":[{"role":"user","content":"hello"}],"tools":[{"type":"function"}],"reasoning_effort":"high","service_tier":"priority"}`), "gpt-5.4-mini", 10, false)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(prepared, &payload))
	messages := payload["messages"].([]any)
	require.Len(t, messages, 2)
	require.Equal(t, "system", messages[0].(map[string]any)["role"])
	require.Equal(t, "hello", messages[1].(map[string]any)["content"])
	require.Nil(t, payload["tools"])
	require.Nil(t, payload["reasoning_effort"])
	require.Nil(t, payload["service_tier"])
}

func TestResponsesPreparationBodyToChatPreservesPreparedInput(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","instructions":"analyze","input":"ship it","stream":false,"max_output_tokens":1200}`)
	converted, err := responsesPreparationBodyToChat(body)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(converted, &payload))
	require.Equal(t, "deepseek-v4-pro", payload["model"])
	require.NotEqual(t, true, payload["stream"])
	require.Equal(t, float64(1200), payload["max_tokens"])
	require.Nil(t, payload["max_completion_tokens"])
	require.Nil(t, payload["max_output_tokens"])
	messages := payload["messages"].([]any)
	require.Len(t, messages, 2)
	require.Equal(t, "system", messages[0].(map[string]any)["role"])
	require.Equal(t, "analyze", messages[0].(map[string]any)["content"])
	require.Equal(t, "user", messages[1].(map[string]any)["role"])
	require.Equal(t, "ship it", messages[1].(map[string]any)["content"])
}

func TestGPT6PreparationFallsBackAfterCandidateFailure(t *testing.T) {
	candidates := gpt6PreparationCandidates("deepseek-v4-pro", true)
	var attempted []string
	result, err := firstSuccessfulGPT6Preparation(context.Background(), candidates, func(candidate gpt6PreparationCandidate) (*gpt6PreparationResult, error) {
		attempted = append(attempted, candidate.Model)
		if candidate.Model != "gpt-5.4-nano" {
			return nil, errors.New("no account available")
		}
		return &gpt6PreparationResult{Model: candidate.Model, Document: "requirements"}, nil
	})
	require.NoError(t, err)
	require.Equal(t, []string{"deepseek-v4-pro", "gpt-5.6-luna", "gpt-5.4-nano"}, attempted)
	require.Equal(t, "gpt-5.4-nano", result.Model)
}

func TestGPT6PreparationCandidatesIncludeAffordableOpenAIFallbacks(t *testing.T) {
	candidates := gpt6PreparationCandidates("deepseek-v4-pro", true)
	models := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		models = append(models, candidate.Model)
	}
	require.Equal(t, []string{
		"deepseek-v4-pro", "gpt-5.6-luna", "gpt-5.4-nano", "gpt-5.4-mini",
		"gpt-5-mini", "gpt-5.2", "gpt-5.4",
	}, models)
	for _, candidate := range candidates[1:] {
		require.Equal(t, service.PlatformOpenAI, candidate.Platform)
		require.Equal(t, service.OpenAIEndpointCapabilityResponses, candidate.Capability)
	}
}

func TestGPT6PreparationStopsFallbackAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var attempts int
	_, err := firstSuccessfulGPT6Preparation(ctx, gpt6PreparationCandidates("deepseek-v4-pro", true), func(gpt6PreparationCandidate) (*gpt6PreparationResult, error) {
		attempts++
		cancel()
		return nil, context.Canceled
	})
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, attempts)
}

func TestGPT6PreparationRequestIDAlwaysUsesPrivatePrefix(t *testing.T) {
	require.Equal(t, "gpt6-prep:upstream-1", gpt6PreparationRequestID("upstream-1", nil, "gpt-6", "deepseek-v4-pro"))
	require.Equal(t, "gpt6-prep:already-private", gpt6PreparationRequestID("gpt6-prep:already-private", nil, "gpt-6", "deepseek-v4-pro"))
	require.NotEmpty(t, gpt6PreparationRequestID("", []byte("request"), "gpt-6", "deepseek-v4-pro"))
}

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
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

func TestGPT6PreparationSkipsResponsesWithoutInput(t *testing.T) {
	cfg := &config.Config{}
	cfg.Billing.GPT6Preparation.Enabled = true
	h := &OpenAIGatewayHandler{cfg: cfg, gatewayService: &service.OpenAIGatewayService{}}
	apiKey := &service.APIKey{Group: &service.Group{Platform: service.PlatformComposite}}
	for _, body := range [][]byte{
		[]byte(`{"model":"gpt-6-luna"}`),
		[]byte(`{"model":"gpt-6-luna","input":null}`),
		[]byte(`{"model":"gpt-6-luna","input":"  "}`),
		[]byte(`{"model":"gpt-6-luna","input":[]}`),
		[]byte(`{"model":"gpt-6-luna","input":[{"type":"additional_tools","tools":[]}]}`),
		[]byte(`{"model":"gpt-6-luna","input":[{"type":"function_call_output","output":"tool result"}]}`),
	} {
		prepared, err := h.prepareGPT6Request(nil, apiKey, "gpt-6-luna", body, true)
		require.NoError(t, err)
		require.Equal(t, body, prepared)
	}
	require.True(t, hasGPT6PreparationInput([]byte(`{"input":"ship it"}`)))
	require.True(t, hasGPT6PreparationInput([]byte(`{"input":[{"role":"user","content":"ship it"}]}`)))
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

func TestBuildGPT6PreparationBodyBoundsOriginalInstructions(t *testing.T) {
	body, err := json.Marshal(map[string]any{
		"model": "gpt-6", "input": "ship it", "instructions": strings.Repeat("private context ", 2000),
	})
	require.NoError(t, err)
	prepared, err := buildGPT6PreparationBody(body, "gpt-5.6-luna", 1200, true)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(prepared, &payload))
	require.LessOrEqual(t, len([]rune(payload["instructions"].(string))), len([]rune(gpt6PreparationInstruction))+gpt6PreparationMaxInstrRunes+100)
}

func TestBuildGPT6PreparationBodyRemovesEmbeddedToolDeclarations(t *testing.T) {
	body := []byte(`{"model":"gpt-6","input":[{"role":"user","content":"ship it"},{"type":"additional_tools","tools":[{"type":"function","name":"exec"}]},{"type":"tool_search_output","call_id":"search_1","tools":[{"type":"function","name":"inspect"}]},{"type":"tool_search_output","call_id":"search_2","output":"found docs","tools":[{"type":"function","name":"read"}]}]}`)
	prepared, err := buildGPT6PreparationBody(body, "deepseek-v4-pro", 1200, true)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(prepared, &payload))
	require.Equal(t, "ship it", payload["input"])

	converted, err := responsesPreparationBodyToChat(prepared)
	require.NoError(t, err)
	require.NotContains(t, string(converted), `"tools"`)
}

func TestGPT6PreparationUsesLatestUserTextWithBoundedContext(t *testing.T) {
	input := []any{
		map[string]any{"role": "user", "content": "old request"},
		map[string]any{"type": "function_call_output", "output": strings.Repeat("tool data", 1000)},
		map[string]any{"role": "user", "content": []any{
			map[string]any{"type": "input_text", "text": "fix the billing bug"},
			map[string]any{"type": "input_image", "image_url": "data:image/png;base64,private"},
		}},
	}
	require.Equal(t, "fix the billing bug", gpt6PreparationTaskInput(input))
	longText := strings.Repeat("a", 5000) + strings.Repeat("b", 5000)
	limited := gpt6PreparationTaskInput(longText)
	require.Contains(t, limited, "[earlier content omitted]")
	require.True(t, strings.HasPrefix(limited, strings.Repeat("a", 4000)))
	require.True(t, strings.HasSuffix(limited, strings.Repeat("b", 4000)))
	require.Less(t, len(limited), len(longText))
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
	for _, candidate := range candidates {
		require.False(t, service.IsGPT6Model(candidate.Model), "preparation must never use a GPT-6 model")
	}
}

func TestGPT6PreparationSkipsConfiguredGPT6ModelAndUsesGPT5Fallback(t *testing.T) {
	candidates := gpt6PreparationCandidates("gpt-6-luna", true)
	require.NotEmpty(t, candidates)
	require.Equal(t, "gpt-5.6-luna", candidates[0].Model)
	for _, candidate := range candidates {
		require.False(t, service.IsGPT6Model(candidate.Model))
	}
}

func TestGPT6PreparationFailureLogDoesNotReportFinalModelDowngrade(t *testing.T) {
	core, observed := observer.New(zapcore.InfoLevel)
	logGPT6PreparationFallback(zap.New(core), "openai.gpt6_preparation_failed", "gpt-6-luna", 0, errors.New("no preparation account"))
	require.Len(t, observed.All(), 1)
	require.Equal(t, zapcore.InfoLevel, observed.All()[0].Level)
	fields := observed.All()[0].ContextMap()
	require.Equal(t, "continue_original_request", fields["preparation_fallback"])
	require.Equal(t, false, fields["formal_model_downgraded"])
	require.Equal(t, "gpt-6-luna", fields["requested_model"])
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

func TestGPT6PreparationStopsFallbackAtDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	var attempts int
	_, err := firstSuccessfulGPT6Preparation(ctx, gpt6PreparationCandidates("deepseek-v4-pro", true), func(gpt6PreparationCandidate) (*gpt6PreparationResult, error) {
		attempts++
		<-ctx.Done()
		return nil, ctx.Err()
	})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, 1, attempts)
}

func TestGPT6PreparationRequestIDAlwaysUsesPrivatePrefix(t *testing.T) {
	require.Equal(t, "gpt6-prep:upstream-1", gpt6PreparationRequestID("upstream-1", nil, "gpt-6", "deepseek-v4-pro"))
	require.Equal(t, "gpt6-prep:already-private", gpt6PreparationRequestID("gpt6-prep:already-private", nil, "gpt-6", "deepseek-v4-pro"))
	require.NotEmpty(t, gpt6PreparationRequestID("", []byte("request"), "gpt-6", "deepseek-v4-pro"))
}

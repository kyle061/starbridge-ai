package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const (
	gpt6PreparationInstruction      = "You are an internal requirements analyst. Read the user's request and produce a concise, implementation-ready requirements document for the final GPT6 executor. Include objective, constraints, acceptance criteria, edge cases, and open assumptions. Do not execute tools or claim that the task is complete. Return only the document."
	gpt6PreparationMaxDocumentBytes = 16 << 10
)

func requiresGPT6Preparation(apiKey *service.APIKey, model string) bool {
	if apiKey == nil || apiKey.Group == nil {
		return false
	}
	// Only the composite route has a DeepSeek preparation stage. OpenAI groups
	// should send GPT6 requests directly to their selected OpenAI account.
	if apiKey.Group.Platform != service.PlatformComposite {
		return false
	}
	return service.IsGPT6Model(model)
}

func gpt6PreparedBodyOrOriginal(original, prepared []byte, preparationErr error) []byte {
	if preparationErr != nil || prepared == nil {
		return original
	}
	return prepared
}

func gpt6PreparationBillingMultiplier(cfg *config.Config) float64 {
	if cfg.Billing.RetailPricing.Enabled {
		return cfg.Billing.RetailPricing.LatestMultiplier
	}
	return cfg.Billing.GPT6Preparation.PreparationMultiplier
}

func (h *OpenAIGatewayHandler) prepareGPT6Request(c *gin.Context, apiKey *service.APIKey, model string, body []byte, responses bool) ([]byte, error) {
	if h == nil || h.gatewayService == nil || h.cfg == nil || !h.cfg.Billing.GPT6Preparation.Enabled || !requiresGPT6Preparation(apiKey, model) {
		return body, nil
	}
	prep := h.cfg.Billing.GPT6Preparation
	prepModel := strings.TrimSpace(prep.Model)
	if prepModel == "" {
		return nil, errors.New("GPT6 preparation model is not configured")
	}
	maxOutput := prep.MaxOutputTokens
	if maxOutput <= 0 {
		maxOutput = 1200
	}

	attempt, err := firstSuccessfulGPT6Preparation(c.Request.Context(), gpt6PreparationCandidates(prepModel, responses), func(candidate gpt6PreparationCandidate) (*gpt6PreparationResult, error) {
		return h.runGPT6PreparationCandidate(c, apiKey, body, maxOutput, responses, candidate)
	})
	if err != nil {
		return nil, fmt.Errorf("GPT6 requirements request failed: %w", err)
	}
	document := attempt.Document
	if len(document) > gpt6PreparationMaxDocumentBytes {
		document = document[:gpt6PreparationMaxDocumentBytes]
	}

	// The preparation row keeps the public alias while its BillingModel and
	// UpstreamModel identify the real cheap model for admin accounting.
	result := attempt.Result
	actualPrepModel := strings.TrimSpace(result.UpstreamModel)
	if actualPrepModel == "" {
		actualPrepModel = attempt.Model
	}
	result.Model = model
	result.BillingModel = actualPrepModel
	result.UpstreamModel = actualPrepModel
	result.RequestID = gpt6PreparationRequestID(result.RequestID, body, model, actualPrepModel)
	h.submitGPT6PreparationUsage(c, apiKey, attempt.Account, result, model, gpt6PreparationBillingMultiplier(h.cfg), body)

	return appendGPT6Requirements(body, document), nil
}

type gpt6PreparationCandidate struct {
	Model              string
	Platform           string
	Capability         service.OpenAIEndpointCapability
	UseChatCompletions bool
}

type gpt6PreparationResult struct {
	Account  *service.Account
	Result   *service.OpenAIForwardResult
	Model    string
	Document string
}

func gpt6PreparationCandidates(prepModel string, responses bool) []gpt6PreparationCandidate {
	candidates := []gpt6PreparationCandidate{
		// DeepSeek accounts are OpenAI-compatible; selecting Chat Completions
		// keeps fixed-chat accounts eligible while their account protocol still
		// controls whether the actual upstream request is native Responses.
		{Model: prepModel, Platform: service.PlatformDeepseek, Capability: service.OpenAIEndpointCapabilityChatCompletions, UseChatCompletions: true},
		// Composite groups may not have DeepSeek accounts, and model-level
		// cooldowns can temporarily remove mini models. Try the other supported
		// low-cost GPT models before falling back to standard-priced models.
		{Model: "gpt-5.6-luna", Platform: service.PlatformOpenAI, Capability: service.OpenAIEndpointCapabilityResponses},
		{Model: "gpt-5.4-nano", Platform: service.PlatformOpenAI, Capability: service.OpenAIEndpointCapabilityResponses},
		{Model: "gpt-5.4-mini", Platform: service.PlatformOpenAI, Capability: service.OpenAIEndpointCapabilityResponses},
		{Model: "gpt-5-mini", Platform: service.PlatformOpenAI, Capability: service.OpenAIEndpointCapabilityResponses},
		{Model: "gpt-5.2", Platform: service.PlatformOpenAI, Capability: service.OpenAIEndpointCapabilityResponses},
		{Model: "gpt-5.4", Platform: service.PlatformOpenAI, Capability: service.OpenAIEndpointCapabilityResponses},
	}
	if !responses {
		for i := 1; i < len(candidates); i++ {
			candidates[i].Capability = service.OpenAIEndpointCapabilityChatCompletions
		}
	}
	return candidates
}

func firstSuccessfulGPT6Preparation(ctx context.Context, candidates []gpt6PreparationCandidate, attempt func(gpt6PreparationCandidate) (*gpt6PreparationResult, error)) (*gpt6PreparationResult, error) {
	var failures []error
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		result, err := attempt(candidate)
		if err == nil && result != nil {
			return result, nil
		}
		if err == nil {
			err = errors.New("preparation returned no result")
		}
		failures = append(failures, fmt.Errorf("%s: %w", candidate.Model, err))
	}
	if len(failures) == 0 {
		return nil, errors.New("no GPT6 preparation candidates")
	}
	return nil, errors.Join(failures...)
}

func (h *OpenAIGatewayHandler) runGPT6PreparationCandidate(c *gin.Context, apiKey *service.APIKey, body []byte, maxOutput int, responses bool, candidate gpt6PreparationCandidate) (*gpt6PreparationResult, error) {
	if apiKey == nil || apiKey.GroupID == nil {
		return nil, errors.New("GPT6 preparation requires a group")
	}
	prepBody, err := buildGPT6PreparationBody(body, candidate.Model, maxOutput, responses)
	if err != nil {
		return nil, fmt.Errorf("build requirements request: %w", err)
	}
	if candidate.UseChatCompletions && responses {
		prepBody, err = responsesPreparationBodyToChat(prepBody)
		if err != nil {
			return nil, fmt.Errorf("convert requirements request to chat completions: %w", err)
		}
	}
	// Suppress composite execution candidates; this pass selects the candidate platform.
	prepCtx := service.WithoutCompositeRouteCandidates(c.Request.Context())
	selection, _, err := h.gatewayService.SelectAccountWithSchedulerForCapability(
		prepCtx, apiKey.GroupID, "", "", candidate.Model, nil,
		service.OpenAIUpstreamTransportHTTPSSE, candidate.Capability, false, false, true, candidate.Platform,
	)
	if err != nil {
		return nil, fmt.Errorf("select preparation account: %w", err)
	}
	if selection == nil || selection.Account == nil {
		return nil, errors.New("no eligible preparation account")
	}
	if selection.ReleaseFunc != nil {
		defer selection.ReleaseFunc()
	}

	// Forward mutates the Gin context and writes the response, so each attempt
	// needs its own recorder and must never receive the client writer.
	recorder := httptest.NewRecorder()
	prepContext, _ := gin.CreateTestContext(recorder)
	prepContext.Request = c.Request.Clone(c.Request.Context())
	prepContext.Request.Body = io.NopCloser(bytes.NewReader(prepBody))
	prepContext.Request.ContentLength = int64(len(prepBody))
	prepContext.Request.Header = c.Request.Header.Clone()
	for key, value := range c.Keys {
		prepContext.Set(key, value)
	}
	service.SetOpenAIClientTransport(prepContext, service.OpenAIClientTransportHTTP)
	prepContext.Set("gpt6_preparation", true)

	var result *service.OpenAIForwardResult
	if candidate.UseChatCompletions || !responses {
		result, err = h.gatewayService.ForwardAsChatCompletions(c.Request.Context(), prepContext, selection.Account, prepBody, "", "")
	} else {
		result, err = h.gatewayService.Forward(c.Request.Context(), prepContext, selection.Account, prepBody)
	}
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.New("preparation returned no forwarding result")
	}
	document := extractGPT6PreparationDocument(recorder.Body.Bytes())
	if document == "" {
		return nil, errors.New("preparation returned an empty requirements document")
	}
	return &gpt6PreparationResult{Account: selection.Account, Result: result, Model: candidate.Model, Document: document}, nil
}

func responsesPreparationBodyToChat(body []byte) ([]byte, error) {
	var responsesReq apicompat.ResponsesRequest
	if err := json.Unmarshal(body, &responsesReq); err != nil {
		return nil, err
	}
	chatReq, err := apicompat.ResponsesToChatCompletionsRequest(&responsesReq)
	if err != nil {
		return nil, err
	}
	chatReq.Tools = nil
	chatReq.ToolChoice = nil
	chatReq.ParallelToolCalls = nil
	chatReq.MaxTokens = chatReq.MaxCompletionTokens
	chatReq.MaxCompletionTokens = nil
	return json.Marshal(chatReq)
}

func gpt6PreparationRequestID(upstreamRequestID string, body []byte, publicModel, actualModel string) string {
	upstreamRequestID = strings.TrimSpace(upstreamRequestID)
	if strings.HasPrefix(upstreamRequestID, "gpt6-prep:") {
		return upstreamRequestID
	}
	if upstreamRequestID != "" {
		return "gpt6-prep:" + upstreamRequestID
	}
	hashInput := append(append(append([]byte(nil), body...), []byte(publicModel)...), []byte(actualModel+time.Now().UTC().String())...)
	hash := sha256.Sum256(hashInput)
	return "gpt6-prep:" + hex.EncodeToString(hash[:8])
}

func (h *OpenAIGatewayHandler) submitGPT6PreparationUsage(c *gin.Context, apiKey *service.APIKey, account *service.Account, result *service.OpenAIForwardResult, publicModel string, multiplier float64, requestBody []byte) {
	if multiplier <= 0 {
		multiplier = 12
	}
	clientIP := ""
	if c != nil {
		clientIP = c.ClientIP()
	}
	input := &service.OpenAIRecordUsageInput{
		Result: result, APIKey: apiKey, User: apiKey.User, Account: account,
		InboundEndpoint: c.Request.URL.Path, UpstreamEndpoint: result.UpstreamEndpoint,
		UserAgent: c.GetHeader("User-Agent"), IPAddress: clientIP,
		RequestPayloadHash: service.HashUsageRequestPayload(requestBody),
		APIKeyService:      h.apiKeyService, PricingAt: time.Now(),
		BillingMultiplierOverride: &multiplier,
		ChannelUsageFields: service.ChannelUsageFields{
			OriginalModel: publicModel, ChannelMappedModel: result.BillingModel,
			BillingModelSource: service.BillingModelSourceUpstream,
		},
	}
	h.submitOpenAIUsageRecordTask(c.Request.Context(), result, func(ctx context.Context) {
		_ = h.gatewayService.RecordUsage(ctx, input)
	})
}

func buildGPT6PreparationBody(body []byte, model string, maxOutput int, responses bool) ([]byte, error) {
	var original map[string]any
	if err := json.Unmarshal(body, &original); err != nil {
		return nil, err
	}
	payload := map[string]any{"model": model, "stream": false}
	if responses {
		instructions := gpt6PreparationInstruction
		if originalInstructions, ok := original["instructions"].(string); ok && strings.TrimSpace(originalInstructions) != "" {
			instructions += "\n\nOriginal request instructions:\n" + originalInstructions
		}
		payload["instructions"] = instructions
		payload["input"] = gpt6PreparationInputWithoutTools(original["input"])
		payload["max_output_tokens"] = maxOutput
	} else {
		payload["max_tokens"] = maxOutput
		messages, _ := original["messages"].([]any)
		payload["messages"] = append([]any{map[string]any{"role": "system", "content": gpt6PreparationInstruction}}, messages...)
	}
	return json.Marshal(payload)
}

func gpt6PreparationInputWithoutTools(input any) any {
	items, ok := input.([]any)
	if !ok {
		return input
	}
	filtered := make([]any, 0, len(items))
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			filtered = append(filtered, item)
			continue
		}
		switch object["type"] {
		case "additional_tools":
			continue
		case "tool_search_output":
			if _, hasTools := object["tools"]; hasTools {
				if _, hasOutput := object["output"]; !hasOutput {
					continue
				}
				copy := make(map[string]any, len(object)-1)
				for key, value := range object {
					if key != "tools" {
						copy[key] = value
					}
				}
				item = copy
			}
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func appendGPT6Requirements(body []byte, document string) []byte {
	var payload map[string]any
	if json.Unmarshal(body, &payload) != nil {
		return body
	}
	addition := "\n\nInternal requirements document prepared for the final GPT6 executor:\n" + document
	if current, ok := payload["instructions"].(string); ok && strings.TrimSpace(current) != "" {
		payload["instructions"] = current + addition
	} else if _, hasInstructions := payload["instructions"]; !hasInstructions {
		payload["instructions"] = addition
	} else if current, ok := payload["input"].(string); ok {
		payload["input"] = current + addition
	} else if current, ok := payload["input"].([]any); ok {
		payload["input"] = append(current, map[string]any{
			"role": "user", "content": addition,
		})
	} else {
		// Preserve structured instructions exactly and add the document as a
		// separate input item understood by the Responses API.
		payload["input"] = []any{map[string]any{"role": "user", "content": addition}}
	}
	if out, err := json.Marshal(payload); err == nil {
		return out
	}
	return body
}

func extractGPT6PreparationDocument(body []byte) string {
	paths := []string{
		"choices.0.message.content", "output.0.content.0.text",
		"output.#(type==\"message\").content.0.text", "content.0.text",
	}
	for _, path := range paths {
		if value := strings.TrimSpace(gjson.GetBytes(body, path).String()); value != "" {
			return value
		}
	}
	return strings.TrimSpace(string(body))
}

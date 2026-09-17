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
	if apiKey.Group.Platform != service.PlatformOpenAI && apiKey.Group.Platform != service.PlatformComposite {
		return false
	}
	return service.IsGPT6Model(model)
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

	prepBody, err := buildGPT6PreparationBody(body, prepModel, maxOutput, responses)
	if err != nil {
		return nil, fmt.Errorf("build GPT6 requirements request: %w", err)
	}
	selection, err := h.selectGPT6PreparationAccount(c.Request.Context(), apiKey, prepModel, responses)
	if err != nil {
		return nil, fmt.Errorf("select GPT6 preparation account: %w", err)
	}
	prepAccount := selection.Account
	if selection.UseChatCompletions && responses {
		prepBody, err = responsesPreparationBodyToChat(prepBody)
		if err != nil {
			return nil, fmt.Errorf("convert GPT6 requirements request to chat completions: %w", err)
		}
	}

	// Forward against an isolated recorder. Forward mutates the Gin context and
	// writes the upstream response, so it must never receive the client writer.
	recorder := httptest.NewRecorder()
	prepContext, _ := gin.CreateTestContext(recorder)
	prepContext.Request = c.Request.Clone(c.Request.Context())
	prepContext.Request.Body = io.NopCloser(bytes.NewReader(prepBody))
	prepContext.Request.ContentLength = int64(len(prepBody))
	prepContext.Request.Header = c.Request.Header.Clone()
	for key, value := range c.Keys {
		prepContext.Set(key, value)
	}
	// The preparation pass always uses an HTTP upstream request. A copied
	// WebSocket marker would otherwise make Forward select a WS transport for
	// this isolated internal call.
	service.SetOpenAIClientTransport(prepContext, service.OpenAIClientTransportHTTP)
	prepContext.Set("gpt6_preparation", true)

	var result *service.OpenAIForwardResult
	var forwardErr error
	if selection.UseChatCompletions {
		result, forwardErr = h.gatewayService.ForwardAsChatCompletions(c.Request.Context(), prepContext, prepAccount, prepBody, "", "")
	} else if responses {
		result, forwardErr = h.gatewayService.Forward(c.Request.Context(), prepContext, prepAccount, prepBody)
	} else {
		result, forwardErr = h.gatewayService.ForwardAsChatCompletions(c.Request.Context(), prepContext, prepAccount, prepBody, "", "")
	}
	if forwardErr != nil {
		return nil, fmt.Errorf("GPT6 requirements request failed: %w", forwardErr)
	}
	if result == nil {
		return nil, errors.New("GPT6 preparation returned no forwarding result")
	}
	document := extractGPT6PreparationDocument(recorder.Body.Bytes())
	if document == "" {
		return nil, errors.New("GPT6 preparation returned an empty requirements document")
	}
	if len(document) > gpt6PreparationMaxDocumentBytes {
		document = document[:gpt6PreparationMaxDocumentBytes]
	}

	// The preparation row keeps the public alias while its BillingModel and
	// UpstreamModel identify the real cheap model for admin accounting.
	actualPrepModel := strings.TrimSpace(result.UpstreamModel)
	if actualPrepModel == "" {
		actualPrepModel = prepModel
	}
	result.Model = model
	result.BillingModel = actualPrepModel
	result.UpstreamModel = actualPrepModel
	result.RequestID = gpt6PreparationRequestID(result.RequestID, body, model, actualPrepModel)
	h.submitGPT6PreparationUsage(c, apiKey, prepAccount, result, model, prep.PreparationMultiplier, body)

	return appendGPT6Requirements(body, document), nil
}

type gpt6PreparationSelection struct {
	Account            *service.Account
	UseChatCompletions bool
}

func (h *OpenAIGatewayHandler) selectGPT6PreparationAccount(ctx context.Context, apiKey *service.APIKey, prepModel string, responses bool) (gpt6PreparationSelection, error) {
	if apiKey == nil || apiKey.GroupID == nil {
		return gpt6PreparationSelection{}, errors.New("GPT6 preparation requires a group")
	}
	// Suppress the composite execution candidates: they intentionally prefer
	// OpenAI for the final task, while this pass explicitly prefers DeepSeek.
	prepCtx := service.WithoutCompositeRouteCandidates(ctx)
	candidates := []struct {
		model              string
		platform           string
		capability         service.OpenAIEndpointCapability
		useChatCompletions bool
	}{
		// DeepSeek accounts are OpenAI-compatible; selecting Chat Completions
		// keeps fixed-chat accounts eligible while their account protocol still
		// controls whether the actual upstream request is native Responses.
		{model: prepModel, platform: service.PlatformDeepseek, capability: service.OpenAIEndpointCapabilityChatCompletions, useChatCompletions: true},
		{model: "gpt-5.4-mini", platform: service.PlatformOpenAI, capability: service.OpenAIEndpointCapabilityResponses, useChatCompletions: false},
		{model: "gpt-5-mini", platform: service.PlatformOpenAI, capability: service.OpenAIEndpointCapabilityResponses, useChatCompletions: false},
	}
	if !responses {
		for i := 1; i < len(candidates); i++ {
			candidates[i].capability = service.OpenAIEndpointCapabilityChatCompletions
		}
	}
	var lastErr error
	for _, candidate := range candidates {
		selection, _, err := h.gatewayService.SelectAccountWithSchedulerForCapability(
			prepCtx, apiKey.GroupID, "", "", candidate.model, nil,
			service.OpenAIUpstreamTransportHTTPSSE, candidate.capability, false, false, true, candidate.platform,
		)
		if err == nil && selection != nil && selection.Account != nil {
			if selection.ReleaseFunc != nil {
				selection.ReleaseFunc()
			}
			return gpt6PreparationSelection{Account: selection.Account, UseChatCompletions: candidate.useChatCompletions}, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("no eligible preparation account")
	}
	return gpt6PreparationSelection{}, lastErr
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
		multiplier = 6
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
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	payload["model"] = model
	payload["stream"] = false
	if responses {
		instructions := gpt6PreparationInstruction
		if original, ok := payload["instructions"].(string); ok && strings.TrimSpace(original) != "" {
			instructions += "\n\nOriginal request instructions:\n" + original
		}
		payload["instructions"] = instructions
		payload["max_output_tokens"] = maxOutput
		delete(payload, "max_tokens")
	} else {
		payload["max_tokens"] = maxOutput
		delete(payload, "max_completion_tokens")
		messages, _ := payload["messages"].([]any)
		payload["messages"] = append([]any{map[string]any{"role": "system", "content": gpt6PreparationInstruction}}, messages...)
	}
	delete(payload, "tools")
	delete(payload, "tool_choice")
	delete(payload, "parallel_tool_calls")
	delete(payload, "previous_response_id")
	return json.Marshal(payload)
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

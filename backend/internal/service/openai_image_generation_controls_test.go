package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenAIGatewayServiceForward_RejectsDisabledImageGenerationIntents(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		body []byte
	}{
		{
			name: "image model",
			body: []byte(`{"model":"gpt-image-2","input":"draw"}`),
		},
		{
			name: "image tool",
			body: []byte(`{"model":"gpt-5.4","input":"draw","tools":[{"type":"image_generation"}]}`),
		},
		{
			name: "image tool choice",
			body: []byte(`{"model":"gpt-5.4","input":"draw","tool_choice":{"type":"image_generation"}}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := &httpUpstreamRecorder{}
			svc := newOpenAIImageGenerationControlTestService(upstream)
			c, recorder := newOpenAIImageGenerationControlTestContext(false, "unit-test-agent/1.0")
			account := newOpenAIImageGenerationControlTestAccount()

			result, err := svc.Forward(context.Background(), c, account, tt.body)

			require.Error(t, err)
			require.Nil(t, result)
			require.Equal(t, http.StatusForbidden, recorder.Code)
			require.Equal(t, "permission_error", gjson.GetBytes(recorder.Body.Bytes(), "error.type").String())
			require.Nil(t, upstream.lastReq, "disabled image request must not reach upstream")
		})
	}
}

func TestOpenAIGatewayServiceForward_DisabledGroupAllowsTextOnlyResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := &httpUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"resp_text","model":"gpt-5.4","usage":{"input_tokens":3,"output_tokens":2}}`)),
		},
	}
	svc := newOpenAIImageGenerationControlTestService(upstream)
	c, recorder := newOpenAIImageGenerationControlTestContext(false, "unit-test-agent/1.0")
	account := newOpenAIImageGenerationControlTestAccount()

	result, err := svc.Forward(context.Background(), c, account, []byte(`{"model":"gpt-5.4","input":"write code","stream":false}`))

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, 3, result.Usage.InputTokens)
	require.Equal(t, 2, result.Usage.OutputTokens)
	require.Equal(t, 0, result.ImageCount)
	require.NotNil(t, upstream.lastReq)
}

func TestOpenAIGatewayServiceForward_CodexImageInjectionRespectsGroupCapability(t *testing.T) {
	gin.SetMode(gin.TestMode)
	legacyDisabled, legacyEnabled := false, true

	tests := []struct {
		name           string
		allowImages    bool
		responsesLite  bool
		wantInjected   bool
		legacyOverride *bool
		stripTools     bool
		model          string
		passthrough    bool
		modelMapping   map[string]any
		wantImageModel string
		wantLiteHint   bool
	}{
		{name: "disabled group skips injection", allowImages: false, wantInjected: false},
		{name: "enabled group provides image tool", allowImages: true, wantInjected: true},
		{name: "responses lite exposes existing provider route", allowImages: true, responsesLite: true, wantLiteHint: true},
		{name: "responses lite disabled group receives no image guidance", responsesLite: true},
		{name: "legacy account and channel disable cannot block enabled group", allowImages: true, legacyOverride: &legacyDisabled, wantInjected: true},
		{name: "legacy account and channel enable cannot bypass disabled group", allowImages: false, legacyOverride: &legacyEnabled, wantInjected: false},
		{name: "legacy strip cannot override enabled group", allowImages: true, stripTools: true, wantInjected: true},
		{name: "gpt6 luna uses group permission", allowImages: true, model: "gpt-6-luna", wantInjected: true},
		{name: "gpt6 sol uses group permission", allowImages: true, model: "gpt-6-sol", wantInjected: true},
		{name: "gpt5 uses group permission", allowImages: true, model: "gpt-5.5", wantInjected: true},
		{name: "passthrough enabled group provides image tool", allowImages: true, passthrough: true, wantInjected: true},
		{name: "passthrough disabled group skips injection", passthrough: true},
		{name: "passthrough ignores legacy strip", allowImages: true, passthrough: true, stripTools: true, wantInjected: true},
		{name: "passthrough lite exposes existing provider route", allowImages: true, passthrough: true, responsesLite: true, wantLiteHint: true},
		{name: "lite cannot bypass image allowlist", allowImages: true, responsesLite: true, modelMapping: map[string]any{"gpt-5.4": "gpt-5.4"}},
		{name: "spark skips injection", allowImages: true, model: "gpt-5.3-codex-spark"},
		{name: "passthrough spark skips injection", allowImages: true, passthrough: true, model: "gpt-5.3-codex-spark"},
		{name: "enabled group cannot authorize an image model", allowImages: true, modelMapping: map[string]any{"gpt-5.4": "gpt-5.4"}},
		{name: "passthrough cannot bypass image allowlist", allowImages: true, passthrough: true, modelMapping: map[string]any{"gpt-5.4": "gpt-5.4"}},
		{name: "authorized image model is offered", allowImages: true, modelMapping: map[string]any{"gpt-5.4": "gpt-5.4", "gpt-image-2": "gpt-image-2"}, wantInjected: true},
		{name: "only authorized image variant is offered", allowImages: true, modelMapping: map[string]any{"gpt-5.4": "gpt-5.4", "gpt-image-2.5-flare": "gpt-image-2.5-flare"}, wantInjected: true, wantImageModel: "gpt-image-2.5-flare"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := &httpUpstreamRecorder{
				resp: &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"id":"resp_codex","model":"gpt-5.4","usage":{"input_tokens":1,"output_tokens":1}}`)),
				},
			}
			svc := newOpenAIImageGenerationControlTestService(upstream)
			c, _ := newOpenAIImageGenerationControlTestContext(tt.allowImages, "codex_cli_rs/0.98.0")
			if tt.responsesLite {
				c.Request.Header.Set(responsesLiteHeader, "true")
			}
			account := newOpenAIImageGenerationControlTestAccount()
			if account.Extra == nil {
				account.Extra = make(map[string]any)
			}
			if tt.legacyOverride != nil {
				account.Extra["codex_image_generation_bridge"] = *tt.legacyOverride
				account.Extra["codex_image_generation_bridge_enabled"] = *tt.legacyOverride
				account.Extra[PlatformOpenAI] = map[string]any{"codex_image_generation_bridge_enabled": *tt.legacyOverride}
				svc.channelService = newOpenAIImageGenerationControlChannelService(4242, &Channel{
					ID: 9001, Status: StatusActive,
					FeaturesConfig: map[string]any{"codex_image_generation_bridge": map[string]any{PlatformOpenAI: *tt.legacyOverride}},
				})
			}
			if tt.stripTools {
				account.Extra["codex_image_generation_explicit_tool_policy"] = "strip"
				account.Extra[PlatformOpenAI] = map[string]any{"codex_image_generation_explicit_tool_policy": "strip"}
			}
			account.Extra["openai_passthrough"] = tt.passthrough
			if tt.modelMapping != nil {
				account.Credentials["model_mapping"] = tt.modelMapping
			}
			model := tt.model
			if model == "" {
				model = "gpt-5.4"
			}

			result, err := svc.Forward(context.Background(), c, account, []byte(`{"model":"`+model+`","input":"write code","stream":false}`))

			require.NoError(t, err)
			require.NotNil(t, result)
			require.Zero(t, result.ImageCount, "tool availability must not bill a text response as an image")
			require.NotNil(t, upstream.lastReq)
			hasImageTool := gjson.GetBytes(upstream.lastBody, `tools.#(type=="image_generation")`).Exists()
			require.Equal(t, tt.wantInjected, hasImageTool)
			expectedLiteHeader := ""
			if tt.responsesLite {
				expectedLiteHeader = "true"
			}
			require.Equal(t, expectedLiteHeader, upstream.lastReq.Header.Get(responsesLiteHeader))
			instructions := gjson.GetBytes(upstream.lastBody, "instructions").String()
			require.Equal(t, tt.wantLiteHint, strings.Contains(instructions, codexImageAPIAvailableMarker))
			require.Equal(t, tt.wantInjected, strings.Contains(instructions, "image_generation"))
			toolChoice := gjson.GetBytes(upstream.lastBody, "tool_choice")
			require.Equal(t, tt.wantInjected, toolChoice.Exists())
			if tt.wantInjected {
				require.Equal(t, "auto", toolChoice.String())
				wantModel := tt.wantImageModel
				if wantModel == "" {
					wantModel = openAIImagesDefaultModel
				}
				require.Equal(t, wantModel, gjson.GetBytes(upstream.lastBody, `tools.#(type=="image_generation").model`).String())
			}
		})
	}
}

func TestOpenAIBuildUpstreamRequestOpenAIPassthroughForwardsResponsesLiteHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := newOpenAIImageGenerationControlTestContext(true, "codex_cli_rs/0.98.0")
	c.Request.Header.Set(responsesLiteHeader, "true")

	svc := newOpenAIImageGenerationControlTestService(&httpUpstreamRecorder{})
	req, err := svc.buildUpstreamRequestOpenAIPassthrough(
		c.Request.Context(),
		c,
		newOpenAIImageGenerationControlTestAccount(),
		[]byte(`{"model":"gpt-5.4","input":"write code"}`),
		"test-token",
	)

	require.NoError(t, err)
	require.Equal(t, "true", req.Header.Get(responsesLiteHeader))
}

func TestOpenAIGatewayServiceForward_ImageModelRequiresAccountAuthorization(t *testing.T) {
	for _, passthrough := range []bool{false, true} {
		for _, body := range []string{
			`{"model":"gpt-image-2","input":"draw"}`,
			`{"model":"gpt-6-luna","tools":[{"type":"image_generation"}]}`,
			`{"model":"gpt-6-luna","tools":[{"type":"image_generation","model":"gpt-image-2.5-flare"}]}`,
			`{"model":"gpt-6-luna","tools":[{"type":"image_generation","model":"gpt-image-2"},{"type":"image_generation","model":"gpt-image-2.5-flare"}]}`,
			`{"model":"gpt-6-luna","input":[{"type":"additional_tools","tools":[{"type":"image_generation"}]}]}`,
		} {
			upstream := &httpUpstreamRecorder{}
			svc := newOpenAIImageGenerationControlTestService(upstream)
			c, recorder := newOpenAIImageGenerationControlTestContext(true, "codex_cli_rs/0.144.1")
			account := newOpenAIImageGenerationControlTestAccount()
			account.Credentials["model_mapping"] = map[string]any{"gpt-6-luna": "gpt-6-luna"}
			account.Extra = map[string]any{"openai_passthrough": passthrough}
			result, err := svc.Forward(context.Background(), c, account, []byte(body))
			require.ErrorContains(t, err, "not authorized")
			require.Nil(t, result)
			require.Equal(t, http.StatusForbidden, recorder.Code)
			require.Nil(t, upstream.lastReq)
		}
	}
}

func TestOpenAIGatewayServiceForward_ExplicitImageToolUsesGroupPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := &httpUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"resp_explicit_image","model":"gpt-5.4","usage":{"input_tokens":2,"output_tokens":1}}`)),
		},
	}
	svc := newOpenAIImageGenerationControlTestService(upstream)
	c, _ := newOpenAIImageGenerationControlTestContext(true, "codex_cli_rs/0.98.0")
	account := newOpenAIImageGenerationControlTestAccount()
	body := []byte(`{"model":"gpt-6-luna","input":"draw","stream":false,"tools":[{"type":"image_generation","format":"jpeg"}]}`)

	result, err := svc.Forward(context.Background(), c, account, body)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, upstream.lastReq)
	require.True(t, gjson.GetBytes(upstream.lastBody, `tools.#(type=="image_generation")`).Exists())
	require.Equal(t, openAIImagesDefaultModel, gjson.GetBytes(upstream.lastBody, `tools.#(type=="image_generation").model`).String())
	require.Equal(t, "gpt-6-luna", gjson.GetBytes(upstream.lastBody, "model").String())
	require.Equal(t, "jpeg", gjson.GetBytes(upstream.lastBody, `tools.#(type=="image_generation").output_format`).String())
	require.False(t, gjson.GetBytes(upstream.lastBody, `tools.#(type=="image_generation").format`).Exists())
	instructions := gjson.GetBytes(upstream.lastBody, "instructions").String()
	require.NotContains(t, instructions, "image_generation")
}

func TestOpenAIGatewayServiceForward_LegacyPolicyCannotStripAuthorizedImageTool(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := &httpUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"resp_stripped_image","model":"gpt-5.4","usage":{"input_tokens":2,"output_tokens":1}}`)),
		},
	}
	svc := newOpenAIImageGenerationControlTestService(upstream)
	c, _ := newOpenAIImageGenerationControlTestContext(true, "codex_cli_rs/0.98.0")
	account := newOpenAIImageGenerationControlTestAccount()
	account.Extra = map[string]any{
		"codex_image_generation_explicit_tool_policy": "strip",
	}
	body := []byte(`{
		"model":"gpt-5.4",
		"input":"draw",
		"stream":false,
		"tools":[
			{"type":"function","name":"shell","parameters":{"type":"object"}},
			{"type":"image_generation","format":"jpeg"}
		],
		"tool_choice":{"type":"image_generation"}
	}`)

	result, err := svc.Forward(context.Background(), c, account, body)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, upstream.lastReq)
	require.True(t, gjson.GetBytes(upstream.lastBody, `tools.#(type=="image_generation")`).Exists())
	require.True(t, gjson.GetBytes(upstream.lastBody, `tools.#(type=="function")`).Exists())
	require.Equal(t, "image_generation", gjson.GetBytes(upstream.lastBody, "tool_choice.type").String())
	instructions := gjson.GetBytes(upstream.lastBody, "instructions").String()
	require.NotContains(t, instructions, "image_generation")
}

func TestOpenAIGatewayServiceForward_DisabledGroupRejectsImageNamespaceDespiteLegacyPolicy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		passthrough bool
	}{
		{name: "managed forwarding"},
		{name: "passthrough forwarding", passthrough: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := &httpUpstreamRecorder{
				resp: &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"id":"resp_stripped_namespace","model":"gpt-5.5","usage":{"input_tokens":2,"output_tokens":1}}`)),
				},
			}
			svc := newOpenAIImageGenerationControlTestService(upstream)
			c, recorder := newOpenAIImageGenerationControlTestContext(false, "codex_cli_rs/0.144.1")
			SetOpenAIClientTransport(c, OpenAIClientTransportHTTP)
			account := newOpenAIImageGenerationControlTestAccount()
			account.Extra = map[string]any{
				"codex_image_generation_explicit_tool_policy": "strip",
				"openai_passthrough":                          tt.passthrough,
			}
			body := []byte(`{
				"model":"gpt-5.5",
				"stream":false,
				"tools":[
					{"type":"function","name":"shell","parameters":{"type":"object"}},
					{"type":"namespace","name":"image_gen","tools":[{"type":"function","name":"imagegen"}]},
					{"type":"namespace","name":"code_tools","tools":[{"type":"function","name":"run"}]}
				],
				"input":[
					{"type":"message","role":"user","content":[{"type":"input_text","text":"write code"}]},
					{"type":"additional_tools","tools":[{"type":"namespace","name":"image_gen","tools":[{"type":"function","name":"imagegen"}]}]}
				],
				"tool_choice":"auto"
			}`)

			result, err := svc.Forward(context.Background(), c, account, body)

			require.Error(t, err)
			require.Nil(t, result)
			require.Equal(t, http.StatusForbidden, recorder.Code)
			require.Nil(t, upstream.lastReq)
		})
	}
}

func TestOpenAIGatewayServiceForward_CodexBridgeDoesNotInjectHostedToolAlongsideImageGenNamespace(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := &httpUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"resp_namespace_image","model":"gpt-5.5","usage":{"input_tokens":1,"output_tokens":1}}`)),
		},
	}
	svc := newOpenAIImageGenerationControlTestService(upstream)
	c, _ := newOpenAIImageGenerationControlTestContext(true, "codex_cli_rs/0.144.1")
	account := newOpenAIImageGenerationControlTestAccount()
	body := []byte(`{
		"model":"gpt-5.5",
		"stream":false,
		"tools":[
			{"type":"function","name":"shell","parameters":{"type":"object"}},
			{"type":"namespace","name":"image_gen","tools":[{"type":"function","name":"imagegen"}]}
		],
		"input":[
			{"type":"message","role":"user","content":[{"type":"input_text","text":"draw a cat"}]},
			{"type":"additional_tools","tools":[{"type":"namespace","name":"image_gen","tools":[{"type":"function","name":"imagegen"}]}]}
		],
		"tool_choice":"auto"
	}`)

	result, err := svc.Forward(context.Background(), c, account, body)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, upstream.lastReq)
	require.False(t, gjson.GetBytes(upstream.lastBody, `tools.#(type=="image_generation")`).Exists())
	require.Equal(t, "namespace", gjson.GetBytes(upstream.lastBody, `tools.#(name=="image_gen").type`).String())
	require.Equal(t, "namespace", gjson.GetBytes(upstream.lastBody, `input.#(type=="additional_tools").tools.#(name=="image_gen").type`).String())
}

func TestOpenAIGatewayServiceForward_CodexBridgePreservesImageGenFunction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		tool string
	}{
		{
			name: "flat function",
			tool: `{"type":"function","name":"image_gen.imagegen","parameters":{"type":"object"}}`,
		},
		{
			name: "nested function",
			tool: `{"type":"function","function":{"name":"image_gen.imagegen","parameters":{"type":"object"}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := &httpUpstreamRecorder{
				resp: &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"id":"resp_function_image","model":"gpt-5.5","usage":{"input_tokens":1,"output_tokens":1}}`)),
				},
			}
			svc := newOpenAIImageGenerationControlTestService(upstream)
			c, _ := newOpenAIImageGenerationControlTestContext(true, "codex_cli_rs/0.144.1")
			account := newOpenAIImageGenerationControlTestAccount()
			body := []byte(`{"model":"gpt-5.5","input":"draw a cat","stream":false,"tools":[` + tt.tool + `]}`)

			result, err := svc.Forward(context.Background(), c, account, body)

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, upstream.lastReq)

			var forwarded map[string]any
			require.NoError(t, json.Unmarshal(upstream.lastBody, &forwarded))
			require.True(t, hasCodexImageGenerationFunctionTool(forwarded))
			require.False(t, gjson.GetBytes(upstream.lastBody, `tools.#(type=="image_generation")`).Exists())
			require.False(t, gjson.GetBytes(upstream.lastBody, "tool_choice").Exists())
			require.NotContains(t, gjson.GetBytes(upstream.lastBody, "instructions").String(), codexImageGenerationBridgeMarker)
		})
	}
}

func TestOpenAIGatewayServiceForward_CodexBridgePreservesExistingToolChoice(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := &httpUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"resp_codex_tool_choice","model":"gpt-5.4","usage":{"input_tokens":1,"output_tokens":1}}`)),
		},
	}
	svc := newOpenAIImageGenerationControlTestService(upstream)
	c, _ := newOpenAIImageGenerationControlTestContext(true, "codex_cli_rs/0.98.0")
	account := newOpenAIImageGenerationControlTestAccount()

	result, err := svc.Forward(context.Background(), c, account, []byte(`{"model":"gpt-5.4","input":"draw","stream":false,"tools":[{"type":"image_generation"}],"tool_choice":{"type":"image_generation"}}`))

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "image_generation", gjson.GetBytes(upstream.lastBody, "tool_choice.type").String())
}

func TestOpenAIGatewayServiceForward_CodexBridgeSkipsCompactRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := &httpUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"resp_codex_compact","model":"gpt-5.4","usage":{"input_tokens":1,"output_tokens":1}}`)),
		},
	}
	svc := newOpenAIImageGenerationControlTestService(upstream)
	c, _ := newOpenAIImageGenerationControlTestContext(true, "codex_cli_rs/0.98.0")
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses/compact", nil)
	c.Request.Header.Set("User-Agent", "codex_cli_rs/0.98.0")
	account := newOpenAIImageGenerationControlTestAccount()

	// /responses/compact 上游不接受 tool_choice，bridge 注入必须整体豁免 compact 请求。
	result, err := svc.Forward(context.Background(), c, account, []byte(`{"model":"gpt-5.4","input":"summarize the conversation","stream":false}`))

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, upstream.lastReq)
	require.False(t, gjson.GetBytes(upstream.lastBody, "tool_choice").Exists())
	require.False(t, gjson.GetBytes(upstream.lastBody, `tools.#(type=="image_generation")`).Exists())
	instructions := gjson.GetBytes(upstream.lastBody, "instructions").String()
	require.NotContains(t, instructions, "image_generation")
}

func TestOpenAIGatewayServiceHandleResponsesImageOutputs_NonStreaming(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := newOpenAIImageGenerationControlTestService(&httpUpstreamRecorder{})
	c, _ := newOpenAIImageGenerationControlTestContext(true, "unit-test-agent/1.0")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body: io.NopCloser(strings.NewReader(`{
			"id":"resp_image_json",
			"model":"gpt-5.4",
			"output":[{"id":"ig_json_1","type":"image_generation_call","result":"final-image"}],
			"usage":{"input_tokens":7,"output_tokens":3,"output_tokens_details":{"image_tokens":2}}
		}`)),
	}

	result, err := svc.handleNonStreamingResponse(context.Background(), resp, c, &Account{ID: 1, Type: AccountTypeAPIKey}, "gpt-5.4", "gpt-5.4")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 1, result.imageCount)
	require.NotNil(t, result.usage)
	require.Equal(t, 7, result.usage.InputTokens)
	require.Equal(t, 3, result.usage.OutputTokens)
	require.Equal(t, 2, result.usage.ImageOutputTokens)
}

func TestOpenAIGatewayServiceHandleResponsesImageOutputs_Streaming(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := newOpenAIImageGenerationControlTestService(&httpUpstreamRecorder{})
	c, recorder := newOpenAIImageGenerationControlTestContext(true, "unit-test-agent/1.0")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(
			"data: {\"type\":\"response.output_item.done\",\"item\":{\"id\":\"ig_stream_1\",\"type\":\"image_generation_call\",\"status\":\"generating\",\"result\":\"final-image\"}}\n\n" +
				"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_image_stream\",\"model\":\"gpt-5.5\",\"output\":[{\"id\":\"ig_stream_1\",\"type\":\"image_generation_call\",\"status\":\"generating\",\"result\":\"final-image\"}],\"usage\":{\"input_tokens\":11,\"output_tokens\":5,\"output_tokens_details\":{\"image_tokens\":4}}}}\n\n",
		)),
	}

	result, err := svc.handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "gpt-5.5", "gpt-5.5")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 1, result.imageCount)
	require.NotNil(t, result.usage)
	require.Equal(t, 11, result.usage.InputTokens)
	require.Equal(t, 5, result.usage.OutputTokens)
	require.Equal(t, 4, result.usage.ImageOutputTokens)
	require.NotContains(t, recorder.Body.String(), `"status":"generating"`)
	require.Equal(t, 2, strings.Count(recorder.Body.String(), `"status":"completed"`))
}

func TestOpenAIGatewayServiceHandleResponsesImageOutputs_StreamingPassthrough(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := newOpenAIImageGenerationControlTestService(&httpUpstreamRecorder{})
	c, recorder := newOpenAIImageGenerationControlTestContext(true, "unit-test-agent/1.0")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(
			"data: {\"type\":\"response.output_item.done\",\"item\":{\"id\":\"ig_stream_1\",\"type\":\"image_generation_call\",\"status\":\"in_progress\",\"result\":\"final-image\"}}\n\n" +
				"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_image_stream\",\"model\":\"gpt-5.5\",\"output\":[{\"id\":\"ig_stream_1\",\"type\":\"image_generation_call\",\"status\":\"in_progress\",\"result\":\"final-image\"}],\"usage\":{\"input_tokens\":11,\"output_tokens\":5,\"output_tokens_details\":{\"image_tokens\":4}}}}\n\n",
		)),
	}

	result, err := svc.handleStreamingResponsePassthrough(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "gpt-5.5", "gpt-5.5")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotContains(t, recorder.Body.String(), `"status":"in_progress"`)
	require.Equal(t, 2, strings.Count(recorder.Body.String(), `"status":"completed"`))
}

func TestNormalizeCompletedImageGenerationStatus(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        string
		wantChanged bool
	}{
		{
			name:        "output item done with result",
			input:       `{"type":"response.output_item.done","item":{"type":"image_generation_call","status":"generating","result":"image-data"}}`,
			want:        `{"type":"response.output_item.done","item":{"type":"image_generation_call","status":"completed","result":"image-data"}}`,
			wantChanged: true,
		},
		{
			name:        "terminal response only changes completed image result",
			input:       `{"type":"response.completed","response":{"output":[{"type":"image_generation_call","status":"in_progress","result":"image-data"},{"type":"image_generation_call","status":"failed","result":"partial-data"}]}}`,
			want:        `{"type":"response.completed","response":{"output":[{"type":"image_generation_call","status":"completed","result":"image-data"},{"type":"image_generation_call","status":"failed","result":"partial-data"}]}}`,
			wantChanged: true,
		},
		{
			name:        "done item without result",
			input:       `{"type":"response.output_item.done","item":{"type":"image_generation_call","status":"generating"}}`,
			want:        `{"type":"response.output_item.done","item":{"type":"image_generation_call","status":"generating"}}`,
			wantChanged: false,
		},
		{
			name:        "non-final image event",
			input:       `{"type":"response.output_item.added","item":{"type":"image_generation_call","status":"generating","result":"image-data"}}`,
			want:        `{"type":"response.output_item.added","item":{"type":"image_generation_call","status":"generating","result":"image-data"}}`,
			wantChanged: false,
		},
		{
			name:        "done preserves base64 result",
			input:       `{"type":"response.done","response":{"output":[{"type":"image_generation_call","status":"generating","result":"iVBORw0KGgoAAAANSUhEUg/+=="}]}}`,
			want:        `{"type":"response.done","response":{"output":[{"type":"image_generation_call","status":"completed","result":"iVBORw0KGgoAAAANSUhEUg/+=="}]}}`,
			wantChanged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed := normalizeCompletedImageGenerationStatus([]byte(tt.input))

			require.Equal(t, tt.wantChanged, changed)
			require.JSONEq(t, tt.want, string(got))
		})
	}
}

// TestHandleStreamingResponse_CyberPolicyCapturesRealUpstreamTokens 锁定流式
// /v1/responses 命中 cyber_policy 的计费正确性：response.failed 自带的真实 usage
// 必须在打 cyber 标记前被解析进 mark；否则计费走 mark.UpstreamInTok 会按 0 token
// 漏记真实用量（该路径返回错误，handler 仅经 RecordCyberPolicyUsageLog 计费）。
func TestHandleStreamingResponse_CyberPolicyCapturesRealUpstreamTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := newOpenAIImageGenerationControlTestService(&httpUpstreamRecorder{})
	c, _ := newOpenAIImageGenerationControlTestContext(false, "unit-test-agent/1.0")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(
			"data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_cyber\"}}\n\n" +
				"data: {\"type\":\"response.failed\",\"response\":{\"id\":\"resp_cyber\",\"error\":{\"code\":\"cyber_policy\",\"message\":\"blocked by network policy\"},\"usage\":{\"input_tokens\":1234,\"output_tokens\":7}}}\n\n",
		)),
	}

	_, err := svc.handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "gpt-5.5", "gpt-5.5")
	require.Error(t, err, "cyber 命中的流式响应应返回错误（sawFailedEvent）")

	mark := GetOpsCyberPolicy(c)
	require.NotNil(t, mark, "必须打上 cyber 标记")
	require.Equal(t, "cyber_policy", mark.Code)
	require.Equal(t, 1234, mark.UpstreamInTok, "必须捕获 response.failed 自带真实 input token，而非解析前的 0")
	require.Equal(t, 7, mark.UpstreamOutTok)
}

func newOpenAIImageGenerationControlTestService(upstream *httpUpstreamRecorder) *OpenAIGatewayService {
	cfg := &config.Config{}
	return &OpenAIGatewayService{
		cfg:              cfg,
		httpUpstream:     upstream,
		cache:            &stubGatewayCache{},
		openaiWSResolver: NewOpenAIWSProtocolResolver(cfg),
		toolCorrector:    NewCodexToolCorrector(),
	}
}

func newOpenAIImageGenerationControlChannelService(groupID int64, ch *Channel) *ChannelService {
	svc := &ChannelService{}
	cache := newEmptyChannelCache()
	if ch != nil {
		cache.channelByGroupID[groupID] = ch
		cache.byID[ch.ID] = ch
	}
	cache.loadedAt = time.Now()
	svc.cache.Store(cache)
	return svc
}

func newOpenAIImageGenerationControlTestContext(allowImages bool, userAgent string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)
	c.Request.Header.Set("User-Agent", userAgent)
	groupID := int64(4242)
	c.Set("api_key", &APIKey{
		ID:      2424,
		GroupID: &groupID,
		Group: &Group{
			ID:                   groupID,
			AllowImageGeneration: allowImages,
			RateMultiplier:       1,
			ImageRateMultiplier:  1,
		},
	})
	return c, recorder
}

func newOpenAIImageGenerationControlTestAccount() *Account {
	return &Account{
		ID:          5151,
		Name:        "openai-image-controls",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key": "sk-test",
		},
	}
}

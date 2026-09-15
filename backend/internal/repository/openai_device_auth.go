package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (s *openaiOAuthService) deviceAuthRequest(ctx context.Context, proxyURL, path string, body map[string]string, result any) (int, error) {
	client, err := createOpenAIReqClient(proxyURL)
	if err != nil {
		return 0, infraerrors.New(http.StatusBadGateway, "OPENAI_DEVICE_REQUEST_FAILED", "Could not initialize OpenAI connection")
	}
	baseURL := s.deviceAuthURL
	if baseURL == "" {
		baseURL = "https://auth.openai.com/api/accounts/deviceauth"
	}
	ua, originator := service.CodexCanonicalAuthIdentity()
	resp, err := client.R().SetContext(ctx).SetHeader("User-Agent", ua).SetHeader("originator", originator).SetBodyJsonMarshal(body).SetSuccessResult(result).Post(baseURL + path)
	if err != nil {
		return 0, infraerrors.New(http.StatusBadGateway, "OPENAI_DEVICE_REQUEST_FAILED", "Could not reach OpenAI. Check the selected proxy and try again.")
	}
	return resp.StatusCode, nil
}

func (s *openaiOAuthService) StartDeviceAuth(ctx context.Context, proxyURL string) (*service.OpenAIDeviceCode, error) {
	var result struct {
		DeviceAuthID  string          `json:"device_auth_id"`
		UserCode      string          `json:"user_code"`
		UserCodeAlias string          `json:"usercode"`
		Interval      json.RawMessage `json:"interval"`
	}
	status, err := s.deviceAuthRequest(ctx, proxyURL, "/usercode", map[string]string{"client_id": openai.ClientID}, &result)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, infraerrors.New(http.StatusBadGateway, "OPENAI_DEVICE_UNAVAILABLE", "OpenAI device login is unavailable. Enable device code login in ChatGPT security settings, or use manual login.")
	}
	if result.UserCode == "" {
		result.UserCode = result.UserCodeAlias
	}
	if result.DeviceAuthID == "" || result.UserCode == "" {
		return nil, infraerrors.New(http.StatusBadGateway, "OPENAI_DEVICE_INVALID_RESPONSE", "OpenAI returned an invalid device code")
	}
	interval, _ := strconv.Atoi(strings.Trim(strings.TrimSpace(string(result.Interval)), "\""))
	return &service.OpenAIDeviceCode{DeviceAuthID: result.DeviceAuthID, UserCode: result.UserCode, Interval: interval}, nil
}

func (s *openaiOAuthService) PollDeviceAuth(ctx context.Context, deviceAuthID, userCode, proxyURL string) (*service.OpenAIDeviceGrant, error) {
	var result struct {
		AuthorizationCode string `json:"authorization_code"`
		CodeVerifier      string `json:"code_verifier"`
	}
	status, err := s.deviceAuthRequest(ctx, proxyURL, "/token", map[string]string{"device_auth_id": deviceAuthID, "user_code": userCode}, &result)
	if err != nil {
		return nil, err
	}
	if status == http.StatusForbidden || status == http.StatusNotFound {
		return nil, nil
	}
	if status < 200 || status >= 300 {
		return nil, infraerrors.New(http.StatusBadGateway, "OPENAI_DEVICE_POLL_FAILED", "OpenAI device login failed. Generate a new code and try again.")
	}
	if result.AuthorizationCode == "" || result.CodeVerifier == "" {
		return nil, infraerrors.New(http.StatusBadGateway, "OPENAI_DEVICE_INVALID_RESPONSE", "OpenAI returned an invalid authorization response")
	}
	return &service.OpenAIDeviceGrant{AuthorizationCode: result.AuthorizationCode, CodeVerifier: result.CodeVerifier}, nil
}

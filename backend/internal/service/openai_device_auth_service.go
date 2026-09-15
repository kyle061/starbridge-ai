package service

import (
	"context"
	"net/http"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

const openAIDeviceTTL = 15 * time.Minute

// OpenAIDeviceClient is optional so existing OAuth clients can retain manual login.
type OpenAIDeviceClient interface {
	StartDeviceAuth(context.Context, string) (*OpenAIDeviceCode, error)
	PollDeviceAuth(context.Context, string, string, string) (*OpenAIDeviceGrant, error)
}
type OpenAIDeviceCode struct {
	DeviceAuthID, UserCode string
	Interval               int
}
type OpenAIDeviceGrant struct{ AuthorizationCode, CodeVerifier string }
type OpenAIDeviceStartResult struct {
	SessionID       string `json:"session_id"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	Interval        int    `json:"interval"`
	ExpiresAt       int64  `json:"expires_at"`
}
type OpenAIDevicePollResult struct {
	Status    string           `json:"status"`
	Interval  int              `json:"interval"`
	TokenInfo *OpenAITokenInfo `json:"token_info,omitempty"`
}
type openAIDeviceSession struct {
	mu        sync.Mutex
	ownerID   int64
	code      OpenAIDeviceCode
	proxyURL  string
	expiresAt time.Time
	nextPoll  time.Time
	grant     *OpenAIDeviceGrant
	result    *OpenAITokenInfo
	cancelled bool
}

func deviceSessionMissing() error {
	return infraerrors.New(http.StatusBadRequest, "OPENAI_DEVICE_SESSION_EXPIRED", "Device login expired or cancelled. Generate a new code.")
}

func (s *OpenAIOAuthService) StartDeviceAuth(ctx context.Context, ownerID int64, proxyID *int64) (*OpenAIDeviceStartResult, error) {
	client, ok := s.oauthClient.(OpenAIDeviceClient)
	if !ok {
		return nil, infraerrors.New(http.StatusNotImplemented, "OPENAI_DEVICE_UNAVAILABLE", "Device login is unavailable")
	}
	proxyURL := ""
	if proxyID != nil {
		proxy, err := s.proxyRepo.GetByID(ctx, *proxyID)
		if err != nil {
			return nil, infraerrors.New(http.StatusBadRequest, "OPENAI_OAUTH_PROXY_NOT_FOUND", "Proxy not found")
		}
		if proxy != nil {
			proxyURL = proxy.URL()
		}
	}
	code, err := client.StartDeviceAuth(ctx, proxyURL)
	if err != nil {
		return nil, err
	}
	id, err := openai.GenerateSessionID()
	if err != nil {
		return nil, err
	}
	if code.Interval < 5 {
		code.Interval = 5
	}
	if code.Interval > 60 {
		code.Interval = 60
	}
	now := time.Now()
	session := &openAIDeviceSession{ownerID: ownerID, code: *code, proxyURL: proxyURL, expiresAt: now.Add(openAIDeviceTTL), nextPoll: now.Add(time.Duration(code.Interval) * time.Second)}
	s.deviceMu.Lock()
	defer s.deviceMu.Unlock()
	// Bound memory and remove expired sessions without retaining tokens indefinitely.
	for key, previous := range s.deviceSessions {
		if !now.Before(previous.expiresAt) {
			delete(s.deviceSessions, key)
		}
	}
	if len(s.deviceSessions) >= 256 {
		return nil, infraerrors.New(http.StatusTooManyRequests, "OPENAI_DEVICE_BUSY", "Too many device login sessions. Try again later.")
	}
	s.deviceSessions[id] = session
	time.AfterFunc(openAIDeviceTTL, func() {
		s.deviceMu.Lock()
		delete(s.deviceSessions, id)
		s.deviceMu.Unlock()
	})
	return &OpenAIDeviceStartResult{SessionID: id, UserCode: code.UserCode, VerificationURL: "https://auth.openai.com/codex/device", Interval: code.Interval, ExpiresAt: session.expiresAt.Unix()}, nil
}

func (s *OpenAIOAuthService) deviceSession(id string, ownerID int64) (*openAIDeviceSession, error) {
	s.deviceMu.Lock()
	defer s.deviceMu.Unlock()
	session := s.deviceSessions[id]
	if session == nil || session.ownerID != ownerID {
		return nil, deviceSessionMissing()
	}
	if !time.Now().Before(session.expiresAt) {
		delete(s.deviceSessions, id)
		return nil, deviceSessionMissing()
	}
	return session, nil
}

func (s *OpenAIOAuthService) CancelDeviceAuth(id string, ownerID int64) {
	session, err := s.deviceSession(id, ownerID)
	if err != nil {
		return
	}
	session.mu.Lock()
	session.cancelled = true
	session.result = nil
	session.grant = nil
	session.mu.Unlock()
	s.deviceMu.Lock()
	delete(s.deviceSessions, id)
	s.deviceMu.Unlock()
}

func (s *OpenAIOAuthService) PollDeviceAuth(ctx context.Context, id string, ownerID int64) (*OpenAIDevicePollResult, error) {
	session, err := s.deviceSession(id, ownerID)
	if err != nil {
		return nil, err
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.cancelled || !time.Now().Before(session.expiresAt) {
		return nil, deviceSessionMissing()
	}
	if session.result != nil {
		return &OpenAIDevicePollResult{Status: "authorized", TokenInfo: session.result}, nil
	}
	pending := &OpenAIDevicePollResult{Status: "pending", Interval: session.code.Interval}
	if time.Now().Before(session.nextPoll) {
		return pending, nil
	}
	session.nextPoll = time.Now().Add(time.Duration(session.code.Interval) * time.Second)
	if session.grant == nil {
		client, ok := s.oauthClient.(OpenAIDeviceClient)
		if !ok {
			return nil, deviceSessionMissing()
		}
		grant, err := client.PollDeviceAuth(ctx, session.code.DeviceAuthID, session.code.UserCode, session.proxyURL)
		if err != nil {
			return nil, err
		}
		if grant == nil {
			return pending, nil
		}
		session.grant = grant
	}
	token, err := s.oauthClient.ExchangeCode(ctx, session.grant.AuthorizationCode, session.grant.CodeVerifier, "https://auth.openai.com/deviceauth/callback", session.proxyURL, openai.ClientID)
	if err != nil {
		return nil, err
	}
	if token == nil || token.AccessToken == "" {
		return nil, infraerrors.New(http.StatusBadGateway, "OPENAI_DEVICE_INVALID_TOKEN", "OpenAI returned an empty token")
	}
	// Cache the result for same-owner retries if a mobile connection loses the response.
	session.result = s.tokenInfoFromResponse(ctx, token, openai.ClientID, session.proxyURL)
	session.grant = nil
	return &OpenAIDevicePollResult{Status: "authorized", TokenInfo: session.result}, nil
}

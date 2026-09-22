//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/resend/resend-go/v4"
	"github.com/stretchr/testify/require"
)

type brevoRoundTripFunc func(*http.Request) (*http.Response, error)

func (f brevoRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func brevoResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func TestSendEmailBrevoPrimaryAndFallback(t *testing.T) {
	for _, tc := range []struct {
		name         string
		status       int
		fallback     bool
		wantFallback int
		wantError    bool
	}{
		{"primary accepted", 201, true, 0, false},
		{"invalid key uses backup", 401, true, 1, false},
		{"quota exhausted uses backup", 429, true, 1, false},
		{"backup disabled", 403, false, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			settings := resendFallbackSettings(1)
			settings[SettingKeyEmailProvider] = EmailProviderBrevo
			settings[SettingKeyBrevoAPIKey] = "xkeysib-test-key"
			settings[SettingKeyBrevoFrom] = "no-reply@example.com"
			settings[SettingKeyBrevoFromName] = "Starbridge AI"
			if !tc.fallback {
				settings[SettingKeyResendFallbackEnabled] = "false"
			}
			svc := NewEmailService(&settingRepoStub{values: settings}, nil)
			primaryCalls, backupCalls := 0, 0
			svc.brevoClient = &http.Client{Transport: brevoRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				primaryCalls++
				require.Equal(t, brevoEmailEndpoint, req.URL.String())
				require.Equal(t, http.MethodPost, req.Method)
				require.Equal(t, "xkeysib-test-key", req.Header.Get("api-key"))
				require.Equal(t, "application/json", req.Header.Get("Content-Type"))
				payload, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				require.JSONEq(t, `{"sender":{"email":"no-reply@example.com","name":"Starbridge AI"},"to":[{"email":"user@example.com"}],"subject":"验证码","htmlContent":"<p>123456</p>"}`, string(payload))
				return brevoResponse(tc.status, `{"messageId":"<test@example.com>"}`), nil
			})}
			svc.resendSend = func(_ context.Context, _ string, email *resend.SendEmailRequest) error {
				backupCalls++
				require.Equal(t, []string{"user@example.com"}, email.To)
				require.Equal(t, "验证码", email.Subject)
				require.Equal(t, "<p>123456</p>", email.Html)
				return nil
			}
			err := svc.SendEmail(context.Background(), "user@example.com", "验证码", "<p>123456</p>")
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, 1, primaryCalls)
			require.Equal(t, tc.wantFallback, backupCalls)
		})
	}
}

func TestBrevoRejectsInvalidResponsesWithoutLeakingSecrets(t *testing.T) {
	config := &BrevoConfig{APIKey: "xkeysib-private-test", From: "no-reply@example.com"}
	for _, tc := range []struct {
		status int
		body   string
	}{
		{401, `{"message":"xkeysib-private-test"}`},
		{201, `{}`},
		{201, `not-json`},
		{302, "redirect"},
	} {
		svc := &EmailService{brevoClient: &http.Client{Transport: brevoRoundTripFunc(func(*http.Request) (*http.Response, error) {
			return brevoResponse(tc.status, tc.body), nil
		})}}
		err := svc.SendEmailWithBrevoConfig(context.Background(), config, "user@example.com", "test", "<p>test</p>")
		require.Error(t, err)
		require.NotContains(t, err.Error(), config.APIKey)
	}
}

func TestBrevoCanceledRequestDoesNotInvokeFallback(t *testing.T) {
	settings := resendFallbackSettings(1)
	settings[SettingKeyEmailProvider] = EmailProviderBrevo
	settings[SettingKeyBrevoAPIKey] = "xkeysib-test"
	settings[SettingKeyBrevoFrom] = "no-reply@example.com"
	svc := NewEmailService(&settingRepoStub{values: settings}, nil)
	svc.brevoClient = &http.Client{Transport: brevoRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, req.Context().Err()
	})}
	svc.resendSend = func(context.Context, string, *resend.SendEmailRequest) error {
		t.Fatal("canceled request must not send through backup")
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, svc.SendEmail(ctx, "user@example.com", "test", "body"), context.Canceled)
}

func TestBrevoConfigValidation(t *testing.T) {
	for _, config := range []*BrevoConfig{
		nil,
		{APIKey: "", From: "no-reply@example.com"},
		{APIKey: "re_wrong_provider", From: "no-reply@example.com"},
		{APIKey: "xsmtpsib-wrong-type", From: "no-reply@example.com"},
		{APIKey: "xkeysib-test", From: "Sender <no-reply@example.com>"},
	} {
		require.Error(t, ValidateBrevoConfig(config))
	}
	require.NoError(t, ValidateBrevoConfig(&BrevoConfig{APIKey: "xkeysib-test", From: "no-reply@example.com"}))
}

func TestBrevoRequestHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc := &EmailService{brevoClient: &http.Client{Transport: brevoRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		cancel()
		<-req.Context().Done()
		return nil, req.Context().Err()
	})}}
	err := svc.SendEmailWithBrevoConfig(ctx, &BrevoConfig{APIKey: "xkeysib-test", From: "no-reply@example.com"}, "user@example.com", "test", "body")
	require.ErrorIs(t, err, context.Canceled)
	require.Positive(t, brevoHTTPClient.Timeout)
	require.ErrorIs(t, brevoHTTPClient.CheckRedirect(nil, nil), http.ErrUseLastResponse)
}

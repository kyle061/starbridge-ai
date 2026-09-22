//go:build unit

package service

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/resend/resend-go/v4"
	"github.com/stretchr/testify/require"
)

func resendFallbackSettings(port int) map[string]string {
	return map[string]string{
		SettingKeySMTPHost:              "127.0.0.1",
		SettingKeySMTPPort:              strconv.Itoa(port),
		SettingKeySMTPUsername:          "user",
		SettingKeySMTPPassword:          "pass",
		SettingKeySMTPFrom:              "primary@example.com",
		SettingKeySMTPFromName:          "Primary",
		SettingKeySMTPUseTLS:            "false",
		SettingKeyResendFallbackEnabled: "true",
		SettingKeyResendAPIKey:          "re_real_test_key",
		SettingKeyResendFrom:            "no-reply@mail.example.com",
		SettingKeyResendFromName:        "Backup Sender",
	}
}

func TestSendEmailSMTPSuccessDoesNotInvokeResend(t *testing.T) {
	_, port := startFakeSMTPServer(t, false, false)
	svc := NewEmailService(&settingRepoStub{values: resendFallbackSettings(port)}, nil)
	resendCalls := 0
	svc.resendSend = func(context.Context, string, *resend.SendEmailRequest) error {
		resendCalls++
		return nil
	}

	require.NoError(t, svc.SendEmail(context.Background(), "user@example.com", "subject", "<p>body</p>"))
	require.Zero(t, resendCalls)
}

func TestSendEmailSMTPFailureUsesResendFallback(t *testing.T) {
	svc := NewEmailService(&settingRepoStub{values: resendFallbackSettings(1)}, nil)
	resendCalls := 0
	svc.resendSend = func(_ context.Context, apiKey string, params *resend.SendEmailRequest) error {
		resendCalls++
		require.Equal(t, "re_real_test_key", apiKey)
		require.Equal(t, `"Backup Sender" <no-reply@mail.example.com>`, params.From)
		require.Equal(t, []string{"user@example.com"}, params.To)
		require.Equal(t, "subject", params.Subject)
		require.Equal(t, "<p>body</p>", params.Html)
		return nil
	}

	require.NoError(t, svc.SendEmail(context.Background(), "user@example.com", "subject", "<p>body</p>"))
	require.Equal(t, 1, resendCalls)
}

func TestSendEmailReportsPrimaryAndFallbackFailures(t *testing.T) {
	svc := NewEmailService(&settingRepoStub{values: resendFallbackSettings(1)}, nil)
	svc.resendSend = func(context.Context, string, *resend.SendEmailRequest) error {
		return errors.New("resend unavailable")
	}

	err := svc.SendEmail(context.Background(), "user@example.com", "subject", "<p>body</p>")
	require.Error(t, err)
	require.Contains(t, err.Error(), "primary SMTP failed")
	require.Contains(t, err.Error(), "Resend fallback failed")
	require.Contains(t, err.Error(), "resend unavailable")
}

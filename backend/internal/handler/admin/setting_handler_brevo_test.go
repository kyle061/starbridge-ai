//go:build unit

package admin

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUpdateSettingsBrevoPersistsAndPreservesSecret(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{service.SettingKeySMTPHost: "smtp.qq.com"})
	rec := doUpdateSettings(t, h, map[string]any{
		"email_provider":   "brevo",
		"brevo_api_key":    " xkeysib-private-test ",
		"brevo_from_email": "no-reply@example.com",
		"brevo_from_name":  "Starbridge AI",
	}, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, "brevo", repo.values[service.SettingKeyEmailProvider])
	require.Equal(t, "xkeysib-private-test", repo.values[service.SettingKeyBrevoAPIKey])
	require.Equal(t, "no-reply@example.com", repo.values[service.SettingKeyBrevoFrom])
	require.Equal(t, "smtp.qq.com", repo.values[service.SettingKeySMTPHost])
	require.NotContains(t, rec.Body.String(), "xkeysib-private-test")
	var result struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	require.Equal(t, true, result.Data["brevo_api_key_configured"])
	require.NotContains(t, result.Data, "brevo_api_key")

	for _, body := range []map[string]any{{"brevo_api_key": " "}, {"site_name": "Starbridge AI"}} {
		rec = doUpdateSettings(t, h, body, nil)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, "brevo", repo.values[service.SettingKeyEmailProvider])
		require.Equal(t, "xkeysib-private-test", repo.values[service.SettingKeyBrevoAPIKey])
		require.Equal(t, "Starbridge AI", repo.values[service.SettingKeyBrevoFromName])
	}

	rec = doUpdateSettings(t, h, map[string]any{"email_provider": "smtp"}, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "smtp", repo.values[service.SettingKeyEmailProvider])
	require.Equal(t, "xkeysib-private-test", repo.values[service.SettingKeyBrevoAPIKey])
}

func TestUpdateSettingsBrevoRejectsInvalidConfiguration(t *testing.T) {
	for _, body := range []map[string]any{
		{"email_provider": "unknown"},
		{"email_provider": "brevo", "brevo_from_email": "no-reply@example.com"},
		{"email_provider": "brevo", "brevo_api_key": "re_wrong", "brevo_from_email": "no-reply@example.com"},
		{"email_provider": "brevo", "brevo_api_key": "xsmtpsib-wrong", "brevo_from_email": "no-reply@example.com"},
		{"email_provider": "brevo", "brevo_api_key": "xkeysib-test", "brevo_from_email": "invalid"},
	} {
		h, repo := newStepUpSwitchTestHandler(t, map[string]string{service.SettingKeyEmailProvider: "smtp"})
		rec := doUpdateSettings(t, h, body, nil)
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "smtp", repo.values[service.SettingKeyEmailProvider])
	}
}

func TestUpdateSettingsBrevoValidatesStoredProviderOnPartialUpdates(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeyEmailProvider: "brevo",
		service.SettingKeyBrevoAPIKey:   "xkeysib-stored",
		service.SettingKeyBrevoFrom:     "no-reply@example.com",
	})
	rec := doUpdateSettings(t, h, map[string]any{"brevo_from_email": ""}, nil)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "no-reply@example.com", repo.values[service.SettingKeyBrevoFrom])
}

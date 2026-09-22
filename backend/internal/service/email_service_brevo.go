package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"
)

const (
	EmailProviderSMTP  = "smtp"
	EmailProviderBrevo = "brevo"
	brevoEmailEndpoint = "https://api.brevo.com/v3/smtp/email"
)

var brevoHTTPClient = &http.Client{
	Timeout: 15 * time.Second,
	// Never forward the API key to a redirect destination.
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

type BrevoConfig struct {
	APIKey   string
	From     string
	FromName string
}

func emailProviderOrDefault(provider string) string {
	if provider = strings.ToLower(strings.TrimSpace(provider)); provider == "" {
		return EmailProviderSMTP
	}
	return provider
}

func ValidateBrevoConfig(config *BrevoConfig) error {
	if config == nil || strings.TrimSpace(config.APIKey) == "" {
		return fmt.Errorf("Brevo API key is required")
	}
	key := strings.TrimSpace(config.APIKey)
	if strings.HasPrefix(key, "re_") || strings.HasPrefix(key, "xsmtpsib-") || strings.ContainsAny(key, "\r\n\t ") {
		return fmt.Errorf("Use a Brevo API key, not a Resend API key or SMTP key")
	}
	from := strings.TrimSpace(config.From)
	address, err := mail.ParseAddress(from)
	if err != nil || address.Address != from {
		return fmt.Errorf("A valid Brevo sender email is required")
	}
	return nil
}

func (s *EmailService) GetBrevoConfig(ctx context.Context) (*BrevoConfig, error) {
	settings, err := s.settingRepo.GetMultiple(ctx, []string{SettingKeyBrevoAPIKey, SettingKeyBrevoFrom, SettingKeyBrevoFromName})
	if err != nil {
		return nil, fmt.Errorf("get Brevo settings: %w", err)
	}
	return &BrevoConfig{
		APIKey:   strings.TrimSpace(settings[SettingKeyBrevoAPIKey]),
		From:     strings.TrimSpace(settings[SettingKeyBrevoFrom]),
		FromName: strings.TrimSpace(settings[SettingKeyBrevoFromName]),
	}, nil
}

func (s *EmailService) sendPrimaryEmail(ctx context.Context, to, subject, body string) (string, error) {
	settings, err := s.settingRepo.GetMultiple(ctx, []string{SettingKeyEmailProvider})
	if err != nil {
		return "email", fmt.Errorf("get primary email provider: %w", err)
	}
	switch provider := emailProviderOrDefault(settings[SettingKeyEmailProvider]); provider {
	case EmailProviderSMTP:
		config, err := s.GetSMTPConfig(ctx)
		if err != nil {
			return "SMTP", err
		}
		return "SMTP", s.SendEmailWithConfig(config, to, subject, body)
	case EmailProviderBrevo:
		config, err := s.GetBrevoConfig(ctx)
		if err != nil {
			return "Brevo", err
		}
		return "Brevo", s.SendEmailWithBrevoConfig(ctx, config, to, subject, body)
	default:
		return "email", fmt.Errorf("unsupported primary email provider")
	}
}

// SendEmailWithBrevoConfig tests or sends through Brevo without invoking fallback.
func (s *EmailService) SendEmailWithBrevoConfig(ctx context.Context, config *BrevoConfig, to, subject, body string) error {
	if err := ValidateBrevoConfig(config); err != nil {
		return err
	}
	to = strings.TrimSpace(to)
	address, err := mail.ParseAddress(to)
	if err != nil || address.Address != to {
		return fmt.Errorf("a valid recipient email is required")
	}
	type contact struct {
		Email string `json:"email"`
		Name  string `json:"name,omitempty"`
	}
	payload, err := json.Marshal(struct {
		Sender  contact   `json:"sender"`
		To      []contact `json:"to"`
		Subject string    `json:"subject"`
		HTML    string    `json:"htmlContent"`
	}{contact{strings.TrimSpace(config.From), strings.TrimSpace(config.FromName)}, []contact{{Email: to}}, subject, body})
	if err != nil {
		return fmt.Errorf("encode Brevo email: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, brevoEmailEndpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create Brevo request: %w", err)
	}
	req.Header.Set("api-key", strings.TrimSpace(config.APIKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	client := s.brevoClient
	if client == nil {
		client = brevoHTTPClient
	}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("Brevo request: %w", ctx.Err())
		}
		return fmt.Errorf("Brevo request failed; check connectivity or timeout")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		// Do not expose the remote body: it can echo credentials or email content.
		reason := "check Brevo transactional email logs"
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			reason = "API key is invalid or expired"
		case http.StatusForbidden:
			reason = "check account activation, phone verification and authorized IPs in Brevo"
		case http.StatusBadRequest:
			reason = "check the verified sender, domain and account sending status in Brevo"
		case http.StatusTooManyRequests:
			reason = "sending limit reached; check the Brevo quota"
		}
		return fmt.Errorf("Brevo HTTP %d: %s", resp.StatusCode, reason)
	}
	var result struct {
		MessageID string `json:"messageId"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&result); err != nil || result.MessageID == "" {
		return fmt.Errorf("Brevo returned no message ID; check transactional logs before retrying")
	}
	return nil
}

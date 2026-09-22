package service

import (
	"context"
	"encoding/json"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Only configured integration credentials may be revealed. Never accept arbitrary
// setting names (which could expose signing/encryption keys or whole documents).
var revealableSettingSecrets = map[string]struct{}{
	SettingKeySMTPPassword: {}, SettingKeyBrevoAPIKey: {}, SettingKeyResendAPIKey: {},
	SettingKeyTurnstileSecretKey: {}, SettingKeyTencentCaptchaAppSecretKey: {},
	SettingKeyTencentCaptchaCloudSecretID: {}, SettingKeyTencentCaptchaCloudSecretKey: {},
	SettingKeyAliyunCaptchaAccessKeySecret: {}, SettingKeyLinuxDoConnectClientSecret: {},
	SettingKeyDingTalkConnectClientSecret: {}, SettingKeyWeChatConnectAppSecret: {},
	SettingKeyWeChatConnectOpenAppSecret: {}, SettingKeyWeChatConnectMPAppSecret: {},
	SettingKeyWeChatConnectMobileAppSecret: {}, SettingKeyOIDCConnectClientSecret: {},
	SettingKeyGitHubOAuthClientSecret: {}, SettingKeyGoogleOAuthClientSecret: {},
	SettingKeyAdminAPIKey: {},
}

var ErrSecretNotRevealable = infraerrors.BadRequest("SECRET_NOT_REVEALABLE", "this credential cannot be revealed")

// GetStoredSecret is only for the password-gated admin endpoint.
func (s *SettingService) GetStoredSecret(ctx context.Context, key, providerType string) (string, error) {
	if key == "web_search_api_key" {
		if !validProviderTypes[providerType] {
			return "", ErrSecretNotRevealable
		}
		raw, err := s.settingRepo.GetValue(ctx, SettingKeyWebSearchEmulationConfig)
		if err != nil {
			return "", err
		}
		var cfg WebSearchEmulationConfig
		if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
			return "", ErrSecretNotRevealable
		}
		for _, provider := range cfg.Providers {
			if provider.Type == providerType {
				return provider.APIKey, nil
			}
		}
		return "", ErrSettingNotFound
	}
	if _, ok := revealableSettingSecrets[key]; !ok {
		return "", ErrSecretNotRevealable
	}
	return s.settingRepo.GetValue(ctx, key)
}

// VerifyAdminPassword always reloads the acting user's password and role.
func (s *UserService) VerifyAdminPassword(ctx context.Context, userID int64, password string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil || !user.IsAdmin() || !user.IsActive() {
		return ErrInsufficientPerms
	}
	if password == "" || !user.CheckPassword(password) {
		return ErrPasswordIncorrect
	}
	return nil
}

func (s *PaymentConfigService) GetProviderSecret(ctx context.Context, id int64, field string) (string, error) {
	if id <= 0 {
		return "", ErrSecretNotRevealable
	}
	inst, err := s.entClient.PaymentProviderInstance.Get(ctx, id)
	if err != nil {
		return "", err
	}
	if !isSensitiveProviderConfigField(inst.ProviderKey, field) {
		return "", ErrSecretNotRevealable
	}
	cfg, err := s.decryptConfig(inst.Config)
	if err != nil {
		return "", err
	}
	return cfg[field], nil
}

func (s *BackupService) GetStoredS3Secret(ctx context.Context) (string, error) {
	cfg, err := s.loadS3Config(ctx)
	if err != nil {
		return "", err
	}
	if cfg == nil {
		return "", ErrSettingNotFound
	}
	return cfg.SecretAccessKey, nil
}

func (s *ImageStorageSettingService) GetStoredS3Secret(ctx context.Context) (string, error) {
	cfg, err := s.effectiveConfig(ctx)
	if err != nil {
		return "", err
	}
	return cfg.SecretAccessKey, nil
}

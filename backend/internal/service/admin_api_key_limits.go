package service

import (
	"context"
	"fmt"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// AdminAPIKeyLimits changes limits without overwriting live usage counters.
type AdminAPIKeyLimits struct {
	Quota       *float64 `json:"quota"`
	RateLimit5h *float64 `json:"rate_limit_5h"`
	RateLimit1d *float64 `json:"rate_limit_1d"`
	RateLimit7d *float64 `json:"rate_limit_7d"`
}

func (s *adminServiceImpl) AdminUpdateAPIKeyLimits(ctx context.Context, keyID int64, req AdminAPIKeyLimits) (*APIKey, error) {
	if err := validateUpdateAPIKeyRequest(UpdateAPIKeyRequest{
		Quota: req.Quota, RateLimit5h: req.RateLimit5h, RateLimit1d: req.RateLimit1d, RateLimit7d: req.RateLimit7d,
	}); err != nil {
		return nil, infraerrors.BadRequest("INVALID_API_KEY_LIMIT", "limits must be finite and non-negative")
	}
	key, err := s.apiKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	var fields APIKeyUpdateFields
	if req.Quota != nil {
		key.Quota = *req.Quota
		fields.Quota = true
		if key.Status == StatusAPIKeyQuotaExhausted && (key.Quota == 0 || key.Quota > key.QuotaUsed) {
			key.Status = StatusActive
			fields.Status = true
		}
	}
	if req.RateLimit5h != nil {
		key.RateLimit5h = *req.RateLimit5h
		fields.RateLimits = true
	}
	if req.RateLimit1d != nil {
		key.RateLimit1d = *req.RateLimit1d
		fields.RateLimits = true
	}
	if req.RateLimit7d != nil {
		key.RateLimit7d = *req.RateLimit7d
		fields.RateLimits = true
	}
	if err := s.apiKeyRepo.Update(ctx, key, fields); err != nil {
		return nil, fmt.Errorf("update api key limits: %w", err)
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByKey(ctx, key.Key)
	}
	return key, nil
}

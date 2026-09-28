package service

import (
	"context"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func accountSupportsGroupImageGeneration(account *Account, platform string) bool {
	if account == nil || (platform != PlatformComposite && account.Platform != platform) {
		return false
	}
	models := account.GetModelMapping()
	if len(models) == 0 {
		switch account.Platform {
		case PlatformOpenAI:
			return account.SupportsOpenAIImageCapability(OpenAIImagesCapabilityBasic)
		case PlatformGemini, PlatformAntigravity, PlatformGrok:
			return true
		default:
			return false
		}
	}
	for source, target := range models {
		for _, model := range []string{source, target} {
			if isOpenAIImageGenerationModel(model) || isImageGenerationModel(model) {
				return true
			}
		}
	}
	for _, candidate := range []string{"gpt-image-2", "gpt-image-2.5-flare", "gemini-3.1-flash-image", "grok-imagine-image"} {
		if mapped, matched := resolveRequestedModelInMapping(models, candidate); matched &&
			(isOpenAIImageGenerationModel(mapped) || isImageGenerationModel(mapped)) {
			return true
		}
	}
	return false
}

func groupHasImageAccount(accounts []*Account, platform string, oauthOnly bool) bool {
	for _, account := range accounts {
		if account != nil && (!oauthOnly || account.Type != AccountTypeAPIKey) && accountSupportsGroupImageGeneration(account, platform) {
			return true
		}
	}
	return false
}

func (s *adminServiceImpl) validateGroupImageAccounts(ctx context.Context, platform string, accountIDs []int64, oauthOnly bool) error {
	if s.accountRepo == nil || len(accountIDs) == 0 {
		return infraerrors.BadRequest("GROUP_IMAGE_MODEL_UNAVAILABLE", "Bind an account with an image-generation model before enabling images for this group")
	}
	accounts, err := s.accountRepo.GetByIDs(ctx, accountIDs)
	if err != nil {
		return err
	}
	if !groupHasImageAccount(accounts, platform, oauthOnly) {
		return infraerrors.BadRequest("GROUP_IMAGE_MODEL_UNAVAILABLE", "No bound account supports an image-generation model for this group")
	}
	return nil
}

func (s *adminServiceImpl) validateExistingGroupImageAccounts(ctx context.Context, group *Group) error {
	if s.accountRepo == nil {
		return infraerrors.BadRequest("GROUP_IMAGE_MODEL_UNAVAILABLE", "No image-capable account is bound to this group")
	}
	accounts, err := s.accountRepo.ListByGroup(ctx, group.ID)
	if err != nil {
		return err
	}
	for i := range accounts {
		if (!group.RequireOAuthOnly || accounts[i].Type != AccountTypeAPIKey) && accountSupportsGroupImageGeneration(&accounts[i], group.Platform) {
			return nil
		}
	}
	return infraerrors.BadRequest("GROUP_IMAGE_MODEL_UNAVAILABLE", "No bound account supports an image-generation model for this group")
}

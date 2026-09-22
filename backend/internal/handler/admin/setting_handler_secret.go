package admin

import (
	"errors"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *SettingHandler) SetSecretRevealDeps(backup *service.BackupService, imageStorage *service.ImageStorageSettingService) {
	h.secretBackupService = backup
	h.secretImageStorageService = imageStorage
}

// RevealSecret returns exactly one saved credential after fresh password verification.
// Neither the password nor the value belongs in audit logs or normal settings DTOs.
func (h *SettingHandler) RevealSecret(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	middleware.SetAuditAction(c, "admin.settings.reveal_secret")
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	role, _ := middleware.GetUserRoleFromContext(c)
	if !ok || subject.UserID <= 0 || role != service.RoleAdmin || c.GetString("auth_method") != "jwt" {
		response.Forbidden(c, "Sign in as an administrator to view credentials")
		return
	}
	var req struct {
		Key          string `json:"key" binding:"required,max=100"`
		Password     string `json:"password" binding:"required,max=1024"`
		ProviderID   int64  `json:"provider_id"`
		ProviderType string `json:"provider_type" binding:"max=32"`
		Field        string `json:"field" binding:"max=100"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Credential name and administrator password are required")
		return
	}
	if h.userService == nil || h.settingService == nil {
		response.InternalError(c, "Credential verification is unavailable")
		return
	}
	if err := h.userService.VerifyAdminPassword(c.Request.Context(), subject.UserID, req.Password); err != nil {
		if errors.Is(err, service.ErrPasswordIncorrect) {
			response.ErrorFrom(c, err)
		} else {
			response.Forbidden(c, "Administrator verification failed")
		}
		return
	}
	var value string
	var err error
	switch req.Key {
	case "payment_provider_secret":
		if h.paymentConfigService == nil {
			err = service.ErrSecretNotRevealable
		} else {
			value, err = h.paymentConfigService.GetProviderSecret(c.Request.Context(), req.ProviderID, req.Field)
		}
	case "backup_s3_secret":
		if h.secretBackupService == nil {
			err = service.ErrSecretNotRevealable
		} else {
			value, err = h.secretBackupService.GetStoredS3Secret(c.Request.Context())
		}
	case "image_storage_s3_secret":
		if h.secretImageStorageService == nil {
			err = service.ErrSecretNotRevealable
		} else {
			value, err = h.secretImageStorageService.GetStoredS3Secret(c.Request.Context())
		}
	default:
		value, err = h.settingService.GetStoredSecret(c.Request.Context(), req.Key, req.ProviderType)
	}
	if err != nil || value == "" {
		// Do not echo storage, decryption or provider errors: they can contain secrets.
		response.BadRequest(c, "No saved credential is available for this field")
		return
	}
	response.Success(c, gin.H{"value": value})
}

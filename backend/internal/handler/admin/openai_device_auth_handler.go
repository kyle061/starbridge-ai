package admin

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

func deviceAuthOwner(c *gin.Context) (int64, bool) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "Sign in as an administrator to bind an account")
		return 0, false
	}
	c.Header("Cache-Control", "no-store")
	return subject.UserID, true
}
func (h *OpenAIOAuthHandler) StartDeviceAuth(c *gin.Context) {
	owner, ok := deviceAuthOwner(c)
	if !ok {
		return
	}
	var req struct {
		ProxyID *int64 `json:"proxy_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
	defer cancel()
	result, err := h.openaiOAuthService.StartDeviceAuth(ctx, owner, req.ProxyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
func (h *OpenAIOAuthHandler) PollDeviceAuth(c *gin.Context) {
	owner, ok := deviceAuthOwner(c)
	if !ok {
		return
	}
	var req struct {
		SessionID string `json:"session_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
	defer cancel()
	result, err := h.openaiOAuthService.PollDeviceAuth(ctx, req.SessionID, owner)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
func (h *OpenAIOAuthHandler) CancelDeviceAuth(c *gin.Context) {
	owner, ok := deviceAuthOwner(c)
	if !ok {
		return
	}
	var req struct {
		SessionID string `json:"session_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	h.openaiOAuthService.CancelDeviceAuth(req.SessionID, owner)
	response.Success(c, gin.H{"cancelled": true})
}

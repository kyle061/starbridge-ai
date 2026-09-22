package admin

import (
	"html"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type SendTestBrevoEmailRequest struct {
	Email         string  `json:"email" binding:"required,email"`
	BrevoAPIKey   string  `json:"brevo_api_key"`
	BrevoFrom     *string `json:"brevo_from_email"`
	BrevoFromName *string `json:"brevo_from_name"`
}

// SendTestBrevoEmail sends directly through Brevo, even before switching primary.
func (h *SettingHandler) SendTestBrevoEmail(c *gin.Context) {
	var req SendTestBrevoEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid test email request")
		return
	}
	config, err := h.emailService.GetBrevoConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if key := strings.TrimSpace(req.BrevoAPIKey); key != "" {
		config.APIKey = key
	}
	if req.BrevoFrom != nil {
		config.From = strings.TrimSpace(*req.BrevoFrom)
	}
	if req.BrevoFromName != nil {
		config.FromName = strings.TrimSpace(*req.BrevoFromName)
	}
	if err := service.ValidateBrevoConfig(config); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	siteName := h.settingService.GetSiteName(c.Request.Context())
	subject := "[" + siteName + "] Brevo 邮件测试"
	body := "<html><head><meta charset=\"UTF-8\"></head><body><h1>" + html.EscapeString(siteName) + "</h1><p>这是一封通过 Brevo 发送的测试邮件，用于验证邮件配置。</p></body></html>"
	if err := h.emailService.SendEmailWithBrevoConfig(c.Request.Context(), config, req.Email, subject, body); err != nil {
		response.BadRequest(c, "Failed to send Brevo test email: "+err.Error())
		return
	}
	response.Success(c, gin.H{"message": "Brevo 已接受测试邮件，请检查收件箱和垃圾邮件文件夹"})
}

package middleware

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSecretRevealLimitCannotBeDisabledOrBypassedByAdmin(t *testing.T) {
	p := &PanelRateLimiter{limiter: &fakePanelAllower{}}
	router := newPanelTestRouter(p.SecretReveal(), &panelTestIdentity{userID: 7, role: service.RoleAdmin})
	for i := 0; i < 5; i++ {
		require.Equal(t, 200, performPanelRequest(router, "127.0.0.1:1234").Code)
	}
	rec := performPanelRequest(router, "127.0.0.1:4321")
	require.Equal(t, 429, rec.Code)
	require.NotEmpty(t, rec.Header().Get("Retry-After"))
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	other := newPanelTestRouter(p.SecretReveal(), &panelTestIdentity{userID: 8, role: service.RoleAdmin})
	require.Equal(t, 200, performPanelRequest(other, "127.0.0.1:1234").Code)
	for _, limiter := range []*PanelRateLimiter{nil, {}, {limiter: &fakePanelAllower{err: errors.New("redis unavailable")}}} {
		r := newPanelTestRouter(limiter.SecretReveal(), &panelTestIdentity{userID: 7, role: service.RoleAdmin})
		require.Equal(t, 503, performPanelRequest(r, "127.0.0.1:1234").Code)
	}
}

func TestSecretRevealAuditOmitsPasswordAndValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &auditCaptureRepository{}
	svc := service.NewAuditLogService(repo, nil)
	svc.Start()
	router := gin.New()
	router.Use(gin.HandlerFunc(NewAuditLogMiddleware(svc)))
	router.POST("/api/v1/admin/settings/reveal-secret", func(c *gin.Context) {
		SetAuditAction(c, "admin.settings.reveal_secret")
		c.JSON(200, gin.H{"value": "saved-secret-canary"})
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/settings/reveal-secret", bytes.NewBufferString(`{"password":"password-canary","key":"resend_api_key"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(httptest.NewRecorder(), req)
	svc.Stop()
	require.Len(t, repo.logs, 1)
	require.NotContains(t, repo.logs[0].RequestBody, "password-canary")
	require.NotContains(t, repo.logs[0].RequestBody, "saved-secret-canary")
}

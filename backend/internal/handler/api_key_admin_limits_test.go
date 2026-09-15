//go:build unit

package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUserAPIKeyEndpointsRejectAdminLimits(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})
	})
	h := NewAPIKeyHandler(nil) // Forbidden requests must never reach the service.
	router.POST("/keys", h.Create)
	router.PUT("/keys/:id", h.Update)
	for _, field := range []string{`"quota":0`, `"rate_limit_5h":0`, `"rate_limit_1d":25`, `"rate_limit_7d":0`, `"expires_in_days":30`} {
		t.Run("create/"+field, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/keys", bytes.NewBufferString(`{"name":"test",`+field+`}`))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(rec, req)
			require.Equal(t, http.StatusForbidden, rec.Code)
		})
	}
	for _, field := range []string{`"quota":0`, `"rate_limit_5h":0`, `"rate_limit_1d":0`, `"rate_limit_7d":0`, `"expires_at":""`, `"reset_quota":true`, `"reset_rate_limit_usage":true`} {
		t.Run("update/"+field, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, "/keys/1", bytes.NewBufferString(`{`+field+`}`))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(rec, req)
			require.Equal(t, http.StatusForbidden, rec.Code)
		})
	}
}

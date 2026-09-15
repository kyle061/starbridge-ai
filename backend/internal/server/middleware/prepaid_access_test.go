//go:build unit

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type prepaidAuthUserRepo struct {
	service.UserRepository
	balance service.PrepaidBalance
}

func (r *prepaidAuthUserRepo) GetPrepaidBalance(context.Context, int64) (*service.PrepaidBalance, error) {
	return &r.balance, nil
}

func TestPrepaidAuthPausesEveryKeyAndResumesOriginalCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, google := range []bool{false, true} {
		t.Run(map[bool]string{false: "openai", true: "google"}[google], func(t *testing.T) {
			users := &prepaidAuthUserRepo{balance: service.PrepaidBalance{Balance: 20, HasPurchased: true}}
			repo := &stubApiKeyRepo{getByKey: func(_ context.Context, key string) (*service.APIKey, error) {
				return &service.APIKey{ID: 1, UserID: 7, Key: key, Status: service.StatusActive,
					Group: &service.Group{Status: service.StatusActive},
					User:  &service.User{ID: 7, Role: service.RoleUser, Status: service.StatusActive, Balance: 999}}, nil
			}}
			cfg := &config.Config{RunMode: config.RunModeStandard, Billing: config.BillingConfig{RequireBalancePurchase: true}}
			svc := service.NewAPIKeyService(repo, users, nil, nil, nil, nil, cfg)
			r := gin.New()
			if google {
				r.Use(APIKeyAuthGoogle(svc, cfg))
			} else {
				r.Use(gin.HandlerFunc(NewAPIKeyAuthMiddleware(svc, nil, cfg)))
			}
			r.POST("/test", func(c *gin.Context) { c.Status(http.StatusOK) })
			for _, state := range []struct {
				balance float64
				paid    bool
				status  int
			}{
				{20, false, http.StatusForbidden}, {20, true, http.StatusOK},
				{0, true, http.StatusForbidden}, {-1, true, http.StatusForbidden},
				{19, true, http.StatusOK},
			} {
				users.balance = service.PrepaidBalance{Balance: state.balance, HasPurchased: state.paid}
				for _, credential := range []string{"sk-first", "sk-second"} {
					req := httptest.NewRequest(http.MethodPost, "/test", nil)
					req.Header.Set("Authorization", "Bearer "+credential)
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)
					require.Equal(t, state.status, w.Code, w.Body.String())
				}
			}
		})
	}
}

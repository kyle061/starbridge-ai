package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type secretUserRepo struct {
	service.UserRepository
	user        *service.User
	requestedID int64
}

func (r *secretUserRepo) GetByID(_ context.Context, id int64) (*service.User, error) {
	r.requestedID = id
	return r.user, nil
}

func TestRevealSecretRequiresCurrentAdminPassword(t *testing.T) {
	user := &service.User{ID: 7, Role: service.RoleAdmin, Status: service.StatusActive}
	require.NoError(t, user.SetPassword("correct-admin-password"))
	for _, tc := range []struct {
		name, password, role, method, key string
		identity                          bool
		status                            int
	}{
		{"success", "correct-admin-password", "admin", "jwt", "resend_api_key", true, 200},
		{"wrong password", "wrong", "admin", "jwt", "resend_api_key", true, 400},
		{"missing password", "", "admin", "jwt", "resend_api_key", true, 400},
		{"missing identity", "correct-admin-password", "admin", "jwt", "resend_api_key", false, 403},
		{"non admin", "correct-admin-password", "user", "jwt", "resend_api_key", true, 403},
		{"machine credential", "correct-admin-password", "admin", "admin_api_key", "resend_api_key", true, 403},
		{"unknown setting", "correct-admin-password", "admin", "jwt", "jwt_signing_key", true, 400},
		{"missing secret", "correct-admin-password", "admin", "jwt", "brevo_api_key", true, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, _ := newStepUpSwitchTestHandler(t, map[string]string{"resend_api_key": "re_test_saved_credential", "jwt_signing_key": "never-reveal"})
			repo := &secretUserRepo{user: user}
			h.SetStepUpDeps(nil, service.NewUserService(repo, nil, nil, nil))
			body, err := json.Marshal(map[string]string{"key": tc.key, "password": tc.password})
			require.NoError(t, err)
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/settings/reveal-secret", bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")
			if tc.identity {
				c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 7})
			}
			c.Set(string(middleware.ContextKeyUserRole), tc.role)
			c.Set("auth_method", tc.method)
			h.RevealSecret(c)
			require.Equal(t, tc.status, rec.Code, rec.Body.String())
			require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
			require.NotContains(t, rec.Body.String(), "correct-admin-password")
			require.NotContains(t, rec.Body.String(), "never-reveal")
			if tc.status == 200 {
				require.Contains(t, rec.Body.String(), "re_test_saved_credential")
				require.Equal(t, int64(7), repo.requestedID)
			} else {
				require.NotContains(t, rec.Body.String(), "re_test_saved_credential")
			}
		})
	}
}

func TestRevealSecretRechecksRoleStatusAndPassword(t *testing.T) {
	ctx := context.Background()
	user := &service.User{ID: 7, Role: service.RoleAdmin, Status: service.StatusActive}
	require.NoError(t, user.SetPassword("first-password"))
	svc := service.NewUserService(&secretUserRepo{user: user}, nil, nil, nil)
	require.NoError(t, svc.VerifyAdminPassword(ctx, 7, "first-password"))
	require.NoError(t, user.SetPassword("changed-password"))
	require.Error(t, svc.VerifyAdminPassword(ctx, 7, "first-password"))
	user.Role = service.RoleUser
	require.Error(t, svc.VerifyAdminPassword(ctx, 7, "changed-password"))
	user.Role, user.Status = service.RoleAdmin, "disabled"
	require.Error(t, svc.VerifyAdminPassword(ctx, 7, "changed-password"))
}

func TestRevealSecretAllowlistAndWebSearchSelection(t *testing.T) {
	stored := map[string]string{
		"smtp_password": "smtp-secret", "brevo_api_key": "brevo-secret", "resend_api_key": "resend-secret",
		"admin_api_key": "admin-secret", "turnstile_secret_key": "turnstile-secret",
		"web_search_emulation_config": `{"providers":[{"type":"brave","api_key":"brave-secret"},{"type":"tavily","api_key":"tavily-secret"}]}`,
	}
	h, _ := newStepUpSwitchTestHandler(t, stored)
	for key, want := range stored {
		if key == "web_search_emulation_config" {
			continue
		}
		got, err := h.settingService.GetStoredSecret(context.Background(), key, "")
		require.NoError(t, err)
		require.Equal(t, want, got)
	}
	for _, provider := range []string{"brave", "tavily"} {
		got, err := h.settingService.GetStoredSecret(context.Background(), "web_search_api_key", provider)
		require.NoError(t, err)
		require.Equal(t, provider+"-secret", got)
	}
	for _, key := range []string{"web_search_emulation_config", "totp_encryption_key", "site_name", "web_search_api_key"} {
		got, err := h.settingService.GetStoredSecret(context.Background(), key, "unknown")
		require.Error(t, err)
		require.Empty(t, got)
	}
}

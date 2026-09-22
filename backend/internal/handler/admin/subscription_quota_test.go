package admin

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type quotaAssignmentGroupRepo struct{ service.GroupRepository }

func (quotaAssignmentGroupRepo) GetByID(context.Context, int64) (*service.Group, error) {
	return &service.Group{ID: 9, SubscriptionType: service.SubscriptionTypeSubscription}, nil
}

type quotaAssignmentSubRepo struct {
	service.UserSubscriptionRepository
	sub *service.UserSubscription
}

func (r *quotaAssignmentSubRepo) ExistsByUserIDAndGroupID(context.Context, int64, int64) (bool, error) {
	return false, nil
}
func (r *quotaAssignmentSubRepo) Create(_ context.Context, sub *service.UserSubscription) error {
	sub.ID = 5
	cp := *sub
	r.sub = &cp
	return nil
}
func (r *quotaAssignmentSubRepo) GetByID(context.Context, int64) (*service.UserSubscription, error) {
	cp := *r.sub
	return &cp, nil
}
func (r *quotaAssignmentSubRepo) GetByIDForUpdate(ctx context.Context, id int64) (*service.UserSubscription, error) {
	return r.GetByID(ctx, id)
}
func (r *quotaAssignmentSubRepo) Update(_ context.Context, sub *service.UserSubscription) error {
	cp := *sub
	r.sub = &cp
	return nil
}

func TestSubscriptionQuotaHandlerPassesEntitlementsAndPreservesUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &quotaAssignmentSubRepo{}
	svc := service.NewSubscriptionService(quotaAssignmentGroupRepo{}, repo, nil, nil, nil)
	t.Cleanup(svc.Stop)
	h := NewSubscriptionHandler(svc)
	router := gin.New()
	router.POST("/assign", h.Assign)
	router.PUT("/:id/quota", h.SetQuota)
	router.PUT("/:id/multiplier", h.SetMultiplier)
	request := httptest.NewRequest(http.MethodPost, "/assign", bytes.NewBufferString(`{"user_id":1,"group_id":9,"validity_days":1,"quota_usd":50,"usage_multiplier":12,"plan_name":"Daily"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Equal(t, float64(50), repo.sub.QuotaUSD)
	require.Equal(t, float64(12), repo.sub.UsageMultiplier)
	require.Equal(t, "Daily", repo.sub.PlanName)
	repo.sub.QuotaUsedUSD = 0.000024
	request = httptest.NewRequest(http.MethodPut, "/5/multiplier", bytes.NewBufferString(`{"usage_multiplier":18}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Equal(t, float64(18), repo.sub.UsageMultiplier)
	require.Equal(t, float64(50), repo.sub.QuotaUSD)
	require.Equal(t, 0.000024, repo.sub.QuotaUsedUSD)
	for _, body := range []string{`{"quota_usd":0}`, `{"quota_usd":-1}`, `{}`, `{"quota_usd":100,"usage_multiplier":0}`} {
		request = httptest.NewRequest(http.MethodPut, "/5/quota", bytes.NewBufferString(body))
		request.Header.Set("Content-Type", "application/json")
		response = httptest.NewRecorder()
		router.ServeHTTP(response, request)
		require.Equal(t, http.StatusBadRequest, response.Code, body)
	}
	request = httptest.NewRequest(http.MethodPut, "/5/quota", bytes.NewBufferString(`{"quota_usd":100}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Equal(t, float64(100), repo.sub.QuotaUSD)
	require.Equal(t, 0.000024, repo.sub.QuotaUsedUSD)
	require.Equal(t, float64(18), repo.sub.UsageMultiplier)
}

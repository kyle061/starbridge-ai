package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type latencyRuleCaptureRepo struct {
	service.OpsRepository
	saved *service.OpsAlertRule
}

func (r *latencyRuleCaptureRepo) CreateAlertRule(_ context.Context, v *service.OpsAlertRule) (*service.OpsAlertRule, error) {
	v.ID = 3
	r.saved = v
	return v, nil
}
func (r *latencyRuleCaptureRepo) UpdateAlertRule(_ context.Context, v *service.OpsAlertRule) (*service.OpsAlertRule, error) {
	r.saved = v
	return v, nil
}

func TestOpsLatencyAlertHandlerCreateAndUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, metric := range []string{"p95_latency_ms", "p99_latency_ms"} {
		for _, method := range []string{http.MethodPost, http.MethodPut} {
			t.Run(metric+"/"+method, func(t *testing.T) {
				repo := &latencyRuleCaptureRepo{}
				svc := service.NewOpsService(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
				h := NewOpsHandler(svc)
				router := gin.New()
				router.POST("/rules", h.CreateAlertRule)
				router.PUT("/rules/:id", h.UpdateAlertRule)
				path := "/rules"
				if method == http.MethodPut {
					path += "/3"
				}
				for _, threshold := range []float64{2000, 0, -1} {
					repo.saved = nil
					body, err := json.Marshal(map[string]any{"name": "Latency", "metric_type": metric, "operator": ">", "threshold": threshold, "severity": "P2", "window_minutes": 5, "sustained_minutes": 10, "cooldown_minutes": 30, "enabled": true, "notify_email": true, "filters": map[string]any{"platform": "openai", "group_id": 7}})
					require.NoError(t, err)
					w := httptest.NewRecorder()
					req := httptest.NewRequest(method, path, bytes.NewReader(body))
					req.Header.Set("Content-Type", "application/json")
					router.ServeHTTP(w, req)
					if threshold < 0 {
						require.Equal(t, 400, w.Code)
						require.Nil(t, repo.saved)
						continue
					}
					require.Equal(t, 200, w.Code, w.Body.String())
					require.NotNil(t, repo.saved)
					require.Equal(t, metric, repo.saved.MetricType)
					require.Equal(t, threshold, repo.saved.Threshold)
					require.Equal(t, 10, repo.saved.SustainedMinutes)
					require.Equal(t, "P2", repo.saved.Severity)
					require.Equal(t, "openai", repo.saved.Filters["platform"])
					require.Equal(t, float64(7), repo.saved.Filters["group_id"])
					require.True(t, repo.saved.NotifyEmail)
				}
			})
		}
	}
}

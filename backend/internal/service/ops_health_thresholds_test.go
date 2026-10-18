//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestComputeDashboardHealthScore_ConfiguredTTFT(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		name  string
		p99   int
		limit *float64
		want  int
	}{
		{"screenshot 3 successful requests", 2325, float64Ptr(10000), 100},
		{"warning boundary", 8000, float64Ptr(10000), 100},
		{"warning range", 9000, float64Ptr(10000), 83},
		{"critical boundary", 10000, float64Ptr(10000), 65},
		{"slow low volume still penalized", 60000, float64Ptr(10000), 65},
		{"admin stricter threshold", 2325, float64Ptr(2000), 65},
		{"disabled signal", 60000, nil, 100},
		{"zero threshold no latency", 0, float64Ptr(0), 100},
		{"zero threshold positive latency", 1, float64Ptr(0), 65},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ov := &OpsDashboardOverview{RequestCountTotal: 3, SuccessCount: 3, TTFT: OpsPercentiles{P99: intPtr(tc.p99)}}
			require.Equal(t, tc.want, computeDashboardHealthScore(now, ov, &OpsMetricThresholds{TTFTp99MsMax: tc.limit}))
		})
	}
	require.Equal(t, 100, computeDashboardHealthScore(now, &OpsDashboardOverview{RequestCountTotal: 3, TTFT: OpsPercentiles{P99: intPtr(2325)}}, nil))
	require.Less(t, computeDashboardHealthScore(now, &OpsDashboardOverview{SystemMetrics: &OpsSystemMetricsSnapshot{DBOK: boolPtr(false)}}, nil), 90, "idle must not hide a database outage")
	require.Less(t, computeDashboardHealthScore(now, &OpsDashboardOverview{RequestCountTotal: 3, ErrorRate: 1}, nil), 90, "low volume must not hide real request failures")
}

func TestOpsJobStale_Schedules(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 10, 16, 17, 37, 56, 0, loc)
	for _, tc := range []struct {
		name, job, schedule string
		last                time.Time
		want                bool
	}{
		{"daily cleanup already ran", opsCleanupJobName, "0 2 * * *", time.Date(2026, 10, 16, 2, 0, 1, 0, loc), false},
		{"daily cleanup missed", opsCleanupJobName, "0 2 * * *", time.Date(2026, 10, 15, 2, 0, 1, 0, loc), true},
		{"weekly cleanup", opsCleanupJobName, "0 2 * * 1", time.Date(2026, 10, 12, 2, 0, 1, 0, loc), false},
		{"hourly aggregation in grace", opsAggHourlyJobName, "", now.Add(-20 * time.Minute), false},
		{"hourly aggregation missed", opsAggHourlyJobName, "", now.Add(-26 * time.Minute), true},
		{"daily aggregation hourly cadence", opsAggDailyJobName, "", now.Add(-50 * time.Minute), false},
		{"daily aggregation missed", opsAggDailyJobName, "", now.Add(-76 * time.Minute), true},
		{"collector missed", "ops_metrics_collector", "", now.Add(-16 * time.Minute), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, opsJobStaleAt(now, &OpsJobHeartbeat{JobName: tc.job, LastSuccessAt: &tc.last}, tc.schedule, loc))
		})
	}
}

func TestPopulateJobStaleness_DisabledCleanupAndActualFailure(t *testing.T) {
	now := time.Now()
	svc := &OpsService{cfg: &config.Config{}}
	hb := &OpsJobHeartbeat{JobName: opsCleanupJobName, LastSuccessAt: timePtr(now.Add(-72 * time.Hour))}
	svc.populateJobStaleness(now, []*OpsJobHeartbeat{nil, hb})
	require.NotNil(t, hb.Stale)
	require.False(t, *hb.Stale)
	hb.LastErrorAt = &now
	require.Less(t, computeInfraHealth(now, &OpsDashboardOverview{JobHeartbeats: []*OpsJobHeartbeat{hb}}), 100.0, "explicit job failures remain visible")
}

type opsHealthRepo struct {
	OpsRepository
	overview *OpsDashboardOverview
}

func (r *opsHealthRepo) GetDashboardOverview(context.Context, *OpsDashboardFilter) (*OpsDashboardOverview, error) {
	return r.overview, nil
}
func (r *opsHealthRepo) GetLatestSystemMetrics(context.Context, int) (*OpsSystemMetricsSnapshot, error) {
	return nil, nil
}
func (r *opsHealthRepo) ListJobHeartbeats(context.Context) ([]*OpsJobHeartbeat, error) {
	return nil, nil
}

func TestOpsDashboardConfiguredThreshold(t *testing.T) {
	settings := newRuntimeSettingRepoStub()
	svc := &OpsService{settingRepo: settings, opsRepo: &opsHealthRepo{overview: &OpsDashboardOverview{RequestCountTotal: 3, TTFT: OpsPercentiles{P99: intPtr(2325)}}}}
	svc.runtimeSettings.Store(&opsRuntimeSettingsSnapshot{monitoringEnabled: true})
	now := time.Now()
	filter := &OpsDashboardFilter{StartTime: now.Add(-time.Hour), EndTime: now, QueryMode: OpsQueryModeRaw}
	for _, tc := range []struct {
		limit float64
		want  int
	}{{2000, 65}, {10000, 100}} {
		thresholds := defaultOpsMetricThresholds()
		thresholds.TTFTp99MsMax = &tc.limit
		_, err := svc.UpdateMetricThresholds(context.Background(), thresholds)
		require.NoError(t, err)
		overview, err := svc.GetDashboardOverview(context.Background(), filter)
		require.NoError(t, err)
		require.Equal(t, tc.want, overview.HealthScore, "dashboard must use the persisted admin threshold")
	}
}

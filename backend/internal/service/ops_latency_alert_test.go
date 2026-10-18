//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type latencyAlertRepo struct {
	OpsRepository
	overview  *OpsDashboardOverview
	queryErr  error
	filter    *OpsDashboardFilter
	rule      *OpsAlertRule
	events    []*OpsAlertEvent
	heartbeat string
}

func (r *latencyAlertRepo) GetDashboardOverview(_ context.Context, f *OpsDashboardFilter) (*OpsDashboardOverview, error) {
	r.filter = f
	return r.overview, r.queryErr
}
func (r *latencyAlertRepo) ListAlertRules(context.Context) ([]*OpsAlertRule, error) {
	return []*OpsAlertRule{r.rule}, nil
}
func (r *latencyAlertRepo) GetLatestSystemMetrics(context.Context, int) (*OpsSystemMetricsSnapshot, error) {
	return nil, nil
}
func (r *latencyAlertRepo) GetActiveAlertEvent(context.Context, int64) (*OpsAlertEvent, error) {
	for _, e := range r.events {
		if e.Status == OpsAlertStatusFiring {
			return e, nil
		}
	}
	return nil, nil
}
func (r *latencyAlertRepo) GetLatestAlertEvent(context.Context, int64) (*OpsAlertEvent, error) {
	if len(r.events) == 0 {
		return nil, nil
	}
	return r.events[len(r.events)-1], nil
}
func (r *latencyAlertRepo) CreateAlertEvent(_ context.Context, e *OpsAlertEvent) (*OpsAlertEvent, error) {
	e.ID = int64(len(r.events) + 1)
	r.events = append(r.events, e)
	return e, nil
}
func (r *latencyAlertRepo) UpdateAlertEventStatus(_ context.Context, id int64, status string, at *time.Time) error {
	e := r.events[id-1]
	e.Status = status
	e.ResolvedAt = at
	return nil
}
func (r *latencyAlertRepo) UpdateAlertEventEmailSent(_ context.Context, id int64, sent bool) error {
	r.events[id-1].EmailSent = sent
	return nil
}
func (r *latencyAlertRepo) UpsertJobHeartbeat(_ context.Context, in *OpsUpsertJobHeartbeatInput) error {
	if in.LastResult != nil {
		r.heartbeat = *in.LastResult
	}
	return nil
}

type latencyAlertTransport func(*http.Request) (*http.Response, error)

func (f latencyAlertTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestOpsLatencyAlertMetric(t *testing.T) {
	now := time.Now().UTC()
	for _, metric := range []string{"p95_latency_ms", "p99_latency_ms"} {
		t.Run(metric, func(t *testing.T) {
			repo := &latencyAlertRepo{overview: &OpsDashboardOverview{SuccessCount: 3, Duration: OpsPercentiles{P95: intPtr(2262), P99: intPtr(2339)}, TTFT: OpsPercentiles{P95: intPtr(2241), P99: intPtr(2325)}}}
			svc := NewOpsAlertEvaluatorService(nil, repo, nil, nil, nil, nil)
			group := int64(7)
			rule := &OpsAlertRule{MetricType: metric}
			read := func() (float64, bool) {
				return svc.computeRuleMetric(context.Background(), rule, nil, now.Add(-5*time.Minute), now, "openai", &group)
			}
			got, ok := read()
			require.True(t, ok)
			want := 2262.0
			if metric == "p99_latency_ms" {
				want = 2339
			}
			require.Equal(t, want, got, "use full request duration rather than TTFT")
			require.Equal(t, OpsQueryModeRaw, repo.filter.QueryMode)
			require.Equal(t, "openai", repo.filter.Platform)
			require.Equal(t, &group, repo.filter.GroupID)
			require.Equal(t, now.Add(-5*time.Minute), repo.filter.StartTime)
			require.Equal(t, now, repo.filter.EndTime)
			repo.overview.SuccessCount = 0
			_, ok = read()
			require.False(t, ok, "empty success window must not evaluate")
			repo.overview.SuccessCount = 3
			repo.overview.Duration = OpsPercentiles{}
			_, ok = read()
			require.False(t, ok, "missing latency is not zero")
			repo.overview.Duration = OpsPercentiles{P95: intPtr(0), P99: intPtr(0)}
			got, ok = read()
			require.True(t, ok)
			require.Zero(t, got, "recorded zero is a valid observation")
			repo.queryErr = errors.New("database unavailable")
			_, ok = read()
			require.False(t, ok)
			repo.queryErr = nil
			repo.overview = nil
			_, ok = read()
			require.False(t, ok)
		})
	}
}

func latencyAlertFixture(t *testing.T, metric string, enabled bool, minSeverity string) (*OpsAlertEvaluatorService, *latencyAlertRepo, *int) {
	t.Helper()
	settings := newRuntimeSettingRepoStub()
	settings.values[SettingKeyOpsAlertRuntimeSettings] = `{"evaluation_interval_seconds":60,"distributed_lock":{"enabled":false}}`
	cfg := defaultOpsEmailNotificationConfig()
	cfg.Alert.Enabled = enabled
	cfg.Alert.MinSeverity = minSeverity
	cfg.Alert.Recipients = []string{"ops@example.com"}
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)
	settings.values[SettingKeyOpsEmailNotificationConfig] = string(raw)
	settings.values[SettingKeyEmailProvider] = EmailProviderBrevo
	settings.values[SettingKeyBrevoAPIKey] = "test-key"
	settings.values[SettingKeyBrevoFrom] = "monitor@example.com"
	calls := 0
	mail := NewEmailService(settings, nil)
	// Exercise email dispatch in memory. No connection to Brevo or real mailbox.
	mail.brevoClient = &http.Client{Transport: latencyAlertTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), "ops@example.com")
		require.Contains(t, string(body), metric)
		require.Contains(t, string(body), "4500.00")
		return &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"messageId":"test"}`)), Header: make(http.Header)}, nil
	})}
	repo := &latencyAlertRepo{overview: &OpsDashboardOverview{SuccessCount: 10, Duration: OpsPercentiles{P95: intPtr(4500), P99: intPtr(4500)}}, rule: &OpsAlertRule{ID: 4, Name: "Latency", Enabled: true, Severity: "P2", MetricType: metric, Operator: ">", Threshold: 3000, WindowMinutes: 5, SustainedMinutes: 2, CooldownMinutes: 30, NotifyEmail: true}}
	ops := &OpsService{settingRepo: settings, opsRepo: repo}
	return NewOpsAlertEvaluatorService(ops, repo, mail, nil, nil, nil), repo, &calls
}

func TestOpsLatencyAlertLifecycle(t *testing.T) {
	for _, metric := range []string{"p95_latency_ms", "p99_latency_ms"} {
		t.Run(metric, func(t *testing.T) {
			svc, repo, calls := latencyAlertFixture(t, metric, true, "")
			evaluate := func() { svc.evaluateOnce(time.Minute) }
			evaluate()
			require.Empty(t, repo.events, "a single breach must not fire")
			repo.overview.Duration = OpsPercentiles{}
			evaluate()
			require.Empty(t, repo.events)
			repo.overview.Duration = OpsPercentiles{P95: intPtr(4500), P99: intPtr(4500)}
			evaluate()
			require.Empty(t, repo.events, "missing data resets sustained breach counter")
			evaluate()
			require.Len(t, repo.events, 1)
			require.Equal(t, OpsAlertStatusFiring, repo.events[0].Status)
			require.Equal(t, 4500.0, *repo.events[0].MetricValue)
			require.Equal(t, 3000.0, *repo.events[0].ThresholdValue)
			require.True(t, repo.events[0].EmailSent)
			require.Equal(t, 1, *calls)
			evaluate()
			require.Len(t, repo.events, 1)
			require.Equal(t, 1, *calls, "active event must not repeatedly email")
			repo.overview.Duration = OpsPercentiles{}
			evaluate()
			require.Equal(t, OpsAlertStatusFiring, repo.events[0].Status, "missing data is not recovery")
			repo.overview.Duration = OpsPercentiles{P95: intPtr(3000), P99: intPtr(3000)}
			evaluate()
			require.Equal(t, OpsAlertStatusResolved, repo.events[0].Status)
			require.NotNil(t, repo.events[0].ResolvedAt)
			repo.overview.Duration = OpsPercentiles{P95: intPtr(4500), P99: intPtr(4500)}
			evaluate()
			evaluate()
			require.Len(t, repo.events, 1, "cooldown prevents repeated incidents")
			repo.events[0].FiredAt = time.Now().Add(-31 * time.Minute)
			evaluate()
			require.Len(t, repo.events, 2)
			require.Equal(t, 2, *calls)
			require.Contains(t, repo.heartbeat, "evaluated=1")
		})
	}
}

func TestOpsLatencyAlertEmailGates(t *testing.T) {
	for _, tc := range []struct {
		name    string
		enabled bool
		min     string
		notify  bool
	}{
		{"global disabled", false, "", true},
		{"critical-only excludes P2", true, "critical", true},
		{"rule notification disabled", true, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo, calls := latencyAlertFixture(t, "p95_latency_ms", tc.enabled, tc.min)
			repo.rule.NotifyEmail = tc.notify
			svc.evaluateOnce(time.Minute)
			svc.evaluateOnce(time.Minute)
			require.Len(t, repo.events, 1)
			require.False(t, repo.events[0].EmailSent)
			require.Zero(t, *calls)
		})
	}
}

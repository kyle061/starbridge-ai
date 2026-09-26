package service

import (
	"testing"
	"time"
)

func TestFinalizeOperationalOrDegradedThreshold(t *testing.T) {
	tests := []struct {
		name    string
		latency time.Duration
		want    string
	}{
		{name: "successful response below threshold stays operational", latency: 9999 * time.Millisecond, want: MonitorStatusOperational},
		{name: "response at threshold is degraded", latency: 10 * time.Second, want: MonitorStatusDegraded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			latencyMs := int(tt.latency / time.Millisecond)
			result := finalizeOperationalOrDegraded(&CheckResult{}, tt.latency, latencyMs)
			if result.Status != tt.want {
				t.Fatalf("status = %q, want %q", result.Status, tt.want)
			}
		})
	}
}

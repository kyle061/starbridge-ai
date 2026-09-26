package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogOpenAIWSAccountSelectionFailure(t *testing.T) {
	t.Run("failover exhaustion logs the upstream cause without a second warning", func(t *testing.T) {
		core, logs := observer.New(zap.DebugLevel)
		lastFailoverErr := &service.UpstreamFailoverError{
			StatusCode: 502,
			Stage:      service.GatewayFailureStageInference,
			Scope:      service.GatewayFailureScopeAccount,
			Reason:     "upstream_unavailable",
		}

		logOpenAIWSAccountSelectionFailure(zap.New(core), service.ErrNoAvailableAccounts, service.PlatformOpenAI, 1, lastFailoverErr)

		require.Len(t, logs.All(), 1)
		entry := logs.All()[0]
		require.Equal(t, zap.InfoLevel, entry.Level)
		require.Equal(t, "openai.websocket_failover_exhausted", entry.Message)
		require.Equal(t, "no available accounts", entry.ContextMap()["error"])
		require.Equal(t, int64(502), entry.ContextMap()["upstream_status"])
		require.Equal(t, "inference", entry.ContextMap()["upstream_stage"])
		require.Equal(t, "account", entry.ContextMap()["upstream_scope"])
		require.Equal(t, "upstream_unavailable", entry.ContextMap()["upstream_reason"])
		require.Equal(t, int64(1), entry.ContextMap()["excluded_account_count"])
	})

	t.Run("initial selection failure remains a warning", func(t *testing.T) {
		core, logs := observer.New(zap.DebugLevel)

		logOpenAIWSAccountSelectionFailure(zap.New(core), service.ErrNoAvailableAccounts, service.PlatformOpenAI, 0, nil)

		require.Len(t, logs.All(), 1)
		entry := logs.All()[0]
		require.Equal(t, zap.WarnLevel, entry.Level)
		require.Equal(t, "openai.websocket_account_select_failed", entry.Message)
		require.Equal(t, int64(0), entry.ContextMap()["excluded_account_count"])
	})
}

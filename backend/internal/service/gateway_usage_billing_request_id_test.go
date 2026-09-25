//go:build unit

package service

import (
	"context"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

func TestResolveUsageBillingRequestID_ForcedWebSearchBeatsClientID(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), ctxkey.ClientRequestID, "client-shared-id")
	ctx = context.WithValue(ctx, ctxkey.RequestID, "server-request-id")
	got := resolveUsageBillingRequestID(ctx, "web_search:uuid-1")
	require.Equal(t, "web_search:uuid-1", got)
}

func TestResolveUsageBillingRequestID_GPT6PreparationBeatsServerAndClientIDs(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), ctxkey.ClientRequestID, "client-shared-id")
	ctx = context.WithValue(ctx, ctxkey.RequestID, "server-request-id")
	got := resolveUsageBillingRequestID(ctx, "gpt6-prep:upstream-1")
	require.Equal(t, "gpt6-prep:upstream-1", got)
}

func TestResolveUsageBillingRequestID_ServerRequestIDWinsOverClientAndUpstream(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), ctxkey.ClientRequestID, "client-shared-id")
	ctx = context.WithValue(ctx, ctxkey.RequestID, "server-request-1")
	got := resolveUsageBillingRequestID(ctx, "resp_abc")
	require.Equal(t, "local:server-request-1", got)
}

func TestResolveUsageBillingRequestID_UpstreamWinsWhenServerRequestIDMissing(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), ctxkey.ClientRequestID, "client-shared-id")
	got := resolveUsageBillingRequestID(ctx, "resp_abc")
	require.Equal(t, "resp_abc", got)
}

func TestResolveUsageBillingRequestID_ClientIsLastFallback(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), ctxkey.ClientRequestID, "client-shared-id")
	got := resolveUsageBillingRequestID(ctx, "")
	require.Equal(t, "client:client-shared-id", got)
}

func TestResolveUsageBillingRequestID_ReusedClientIDDoesNotReuseBillingID(t *testing.T) {
	t.Parallel()
	newContext := func(serverRequestID string) context.Context {
		ctx := context.WithValue(context.Background(), ctxkey.ClientRequestID, "client-shared-id")
		return context.WithValue(ctx, ctxkey.RequestID, serverRequestID)
	}
	first := resolveUsageBillingRequestID(newContext("server-request-1"), "upstream-1")
	second := resolveUsageBillingRequestID(newContext("server-request-2"), "upstream-2")
	require.Equal(t, "local:server-request-1", first)
	require.Equal(t, "local:server-request-2", second)
	require.NotEqual(t, first, second)
}

func TestIsForcedUsageBillingRequestID(t *testing.T) {
	t.Parallel()
	require.True(t, isForcedUsageBillingRequestID("web_search:x"))
	require.True(t, isForcedUsageBillingRequestID("gpt6-prep:x"))
	require.True(t, isForcedUsageBillingRequestID("grok-video:task-1"))
	require.True(t, isForcedUsageBillingRequestID("grok_audio:up-1"))
	require.True(t, isForcedUsageBillingRequestID("grok_realtime:sess-1"))
	require.False(t, isForcedUsageBillingRequestID("resp_abc"))
}

func TestStableGrokAudioBillingRequestID(t *testing.T) {
	t.Parallel()
	require.Equal(t, "grok_audio:up-1", StableGrokAudioBillingRequestID("up-1"))
	require.Equal(t, "grok_audio:up-1", StableGrokAudioBillingRequestID("grok_audio:up-1"))
	got := StableGrokAudioBillingRequestID("")
	require.True(t, strings.HasPrefix(got, "grok_audio:"))
	require.Greater(t, len(got), len("grok_audio:"))
}

func TestStableGrokRealtimeBillingRequestID(t *testing.T) {
	t.Parallel()
	require.Equal(t, "grok_realtime:s1", StableGrokRealtimeBillingRequestID("s1"))
	require.Equal(t, "grok_realtime:s1", StableGrokRealtimeBillingRequestID("grok_realtime:s1"))
	got := StableGrokRealtimeBillingRequestID("")
	require.True(t, strings.HasPrefix(got, "grok_realtime:"))
}

func TestResolveUsageBillingRequestID_ForcedGrokAudioBeatsClientID(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), ctxkey.ClientRequestID, "client-shared-id")
	ctx = context.WithValue(ctx, ctxkey.RequestID, "server-request-id")
	got := resolveUsageBillingRequestID(ctx, StableGrokAudioBillingRequestID("up-9"))
	require.Equal(t, "grok_audio:up-9", got)
}

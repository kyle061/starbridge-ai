package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type openAIForwardTimingKey struct{}

var openAISlowForwardLogThrottle = newAccountWriteThrottle(time.Minute)

// Measures existing HTTP traffic without reading bodies or sending probes.
type openAIForwardTiming struct {
	startedAt       time.Time
	metricStartedAt time.Time
	mu              sync.Mutex
	attempts        []*openAIHTTPAttemptTiming
}

type openAIHTTPAttemptTiming struct {
	startedAt, connectedAt, writtenAt, firstByteAt, headersAt time.Time
	reused                                                    bool
}

func (t *openAIForwardTiming) traceRequest(request *http.Request) (*http.Request, func()) {
	a := &openAIHTTPAttemptTiming{startedAt: time.Now()}
	t.mu.Lock()
	t.attempts = append(t.attempts, a)
	t.mu.Unlock()
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			t.mu.Lock()
			a.connectedAt, a.reused = time.Now(), info.Reused
			t.mu.Unlock()
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			if info.Err == nil {
				t.mu.Lock()
				a.writtenAt = time.Now()
				t.mu.Unlock()
			}
		},
		GotFirstResponseByte: func() {
			t.mu.Lock()
			a.firstByteAt = time.Now()
			t.mu.Unlock()
		},
	}
	return request.WithContext(httptrace.WithClientTrace(request.Context(), trace)), func() {
		t.mu.Lock()
		a.headersAt = time.Now()
		t.mu.Unlock()
	}
}

func timingIntervalMs(start, end time.Time) int64 {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return -1
	}
	return end.Sub(start).Milliseconds()
}

func (t *openAIForwardTiming) summary(firstTokenMs int) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.attempts) == 0 {
		return ""
	}
	first, last := t.attempts[0], t.attempts[len(t.attempts)-1]
	firstTokenAt := t.metricStartedAt.Add(time.Duration(firstTokenMs) * time.Millisecond)
	return fmt.Sprintf("openai.slow_first_token ttft_ms=%d prepare_ms=%d http_attempts=%d before_last_attempt_ms=%d conn_ms=%d upload_ms=%d first_byte_wait_ms=%d headers_ms=%d after_headers_ms=%d conn_reused=%t",
		firstTokenMs, timingIntervalMs(t.startedAt, first.startedAt), len(t.attempts),
		timingIntervalMs(first.startedAt, last.startedAt),
		timingIntervalMs(last.startedAt, last.connectedAt),
		timingIntervalMs(last.connectedAt, last.writtenAt),
		timingIntervalMs(last.writtenAt, last.firstByteAt),
		timingIntervalMs(last.startedAt, last.headersAt),
		timingIntervalMs(last.headersAt, firstTokenAt), last.reused)
}

func (t *openAIForwardTiming) logSlowRequest(ctx context.Context, c *gin.Context, account *Account, result *OpenAIForwardResult) {
	if result == nil || result.FirstTokenMs == nil || *result.FirstTokenMs < 5000 {
		return
	}
	message := t.summary(*result.FirstTokenMs)
	if message == "" || !openAISlowForwardLogThrottle.Allow(account.ID, time.Now()) {
		return
	}
	fields := []zap.Field{
		zap.String("component", "gateway.latency"),
		zap.Int64("account_id", account.ID),
		zap.String("platform", account.Platform),
		zap.String("model", result.Model),
	}
	for _, key := range []string{OpsAuthLatencyMsKey, OpsRoutingLatencyMsKey} {
		if value, ok := c.Get(key); ok {
			fields = append(fields, zap.Any(key, value))
			message += fmt.Sprintf(" %s=%v", key, value)
		}
	}
	logger.FromContext(ctx).Info(message, fields...)
}

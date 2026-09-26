package service

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOpenAIForwardTimingDoesNotWaitForOrReadStreamBody(t *testing.T) {
	releaseBody := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, "request-body", string(body))
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-releaseBody
		_, _ = io.WriteString(w, "data: stream-output\n\n")
	}))
	defer server.Close()
	defer close(releaseBody)
	request, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader("request-body"))
	require.NoError(t, err)
	parentCalled := false
	request = request.WithContext(httptrace.WithClientTrace(request.Context(), &httptrace.ClientTrace{
		GotFirstResponseByte: func() { parentCalled = true },
	}))
	timing := &openAIForwardTiming{startedAt: time.Now(), metricStartedAt: time.Now()}
	traced, finish := timing.traceRequest(request)
	client := server.Client()
	client.Timeout = time.Second
	response, err := client.Do(traced)
	finish()
	require.NoError(t, err)
	defer response.Body.Close()
	require.True(t, parentCalled, "preserve existing transport instrumentation")
	require.Len(t, timing.attempts, 1)
	require.False(t, timing.attempts[0].headersAt.IsZero())
	require.Contains(t, timing.summary(5000), "http_attempts=1")
	require.NotContains(t, timing.summary(5000), "request-body")
}

func TestOpenAIForwardTimingSeparatesPreparationUploadAndUpstreamWait(t *testing.T) {
	start := time.Now()
	timing := &openAIForwardTiming{startedAt: start, metricStartedAt: start,
		attempts: []*openAIHTTPAttemptTiming{{
			startedAt: start.Add(20 * time.Millisecond), connectedAt: start.Add(50 * time.Millisecond),
			writtenAt: start.Add(100 * time.Millisecond), firstByteAt: start.Add(200 * time.Millisecond),
			headersAt: start.Add(220 * time.Millisecond), reused: true,
		}},
	}
	message := timing.summary(5220)
	for _, field := range []string{"prepare_ms=20", "conn_ms=30", "upload_ms=50", "first_byte_wait_ms=100", "headers_ms=200", "after_headers_ms=5000", "conn_reused=true"} {
		require.Contains(t, message, field)
	}
	timing.attempts = append(timing.attempts, &openAIHTTPAttemptTiming{startedAt: start.Add(time.Second), headersAt: start.Add(2 * time.Second)})
	message = timing.summary(6000)
	require.Contains(t, message, "http_attempts=2")
	require.Contains(t, message, "before_last_attempt_ms=980")
	require.Contains(t, message, "conn_ms=-1", "absent transport callbacks must not look like zero network latency")
	require.Contains(t, message, "after_headers_ms=4000")
}

package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/stretchr/testify/require"
)

type deviceAuthStub struct {
	OpenAIOAuthClient
	grant                     *OpenAIDeviceGrant
	pollCalls, exchangeCalls  int
	exchangeErr               error
	redirect, proxy, clientID string
}

func (c *deviceAuthStub) StartDeviceAuth(context.Context, string) (*OpenAIDeviceCode, error) {
	return &OpenAIDeviceCode{DeviceAuthID: "private-device-id", UserCode: "ABCD-EFGH", Interval: 1}, nil
}
func (c *deviceAuthStub) PollDeviceAuth(_ context.Context, _, _, proxy string) (*OpenAIDeviceGrant, error) {
	c.pollCalls++
	c.proxy = proxy
	return c.grant, nil
}
func (c *deviceAuthStub) ExchangeCode(_ context.Context, code, verifier, redirect, proxy, clientID string) (*openai.TokenResponse, error) {
	c.exchangeCalls++
	c.redirect = redirect
	c.clientID = clientID
	c.proxy = proxy
	if c.exchangeErr != nil {
		return nil, c.exchangeErr
	}
	return &openai.TokenResponse{AccessToken: "access", RefreshToken: "refresh", ExpiresIn: 3600}, nil
}
func newDeviceTest(t *testing.T) (*OpenAIOAuthService, *deviceAuthStub, *OpenAIDeviceStartResult) {
	t.Helper()
	client := &deviceAuthStub{}
	svc := NewOpenAIOAuthService(nil, client)
	t.Cleanup(svc.Stop)
	result, err := svc.StartDeviceAuth(context.Background(), 7, nil)
	require.NoError(t, err)
	t.Cleanup(func() { svc.CancelDeviceAuth(result.SessionID, 7) })
	return svc, client, result
}
func allowDevicePoll(svc *OpenAIOAuthService, id string) {
	session, _ := svc.deviceSession(id, 7)
	session.mu.Lock()
	session.nextPoll = time.Time{}
	session.mu.Unlock()
}
func TestOpenAIDeviceOwnerExpiryAndCancellation(t *testing.T) {
	svc, client, start := newDeviceTest(t)
	require.Equal(t, 5, start.Interval)
	require.Equal(t, "https://auth.openai.com/codex/device", start.VerificationURL)
	_, err := svc.PollDeviceAuth(context.Background(), start.SessionID, 8)
	require.Error(t, err)
	svc.CancelDeviceAuth(start.SessionID, 8)
	_, err = svc.PollDeviceAuth(context.Background(), start.SessionID, 7)
	require.NoError(t, err)
	require.Zero(t, client.pollCalls)
	svc.CancelDeviceAuth(start.SessionID, 7)
	_, err = svc.PollDeviceAuth(context.Background(), start.SessionID, 7)
	require.Error(t, err)
	start, err = svc.StartDeviceAuth(context.Background(), 7, nil)
	require.NoError(t, err)
	svc.deviceSessions[start.SessionID].expiresAt = time.Now().Add(-time.Second)
	_, err = svc.PollDeviceAuth(context.Background(), start.SessionID, 7)
	require.Error(t, err)
	require.Zero(t, client.exchangeCalls)
}
func TestOpenAIDevicePendingThrottleAndConcurrentCompletion(t *testing.T) {
	svc, client, start := newDeviceTest(t)
	allowDevicePoll(svc, start.SessionID)
	pending, err := svc.PollDeviceAuth(context.Background(), start.SessionID, 7)
	require.NoError(t, err)
	require.Equal(t, "pending", pending.Status)
	_, err = svc.PollDeviceAuth(context.Background(), start.SessionID, 7)
	require.NoError(t, err)
	require.Equal(t, 1, client.pollCalls)
	client.grant = &OpenAIDeviceGrant{AuthorizationCode: "code", CodeVerifier: "verifier"}
	allowDevicePoll(svc, start.SessionID)
	var wg sync.WaitGroup
	results := make(chan *OpenAIDevicePollResult, 8)
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := svc.PollDeviceAuth(context.Background(), start.SessionID, 7)
			results <- r
			errs <- e
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	for r := range results {
		require.Equal(t, "authorized", r.Status)
		require.Equal(t, "access", r.TokenInfo.AccessToken)
	}
	require.Equal(t, 1, client.exchangeCalls)
	require.Equal(t, 2, client.pollCalls)
	require.Equal(t, "https://auth.openai.com/deviceauth/callback", client.redirect)
	require.Equal(t, openai.ClientID, client.clientID)
}
func TestOpenAIDeviceRetriesExchangeWithoutConsumingGrantAgain(t *testing.T) {
	svc, client, start := newDeviceTest(t)
	client.grant = &OpenAIDeviceGrant{AuthorizationCode: "code", CodeVerifier: "verifier"}
	client.exchangeErr = errors.New("temporary failure")
	session, _ := svc.deviceSession(start.SessionID, 7)
	session.proxyURL = "http://proxy.test:8080"
	allowDevicePoll(svc, start.SessionID)
	_, err := svc.PollDeviceAuth(context.Background(), start.SessionID, 7)
	require.Error(t, err)
	client.exchangeErr = nil
	allowDevicePoll(svc, start.SessionID)
	result, err := svc.PollDeviceAuth(context.Background(), start.SessionID, 7)
	require.NoError(t, err)
	require.Equal(t, "authorized", result.Status)
	require.Equal(t, 1, client.pollCalls)
	require.Equal(t, 2, client.exchangeCalls)
	require.Equal(t, "http://proxy.test:8080", client.proxy)
}

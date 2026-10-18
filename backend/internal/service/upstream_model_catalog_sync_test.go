package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type upstreamCatalogSyncRepoStub struct {
	AccountRepository
	accounts []Account
}

func (r *upstreamCatalogSyncRepoStub) ListActive(context.Context) ([]Account, error) {
	return append([]Account(nil), r.accounts...), nil
}

type upstreamCatalogSyncerStub struct {
	mu    sync.Mutex
	calls []int64
}

func (s *upstreamCatalogSyncerStub) SyncUpstreamModelCatalog(_ context.Context, account *Account) (*UpstreamModelCatalog, error) {
	s.mu.Lock()
	s.calls = append(s.calls, account.ID)
	s.mu.Unlock()
	return &UpstreamModelCatalog{Models: []string{"deepseek-v6", "gpt-7.2"}}, nil
}

func (s *upstreamCatalogSyncerStub) calledIDs() []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]int64(nil), s.calls...)
}

func TestUpstreamModelCatalogSyncSkipsFreshAccounts(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	fresh := Account{ID: 1, Platform: PlatformOpenAI, Schedulable: true}
	fresh.SetUpstreamModelCatalogSnapshot(UpstreamModelCatalogSnapshot{
		Source:   "upstream",
		SyncedAt: now.Add(-time.Hour).Format(time.RFC3339),
		Models:   []string{"gpt-6.1-sol"},
	})
	repo := &upstreamCatalogSyncRepoStub{accounts: []Account{
		fresh,
		{ID: 2, Platform: PlatformDeepseek, Schedulable: true},
		{ID: 3, Platform: PlatformOpenAI, Schedulable: false},
	}}
	syncer := &upstreamCatalogSyncerStub{}
	svc := NewUpstreamModelCatalogSyncService(repo, syncer, nil, 6*time.Hour)
	svc.now = func() time.Time { return now }

	svc.syncAccounts(context.Background(), true)

	require.ElementsMatch(t, []int64{2, 3}, syncer.calledIDs())
}

func TestUpstreamModelCatalogSyncRunsInitialScanAndStops(t *testing.T) {
	repo := &upstreamCatalogSyncRepoStub{accounts: []Account{{ID: 42, Platform: PlatformDeepseek, Schedulable: true}}}
	syncer := &upstreamCatalogSyncerStub{}
	svc := NewUpstreamModelCatalogSyncService(repo, syncer, nil, time.Hour)
	svc.Start()
	defer svc.Stop()

	deadline := time.Now().Add(time.Second)
	for len(syncer.calledIDs()) == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	require.Equal(t, []int64{42}, syncer.calledIDs())
}

func TestUpstreamModelCatalogRecentlySyncedRejectsInvalidAndFutureTimestamps(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	for _, syncedAt := range []string{"invalid", now.Add(time.Minute).Format(time.RFC3339)} {
		account := Account{ID: 1}
		account.SetUpstreamModelCatalogSnapshot(UpstreamModelCatalogSnapshot{
			SyncedAt: syncedAt,
			Models:   []string{"model"},
		})
		require.False(t, upstreamModelCatalogRecentlySynced(&account, now, time.Hour))
	}
}

func TestModelCatalogIDsChanged(t *testing.T) {
	require.True(t, modelCatalogIDsChanged(nil, []string{"deepseek-v6"}))
	require.False(t, modelCatalogIDsChanged(&UpstreamModelCatalogSnapshot{Models: []string{"gpt-7", "deepseek-v6"}}, []string{"deepseek-v6", "gpt-7"}))
	require.True(t, modelCatalogIDsChanged(&UpstreamModelCatalogSnapshot{Models: []string{"gpt-7"}}, []string{"gpt-7", "gpt-8"}))
}

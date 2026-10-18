package service

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const (
	upstreamModelCatalogSyncInterval = 6 * time.Hour
	upstreamModelCatalogSyncTimeout  = 45 * time.Second
	upstreamModelCatalogSyncWorkers  = 4
)

type upstreamModelCatalogSyncer interface {
	SyncUpstreamModelCatalog(ctx context.Context, account *Account) (*UpstreamModelCatalog, error)
}

// UpstreamModelCatalogSyncService keeps account-scoped model permissions fresh
// without making request handlers call upstream catalog endpoints.
type UpstreamModelCatalogSyncService struct {
	accountRepo AccountRepository
	syncer      upstreamModelCatalogSyncer
	gateway     *GatewayService
	interval    time.Duration
	stopCh      chan struct{}
	stopOnce    sync.Once
	startOnce   sync.Once
	wg          sync.WaitGroup
	mu          sync.Mutex
	cancel      context.CancelFunc
	now         func() time.Time
}

func NewUpstreamModelCatalogSyncService(
	accountRepo AccountRepository,
	syncer upstreamModelCatalogSyncer,
	gateway *GatewayService,
	interval time.Duration,
) *UpstreamModelCatalogSyncService {
	return &UpstreamModelCatalogSyncService{
		accountRepo: accountRepo,
		syncer:      syncer,
		gateway:     gateway,
		interval:    interval,
		stopCh:      make(chan struct{}),
		now:         time.Now,
	}
}

func (s *UpstreamModelCatalogSyncService) Start() {
	if s == nil || s.accountRepo == nil || s.syncer == nil || s.interval <= 0 {
		return
	}
	s.startOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		s.mu.Lock()
		s.cancel = cancel
		s.mu.Unlock()
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			ticker := time.NewTicker(s.interval)
			defer ticker.Stop()
			s.syncAccounts(ctx, true)
			for {
				select {
				case <-ticker.C:
					s.syncAccounts(ctx, false)
				case <-s.stopCh:
					return
				case <-ctx.Done():
					return
				}
			}
		}()
	})
}

func (s *UpstreamModelCatalogSyncService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() { close(s.stopCh) })
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Unlock()
	s.wg.Wait()
}

func (s *UpstreamModelCatalogSyncService) syncAccounts(ctx context.Context, skipFresh bool) {
	accounts, err := s.accountRepo.ListActive(ctx)
	if err != nil {
		slog.Warn("upstream_model_catalog_sync_account_list_failed", "error", err)
		return
	}

	semaphore := make(chan struct{}, upstreamModelCatalogSyncWorkers)
	var wg sync.WaitGroup
	var changed sync.Map
	for i := range accounts {
		account := accounts[i]
		if skipFresh && upstreamModelCatalogRecentlySynced(&account, s.now(), s.interval) {
			continue
		}
		select {
		case semaphore <- struct{}{}:
		case <-ctx.Done():
			wg.Wait()
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-semaphore }()
			accountCtx, cancel := context.WithTimeout(ctx, upstreamModelCatalogSyncTimeout)
			defer cancel()
			previous := account.GetUpstreamModelCatalogSnapshot()
			catalog, syncErr := s.syncer.SyncUpstreamModelCatalog(accountCtx, &account)
			if syncErr != nil {
				slog.Warn("upstream_model_catalog_sync_failed",
					"account_id", account.ID,
					"platform", account.Platform,
					"error", syncErr,
				)
				return
			}
			if catalog != nil && modelCatalogIDsChanged(previous, catalog.Models) {
				changed.Store(account.Platform, struct{}{})
			}
		}()
	}
	wg.Wait()
	if s.gateway != nil {
		changed.Range(func(key, _ any) bool {
			if platform, ok := key.(string); ok {
				s.gateway.InvalidateAvailableModelsCache(nil, platform)
			}
			return true
		})
	}
}

func upstreamModelCatalogRecentlySynced(account *Account, now time.Time, interval time.Duration) bool {
	if account == nil || interval <= 0 {
		return false
	}
	snapshot := account.GetUpstreamModelCatalogSnapshot()
	if snapshot == nil {
		return false
	}
	syncedAt, err := time.Parse(time.RFC3339, snapshot.SyncedAt)
	if err != nil {
		return false
	}
	age := now.Sub(syncedAt)
	return age >= 0 && age < interval
}

func modelCatalogIDsChanged(previous *UpstreamModelCatalogSnapshot, next []string) bool {
	if previous == nil {
		return len(next) > 0
	}
	previousIDs := dedupeAndSortModelIDs(previous.Models)
	nextIDs := dedupeAndSortModelIDs(next)
	if len(previousIDs) != len(nextIDs) {
		return true
	}
	for i := range previousIDs {
		if previousIDs[i] != nextIDs[i] {
			return true
		}
	}
	return false
}

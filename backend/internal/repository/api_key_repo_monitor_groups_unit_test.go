package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyRepositoryListActiveGroupIDsByUserID(t *testing.T) {
	repo, client := newAPIKeyRepoSQLite(t)
	ctx := context.Background()
	owner := mustCreateAPIKeyRepoUser(t, ctx, client, "monitor-groups-owner@test.com")
	other := mustCreateAPIKeyRepoUser(t, ctx, client, "monitor-groups-other@test.com")
	groupA, err := client.Group.Create().SetName("monitor-group-a").Save(ctx)
	require.NoError(t, err)
	groupB, err := client.Group.Create().SetName("monitor-group-b").Save(ctx)
	require.NoError(t, err)
	groupC, err := client.Group.Create().SetName("monitor-group-c").Save(ctx)
	require.NoError(t, err)
	past := time.Now().Add(-time.Hour)
	add := func(userID int64, name, status string, groupID *int64, expiresAt *time.Time) *service.APIKey {
		key := &service.APIKey{UserID: userID, Key: "sk-" + name, Name: name, Status: status, GroupID: groupID, ExpiresAt: expiresAt}
		require.NoError(t, repo.Create(ctx, key))
		return key
	}
	add(owner.ID, "active-a", service.StatusActive, &groupA.ID, nil)
	add(owner.ID, "active-a-duplicate", service.StatusActive, &groupA.ID, nil)
	add(owner.ID, "active-b", service.StatusActive, &groupB.ID, nil)
	add(owner.ID, "disabled", service.StatusAPIKeyDisabled, &groupC.ID, nil)
	add(owner.ID, "expired", service.StatusActive, &groupC.ID, &past)
	add(owner.ID, "unbound", service.StatusActive, nil, nil)
	add(other.ID, "other-user", service.StatusActive, &groupB.ID, nil)

	ids, err := repo.ListActiveGroupIDsByUserID(ctx, owner.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, []int64{groupA.ID, groupB.ID}, ids)

	ids, err = repo.ListActiveGroupIDsByUserID(ctx, 999999)
	require.NoError(t, err)
	require.Empty(t, ids)
}

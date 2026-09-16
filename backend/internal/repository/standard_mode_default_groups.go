package repository

import (
	"context"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/compositemodelroute"
	"github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const standardRelayDefaultGroupName = "composite-default"
const standardRelayDefaultGroupDescription = "Default OpenAI + DeepSeek relay"

type standardRelayRoute struct {
	groupID        int64
	publicModel    string
	matchType      string
	targetPlatform string
	upstreamModel  string
	priority       int
	notes          string
}

// ensureStandardRelayDefaultGroups creates the one-click relay groups used by
// the standard-mode onboarding path. Existing operator-created groups are
// left untouched.
func ensureStandardRelayDefaultGroups(ctx context.Context, client *dbent.Client) error {
	if client == nil {
		return fmt.Errorf("nil ent client")
	}
	defaults := []struct {
		name     string
		platform string
		rate     float64
	}{
		{name: service.PlatformOpenAI + "-default", platform: service.PlatformOpenAI, rate: 1.0},
		{name: service.PlatformDeepseek + "-default", platform: service.PlatformDeepseek, rate: 1.0},
		// Astra and other premium models use the relay's single configured
		// multiplier. The existing OpenAI priority-tier pricing remains intact.
		{name: "composite-default", platform: service.PlatformComposite, rate: 1.5},
	}
	for _, item := range defaults {
		if err := createStandardRelayGroupIfNotExists(ctx, client, item.name, item.platform, item.rate); err != nil {
			return err
		}
	}

	// Seed the failover chain only for the group created by this onboarding
	// path. An operator-created group with the same name remains untouched.
	compositeGroup, err := client.Group.Query().
		Where(
			group.NameEQ(standardRelayDefaultGroupName),
			group.PlatformEQ(service.PlatformComposite),
			group.DescriptionEQ(standardRelayDefaultGroupDescription),
			group.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("load default composite group: %w", err)
	}

	for _, route := range standardRelayDefaultRoutes(compositeGroup.ID) {
		if err := createStandardRelayRouteIfNotExists(ctx, client, route); err != nil {
			return err
		}
	}
	return nil
}

func standardRelayDefaultRoutes(groupID int64) []standardRelayRoute {
	return []standardRelayRoute{
		{
			groupID:        groupID,
			publicModel:    "gpt-6-astra",
			matchType:      service.CompositeRouteMatchExact,
			targetPlatform: service.PlatformOpenAI,
			upstreamModel:  "gpt-6-astra",
			priority:       10,
			notes:          "Primary OpenAI Astra route",
		},
		{
			groupID:        groupID,
			publicModel:    "gpt-6-astra",
			matchType:      service.CompositeRouteMatchExact,
			targetPlatform: service.PlatformDeepseek,
			upstreamModel:  "deepseek-v4-pro",
			priority:       20,
			notes:          "DeepSeek fallback when OpenAI is unavailable",
		},
		{
			groupID:        groupID,
			publicModel:    "deepseek-v4-pro",
			matchType:      service.CompositeRouteMatchExact,
			targetPlatform: service.PlatformDeepseek,
			upstreamModel:  "deepseek-v4-pro",
			priority:       10,
			notes:          "Primary DeepSeek route",
		},
		{
			groupID:        groupID,
			publicModel:    "deepseek-v4-pro",
			matchType:      service.CompositeRouteMatchExact,
			targetPlatform: service.PlatformOpenAI,
			upstreamModel:  "gpt-6-astra",
			priority:       20,
			notes:          "OpenAI fallback when DeepSeek is unavailable",
		},
	}
}

func createStandardRelayRouteIfNotExists(ctx context.Context, client *dbent.Client, route standardRelayRoute) error {
	exists, err := client.CompositeModelRoute.Query().
		Where(
			compositemodelroute.GroupIDEQ(route.groupID),
			compositemodelroute.PublicModelEQ(route.publicModel),
			compositemodelroute.MatchTypeEQ(route.matchType),
			compositemodelroute.TargetPlatformEQ(route.targetPlatform),
			compositemodelroute.UpstreamModelEQ(route.upstreamModel),
			compositemodelroute.EndpointEQ(service.CompositeRouteEndpointAny),
			compositemodelroute.DeletedAtIsNil(),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check default relay route %s/%s: %w", route.publicModel, route.targetPlatform, err)
	}
	if exists {
		return nil
	}

	_, err = client.CompositeModelRoute.Create().
		SetGroupID(route.groupID).
		SetPublicModel(route.publicModel).
		SetMatchType(route.matchType).
		SetTargetPlatform(route.targetPlatform).
		SetUpstreamModel(route.upstreamModel).
		SetEndpoint(service.CompositeRouteEndpointAny).
		SetPriority(route.priority).
		SetEnabled(true).
		SetNotes(route.notes).
		Save(ctx)
	if err != nil {
		if dbent.IsConstraintError(err) {
			return nil
		}
		return fmt.Errorf("create default relay route %s/%s: %w", route.publicModel, route.targetPlatform, err)
	}
	return nil
}

func createStandardRelayGroupIfNotExists(ctx context.Context, client *dbent.Client, name, platform string, rateMultiplier float64) error {
	exists, err := client.Group.Query().
		Where(group.NameEQ(name), group.DeletedAtIsNil()).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check group exists %s: %w", name, err)
	}
	if exists {
		return nil
	}

	_, err = client.Group.Create().
		SetName(name).
		SetDescription(standardRelayDefaultGroupDescription).
		SetPlatform(platform).
		SetStatus(service.StatusActive).
		SetSubscriptionType(service.SubscriptionTypeStandard).
		SetRateMultiplier(rateMultiplier).
		SetIsExclusive(false).
		Save(ctx)
	if err != nil {
		if dbent.IsConstraintError(err) {
			return nil
		}
		return fmt.Errorf("create default relay group %s: %w", name, err)
	}
	return nil
}

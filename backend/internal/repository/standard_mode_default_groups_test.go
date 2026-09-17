package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type standardRelayRouteRepoStub struct {
	service.CompositeModelRouteRepository
	routes []service.CompositeModelRoute
}

func (s standardRelayRouteRepoStub) ListByGroup(_ context.Context, groupID int64, _ bool) ([]service.CompositeModelRoute, error) {
	routes := make([]service.CompositeModelRoute, 0, len(s.routes))
	for _, route := range s.routes {
		if route.GroupID == groupID {
			routes = append(routes, route)
		}
	}
	return routes, nil
}

func TestStandardRelayDefaultRoutesKeepGPT6ExecutionOnOpenAI(t *testing.T) {
	const groupID int64 = 7
	repo := standardRelayRouteRepoStub{}
	for i, route := range standardRelayDefaultRoutes(groupID) {
		require.Equal(t, groupID, route.groupID)
		repo.routes = append(repo.routes, service.CompositeModelRoute{
			ID: int64(i + 1), GroupID: route.groupID,
			PublicModel: route.publicModel, MatchType: route.matchType,
			TargetPlatform: route.targetPlatform, UpstreamModel: route.upstreamModel,
			Priority: route.priority, Endpoint: service.CompositeRouteEndpointAny, Enabled: true,
		})
	}
	require.Len(t, repo.routes, 6)
	resolver := service.NewCompositeRouteResolver(repo)
	for _, model := range []string{"gpt-6", "gpt-6-astra", "deepseek-v4-pro"} {
		for _, endpoint := range []string{service.CompositeRouteEndpointResponses, service.CompositeRouteEndpointChatCompletions, service.CompositeRouteEndpointMessages} {
			t.Run(model+"/"+endpoint, func(t *testing.T) {
				candidates, err := resolver.ResolveCandidates(context.Background(), groupID, model, endpoint)
				require.NoError(t, err)
				require.Len(t, candidates, 2)
				if service.IsGPT6Model(model) {
					require.Equal(t, service.PlatformOpenAI, candidates[0].TargetPlatform)
					require.Equal(t, "gpt-6-astra", candidates[0].UpstreamModel)
					require.Equal(t, service.PlatformDeepseek, candidates[1].TargetPlatform)
					require.Equal(t, "deepseek-v4-pro", candidates[1].UpstreamModel)
				} else {
					require.Equal(t, service.PlatformDeepseek, candidates[0].TargetPlatform)
					require.Equal(t, "deepseek-v4-pro", candidates[0].UpstreamModel)
					require.Equal(t, service.PlatformOpenAI, candidates[1].TargetPlatform)
				}
				for _, candidate := range candidates {
					require.Equal(t, model, candidate.PublicModel)
					require.Equal(t, service.CompositeRouteSourceExplicit, candidate.Source)
				}
			})
		}
	}
	decision, err := resolver.Resolve(context.Background(), groupID+1, "gpt-6-astra", service.CompositeRouteEndpointResponses)
	require.NoError(t, err)
	require.Equal(t, service.PlatformOpenAI, decision.TargetPlatform, "default routes must not affect other groups")
}

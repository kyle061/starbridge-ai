package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompositeRouteFailoverMigrationAllowsMultipleTargets(t *testing.T) {
	content, err := FS.ReadFile("239_composite_route_failover.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "DROP INDEX IF EXISTS idx_composite_model_routes_unique_active")
	require.Contains(t, sql,
		"ON composite_model_routes ( group_id, endpoint, match_type, public_model, target_platform, upstream_model ) WHERE deleted_at IS NULL")
}

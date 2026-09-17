package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStarbridgeDeepSeekPrimaryMigrationPreservesCustomRoutesAndUsage(t *testing.T) {
	content, err := FS.ReadFile("245_starbridge_deepseek_primary.sql")
	require.NoError(t, err)
	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "SET priority = 10, notes = 'Primary OpenAI GPT6 execution route'")
	require.Contains(t, sql, "SET priority = 20, notes = 'DeepSeek requirements preparation fallback'")
	for _, guard := range []string{
		"relay.name = 'composite-default'", "relay.platform = 'composite'",
		"relay.description = 'Default OpenAI + DeepSeek relay'", "relay.deleted_at IS NULL",
		"route.deleted_at IS NULL", "route.enabled = true", "route.public_model IN ('gpt-6', 'gpt-6-astra')",
		"route.match_type = 'exact'", "route.endpoint = 'any'",
	} {
		require.Equal(t, 2, strings.Count(sql, guard), "both updates must guard %s", guard)
	}
	require.Contains(t, sql, "route.target_platform = 'openai' AND route.upstream_model = 'gpt-6-astra'")
	require.Contains(t, sql, "route.target_platform = 'deepseek' AND route.upstream_model = 'deepseek-v4-pro'")
	require.NotContains(t, sql, "UPDATE usage_logs")
}

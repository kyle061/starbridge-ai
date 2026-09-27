package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemoveGPT6PreparationFallbackMigrationOnlyRemovesSeededRoutes(t *testing.T) {
	content, err := FS.ReadFile("256_remove_gpt6_preparation_fallback.sql")
	require.NoError(t, err)
	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "SET enabled = false, deleted_at = NOW(), updated_at = NOW()")
	require.Contains(t, sql, "relay.name = 'composite-default'")
	require.Contains(t, sql, "relay.description = 'Default OpenAI + DeepSeek relay'")
	require.Contains(t, sql, "route.public_model IN ('gpt-6', 'gpt-6-astra')")
	require.Contains(t, sql, "route.target_platform = 'deepseek'")
	require.Contains(t, sql, "route.notes = 'DeepSeek requirements preparation fallback'")
	require.NotContains(t, sql, "UPDATE usage_logs")
}

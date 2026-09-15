package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStarbridgeMVPRegistrationMigration(t *testing.T) {
	content, err := FS.ReadFile("240_enable_starbridge_mvp_registration.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "UPDATE settings SET value = 'true', updated_at = NOW()")
	require.Contains(t, sql, "WHERE key = 'registration_enabled' AND value IS DISTINCT FROM 'true'")
	require.NotContains(t, sql, "INSERT INTO settings")
}

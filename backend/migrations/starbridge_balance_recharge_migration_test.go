package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStarbridgeBalanceRechargeMigrationReenablesBalanceOrders(t *testing.T) {
	content, err := FS.ReadFile("247_enable_balance_recharge.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "('BALANCE_PAYMENT_DISABLED', 'false')")
	require.Contains(t, sql, "WHERE EXISTS (SELECT 1 FROM users)")
	require.Contains(t, sql, "ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value")
	require.NotContains(t, sql, "'BALANCE_PAYMENT_DISABLED', 'true'")
}

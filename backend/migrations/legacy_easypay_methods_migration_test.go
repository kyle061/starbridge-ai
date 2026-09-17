package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLegacyEasyPayMethodsMigrationRestoresBuiltinMethods(t *testing.T) {
	content, err := FS.ReadFile("248_repair_legacy_easypay_methods.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "SET supported_types = 'alipay,wxpay'")
	require.Contains(t, sql, "WHERE provider_key = 'easypay'")
	require.Contains(t, sql, "TRIM(supported_types) = ''")
	require.Contains(t, sql, "LOWER(TRIM(supported_types)) = 'easypay'")
}

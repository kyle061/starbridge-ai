package migrations

import (
	"strings"
	"testing"
)

func TestLegacyPaymentMethodsMigrationRepairsVisibleRouting(t *testing.T) {
	content, err := FS.ReadFile("249_enable_legacy_payment_methods.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := strings.ToLower(string(content))
	for _, fragment := range []string{
		"payment_visible_method_alipay_source",
		"payment_visible_method_wxpay_source",
		"easypay_alipay",
		"easypay_wxpay",
		"payment_visible_method_alipay_enabled",
		"payment_visible_method_wxpay_enabled",
		"payment_provider_instances",
		"on conflict (key)",
	} {
		if !strings.Contains(sql, strings.ToLower(fragment)) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
}

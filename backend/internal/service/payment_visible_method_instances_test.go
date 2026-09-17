package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

func TestEnsureVisibleMethodRoutingForProviderRepairsEasyPayDefaults(t *testing.T) {
	t.Parallel()

	repo := &paymentConfigSettingRepoStub{values: map[string]string{}}
	svc := &PaymentConfigService{settingRepo: repo}

	if err := svc.ensureVisibleMethodRoutingForProvider(context.Background(), payment.TypeEasyPay, ""); err != nil {
		t.Fatalf("ensureVisibleMethodRoutingForProvider: %v", err)
	}

	for key, want := range map[string]string{
		SettingPaymentVisibleMethodAlipayEnabled: "true",
		SettingPaymentVisibleMethodAlipaySource:  VisibleMethodSourceEasyPayAlipay,
		SettingPaymentVisibleMethodWxpayEnabled:  "true",
		SettingPaymentVisibleMethodWxpaySource:   VisibleMethodSourceEasyPayWechat,
	} {
		if got := repo.values[key]; got != want {
			t.Fatalf("setting %s = %q, want %q", key, got, want)
		}
	}
}

func TestEnsureVisibleMethodRoutingForProviderPreservesExistingRoute(t *testing.T) {
	t.Parallel()

	repo := &paymentConfigSettingRepoStub{values: map[string]string{
		SettingPaymentVisibleMethodAlipayEnabled: "false",
		SettingPaymentVisibleMethodAlipaySource:  VisibleMethodSourceOfficialAlipay,
		SettingPaymentVisibleMethodWxpayEnabled:  "false",
		SettingPaymentVisibleMethodWxpaySource:   "",
	}}
	svc := &PaymentConfigService{settingRepo: repo}

	if err := svc.ensureVisibleMethodRoutingForProvider(context.Background(), payment.TypeEasyPay, "alipay,wxpay"); err != nil {
		t.Fatalf("ensureVisibleMethodRoutingForProvider: %v", err)
	}

	if got := repo.values[SettingPaymentVisibleMethodAlipayEnabled]; got != "false" {
		t.Fatalf("alipay enabled = %q, want false", got)
	}
	if got := repo.values[SettingPaymentVisibleMethodAlipaySource]; got != VisibleMethodSourceOfficialAlipay {
		t.Fatalf("alipay source = %q, want official route", got)
	}
	if got := repo.values[SettingPaymentVisibleMethodWxpayEnabled]; got != "true" {
		t.Fatalf("wxpay enabled = %q, want true", got)
	}
	if got := repo.values[SettingPaymentVisibleMethodWxpaySource]; got != VisibleMethodSourceEasyPayWechat {
		t.Fatalf("wxpay source = %q, want EasyPay route", got)
	}
}

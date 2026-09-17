package provider

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

func newIPv4EasyPayTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for EasyPay test server: %v", err)
	}
	server := &httptest.Server{Listener: listener, Config: &http.Server{Handler: handler}}
	server.Start()
	t.Cleanup(server.Close)
	return server
}

func TestEasyPayClassicMAPIReceivesRechargeAndSubscriptionRequests(t *testing.T) {
	t.Parallel()

	const (
		pid       = "merchant-123"
		pkey      = "merchant-secret"
		notifyURL = "https://merchant.example/payment/webhook/easypay"
		returnURL = "https://merchant.example/payment/result"
	)

	tests := []struct {
		name        string
		orderID     string
		paymentType string
		subject     string
		amount      string
		isMobile    bool
	}{
		{
			name:        "balance recharge",
			orderID:     "sub2_balance_1001",
			paymentType: payment.TypeAlipay,
			subject:     "Starbridge AI 10.00 CNY",
			amount:      "10.00",
		},
		{
			name:        "subscription",
			orderID:     "sub2_subscription_2001",
			paymentType: payment.TypeWxpay,
			subject:     "Starbridge AI Pro 90 days",
			amount:      "15.00",
			isMobile:    true,
		},
	}

	server := newIPv4EasyPayTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mapi.php" {
			t.Errorf("request path = %q, want /mapi.php", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("request method = %q, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Errorf("content type = %q, want application/x-www-form-urlencoded", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
			return
		}

		form := r.PostForm
		for key, want := range map[string]string{
			"pid":        pid,
			"notify_url": notifyURL,
			"return_url": returnURL,
		} {
			if got := form.Get(key); got != want {
				t.Errorf("form[%s] = %q, want %q", key, got, want)
			}
		}
		if got := form.Get("type"); got != payment.TypeAlipay && got != payment.TypeWxpay {
			t.Errorf("form[type] = %q, want alipay or wxpay", got)
		}
		if form.Get("out_trade_no") == "" || form.Get("money") == "" || form.Get("name") == "" {
			t.Errorf("missing required order fields: %v", form)
		}
		if form.Get("sign_type") != signTypeMD5 || form.Get("sign") == "" {
			t.Errorf("missing classic signature fields: %v", form)
		} else {
			withoutSign := make(map[string]string, len(form))
			for key := range form {
				withoutSign[key] = form.Get(key)
			}
			if got, want := form.Get("sign"), easyPaySign(withoutSign, pkey); got != want {
				t.Errorf("form[sign] = %q, want %q", got, want)
			}
		}

		if form.Get("type") == payment.TypeWxpay && form.Get("device") != "mobile" {
			t.Errorf("mobile wxpay device = %q, want mobile", form.Get("device"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":     easypayCodeSuccess,
			"msg":      "ok",
			"trade_no": "trade-" + form.Get("out_trade_no"),
			"payurl":   "/pay/" + form.Get("out_trade_no"),
			"qrcode":   "/qr/" + form.Get("out_trade_no"),
		})
	}))

	provider, err := NewEasyPay("instance-123", map[string]string{
		"pid":       pid,
		"pkey":      pkey,
		"apiBase":   server.URL + "/mapi.php",
		"notifyUrl": notifyURL,
		"returnUrl": returnURL,
	})
	if err != nil {
		t.Fatalf("NewEasyPay: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := provider.CreatePayment(context.Background(), payment.CreatePaymentRequest{
				OrderID:     tt.orderID,
				Amount:      tt.amount,
				PaymentType: tt.paymentType,
				Subject:     tt.subject,
				IsMobile:    tt.isMobile,
			})
			if err != nil {
				t.Fatalf("CreatePayment: %v", err)
			}
			if response.TradeNo != "trade-"+tt.orderID {
				t.Fatalf("TradeNo = %q, want trade-%s", response.TradeNo, tt.orderID)
			}
			if want := server.URL + "/pay/" + tt.orderID; response.PayURL != want {
				t.Fatalf("PayURL = %q, want %q", response.PayURL, want)
			}
			if want := server.URL + "/qr/" + tt.orderID; response.QRCode != want {
				t.Fatalf("QRCode = %q, want %q", response.QRCode, want)
			}
		})
	}
}

func TestEasyPayClassicMAPIUsesRequestCallbackURLs(t *testing.T) {
	t.Parallel()

	var got url.Values
	server := newIPv4EasyPayTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
			return
		}
		got = r.PostForm
		_, _ = w.Write([]byte(`{"code":1,"trade_no":"trade-override","payurl":"https://pay.example/checkout"}`))
	}))

	provider, err := NewEasyPay("instance-override", map[string]string{
		"pid":       "pid-1",
		"pkey":      "key-1",
		"apiBase":   server.URL,
		"notifyUrl": "https://config.example/notify",
		"returnUrl": "https://config.example/return",
	})
	if err != nil {
		t.Fatalf("NewEasyPay: %v", err)
	}

	_, err = provider.CreatePayment(context.Background(), payment.CreatePaymentRequest{
		OrderID:     "request-url-order",
		Amount:      "2.00",
		PaymentType: payment.TypeAlipay,
		Subject:     "Request callback URLs",
		NotifyURL:   "https://request.example/notify",
		ReturnURL:   "https://request.example/return",
	})
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}
	if got.Get("notify_url") != "https://request.example/notify" || got.Get("return_url") != "https://request.example/return" {
		t.Fatalf("request callback URLs = %v, want request URLs", got)
	}
}

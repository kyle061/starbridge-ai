package provider

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

func TestEzfpSignaturePayloadSortsAndExcludesProtocolFields(t *testing.T) {
	t.Parallel()

	params := map[string]string{
		"z":         "last",
		"a":         "first",
		"empty":     "",
		"sign":      "ignored",
		"sign_type": "RSA",
	}

	if got, want := ezfpSignaturePayload(params), "a=first&z=last"; got != want {
		t.Fatalf("ezfpSignaturePayload() = %q, want %q", got, want)
	}
}

func TestEasyPayEzfpCreatePayment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		payType    string
		payInfo    string
		wantQRCode string
		wantPayURL string
	}{
		{
			name:       "qrcode",
			payType:    "qrcode",
			payInfo:    "/api/pay/toapp/order-1",
			wantQRCode: "relative",
		},
		{
			name:       "jump",
			payType:    "jump",
			payInfo:    "weixin://wxpay/bizpayurl?pr=ORDER1",
			wantPayURL: "weixin://wxpay/bizpayurl?pr=ORDER1",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider, _, serverURL := newEzfpTestProvider(t, func(w http.ResponseWriter, r *http.Request, privateKey *rsa.PrivateKey) {
				if r.URL.Path != ezfpCreatePath {
					t.Errorf("path = %q, want %q", r.URL.Path, ezfpCreatePath)
				}
				params := parseAndVerifyEzfpRequest(t, r, &privateKey.PublicKey)
				if params["pid"] != "pid-1" || params["type"] != payment.TypeAlipay {
					t.Errorf("request identity = %v, want pid-1/alipay", params)
				}
				if params["method"] != "web" || params["device"] != "pc" || params["channel_id"] != "channel-alipay" {
					t.Errorf("request routing = %v, want web/pc/channel-alipay", params)
				}
				body := signedEzfpJSON(t, privateKey, map[string]string{
					"code":     "0",
					"trade_no": "trade-1",
					"pay_type": tt.payType,
					"pay_info": tt.payInfo,
				})
				_, _ = w.Write(body)
			})

			response, err := provider.CreatePayment(context.Background(), payment.CreatePaymentRequest{
				OrderID:     "order-1",
				Amount:      "12.34",
				PaymentType: payment.TypeAlipay,
				Subject:     "Test recharge",
				ClientIP:    "127.0.0.1",
			})
			if err != nil {
				t.Fatalf("CreatePayment() error = %v", err)
			}
			if response.TradeNo != "trade-1" {
				t.Fatalf("TradeNo = %q, want trade-1", response.TradeNo)
			}
			if tt.wantQRCode == "relative" {
				if want := serverURL + "/api/pay/toapp/order-1"; response.QRCode != want {
					t.Fatalf("QRCode = %q, want %q", response.QRCode, want)
				}
				if response.PayURL != "" {
					t.Fatalf("PayURL = %q, want empty", response.PayURL)
				}
			} else if response.PayURL != tt.wantPayURL {
				t.Fatalf("PayURL = %q, want %q", response.PayURL, tt.wantPayURL)
			}
		})
	}
}

func TestEasyPayEzfpQueryOrderStatusMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		status     string
		wantStatus string
	}{
		{name: "paid", status: "1", wantStatus: payment.ProviderStatusPaid},
		{name: "refunded", status: "2", wantStatus: payment.ProviderStatusRefunded},
		{name: "pending", status: "0", wantStatus: payment.ProviderStatusPending},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			provider, _, _ := newEzfpTestProvider(t, func(w http.ResponseWriter, r *http.Request, privateKey *rsa.PrivateKey) {
				if r.URL.Path != ezfpQueryPath {
					t.Errorf("path = %q, want %q", r.URL.Path, ezfpQueryPath)
				}
				params := parseAndVerifyEzfpRequest(t, r, &privateKey.PublicKey)
				if params["out_trade_no"] != "order-query" {
					t.Errorf("out_trade_no = %q, want order-query", params["out_trade_no"])
				}
				_, _ = w.Write(signedEzfpJSON(t, privateKey, map[string]string{
					"code":     "0",
					"status":   tt.status,
					"trade_no": "trade-query",
					"money":    "3.21",
				}))
			})

			response, err := provider.QueryOrder(context.Background(), "order-query")
			if err != nil {
				t.Fatalf("QueryOrder() error = %v", err)
			}
			if response.Status != tt.wantStatus {
				t.Fatalf("Status = %q, want %q", response.Status, tt.wantStatus)
			}
			if response.TradeNo != "trade-query" || response.Amount != 3.21 {
				t.Fatalf("response = %+v, want trade-query/3.21", response)
			}
		})
	}
}

func TestEasyPayEzfpVerifyNotification(t *testing.T) {
	t.Parallel()

	provider, privateKey, _ := newEzfpTestProvider(t, nil)
	params := map[string]string{
		"pid":          "pid-1",
		"trade_no":     "trade-notify",
		"out_trade_no": "order-notify",
		"type":         payment.TypeWxpay,
		"trade_status": tradeStatusSuccess,
		"money":        "8.88",
		"timestamp":    "1700000000",
	}

	notification, err := provider.VerifyNotification(context.Background(), signedEzfpValues(t, privateKey, params), nil)
	if err != nil {
		t.Fatalf("VerifyNotification() error = %v", err)
	}
	if notification.Status != payment.ProviderStatusSuccess || notification.OrderID != "order-notify" || notification.Amount != 8.88 {
		t.Fatalf("notification = %+v, want successful order-notify/8.88", notification)
	}

	params["pid"] = "another-pid"
	if _, err := provider.VerifyNotification(context.Background(), signedEzfpValues(t, privateKey, params), nil); err == nil {
		t.Fatal("VerifyNotification() accepted a notification for another pid")
	}
}

func TestEasyPayEzfpRefundAndRefundQuery(t *testing.T) {
	t.Parallel()

	provider, _, _ := newEzfpTestProvider(t, func(w http.ResponseWriter, r *http.Request, privateKey *rsa.PrivateKey) {
		params := parseAndVerifyEzfpRequest(t, r, &privateKey.PublicKey)
		switch r.URL.Path {
		case ezfpRefundPath:
			if params["out_trade_no"] != "order-refund" || params["money"] != "2.50" || params["out_refund_no"] == "" {
				t.Errorf("refund request = %v, want order-refund/2.50 with refund number", params)
			}
			_, _ = w.Write(signedEzfpJSON(t, privateKey, map[string]string{
				"code":      "0",
				"refund_no": "refund-1",
			}))
		case ezfpRefundQueryPath:
			if params["refund_no"] != "refund-1" {
				t.Errorf("refund_no = %q, want refund-1", params["refund_no"])
			}
			_, _ = w.Write(signedEzfpJSON(t, privateKey, map[string]string{
				"code":      "0",
				"status":    "1",
				"refund_no": "refund-1",
			}))
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	})

	refund, err := provider.Refund(context.Background(), payment.RefundRequest{
		OrderID: "order-refund",
		Amount:  "2.50",
	})
	if err != nil {
		t.Fatalf("Refund() error = %v", err)
	}
	if refund.RefundID != "refund-1" || refund.Status != payment.ProviderStatusSuccess {
		t.Fatalf("refund = %+v, want refund-1/success", refund)
	}

	queried, err := provider.QueryRefund(context.Background(), payment.RefundQueryRequest{RefundID: "refund-1"})
	if err != nil {
		t.Fatalf("QueryRefund() error = %v", err)
	}
	if queried.RefundID != "refund-1" || queried.Status != payment.ProviderStatusSuccess {
		t.Fatalf("refund query = %+v, want refund-1/success", queried)
	}
}

func TestEasyPayEzfpSupportedTypesExcludeLegacyCustomMethods(t *testing.T) {
	t.Parallel()

	provider, _, _ := newEzfpTestProvider(t, nil)
	provider.config["customMethods"] = `[{"type":"legacy_card","upstreamType":"card"}]`

	got := provider.SupportedTypes()
	if strings.Join(got, ",") != payment.TypeAlipay+","+payment.TypeWxpay {
		t.Fatalf("SupportedTypes() = %v, want only alipay and wxpay", got)
	}
}

func newEzfpTestProvider(t *testing.T, handler func(http.ResponseWriter, *http.Request, *rsa.PrivateKey)) (*EasyPay, *rsa.PrivateKey, string) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handler == nil {
			t.Errorf("unexpected request to %s", r.URL.Path)
			return
		}
		handler(w, r, privateKey)
	}))
	t.Cleanup(server.Close)

	privatePEM, err := marshalEzfpPrivateKey(privateKey)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	publicPEM, err := marshalEzfpPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	provider, err := NewEasyPay("ezfp-test", map[string]string{
		"pid":        "pid-1",
		"privateKey": privatePEM,
		"publicKey":  publicPEM,
		"apiBase":    server.URL,
		"notifyUrl":  "https://merchant.example/notify",
		"returnUrl":  "https://merchant.example/return",
		"cidAlipay":  "channel-alipay",
	})
	if err != nil {
		t.Fatalf("NewEasyPay: %v", err)
	}
	return provider, privateKey, server.URL
}

func marshalEzfpPrivateKey(key *rsa.PrivateKey) (string, error) {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})), nil
}

func marshalEzfpPublicKey(key *rsa.PublicKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})), nil
}

func parseAndVerifyEzfpRequest(t *testing.T, r *http.Request, publicKey *rsa.PublicKey) map[string]string {
	t.Helper()
	if err := r.ParseForm(); err != nil {
		t.Errorf("ParseForm: %v", err)
		return nil
	}
	params := make(map[string]string, len(r.PostForm))
	for key := range r.PostForm {
		params[key] = r.PostForm.Get(key)
	}
	decoded, err := base64.StdEncoding.DecodeString(params["sign"])
	if err != nil {
		t.Errorf("decode request signature: %v", err)
		return params
	}
	digest := sha256.Sum256([]byte(ezfpSignaturePayload(params)))
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], decoded); err != nil {
		t.Errorf("request signature verification failed: %v", err)
	}
	return params
}

func signedEzfpJSON(t *testing.T, privateKey *rsa.PrivateKey, fields map[string]string) []byte {
	t.Helper()
	params := cloneStringMap(fields)
	params["sign_type"] = ezfpSignTypeRSA
	digest := sha256.Sum256([]byte(ezfpSignaturePayload(params)))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("sign response: %v", err)
	}
	params["sign"] = base64.StdEncoding.EncodeToString(signature)
	body, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	return body
}

func signedEzfpValues(t *testing.T, privateKey *rsa.PrivateKey, fields map[string]string) string {
	t.Helper()
	params := cloneStringMap(fields)
	params["sign_type"] = ezfpSignTypeRSA
	digest := sha256.Sum256([]byte(ezfpSignaturePayload(params)))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("sign notification: %v", err)
	}
	params["sign"] = base64.StdEncoding.EncodeToString(signature)
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	return values.Encode()
}

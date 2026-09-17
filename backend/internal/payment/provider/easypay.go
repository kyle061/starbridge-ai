// Package provider contains concrete payment provider implementations.
package provider

import (
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

// EasyPay constants.
const (
	easypayCodeSuccess     = 1
	easypayStatusPaid      = 1
	ezfpCodeSuccess        = 0
	easypayHTTPTimeout     = 10 * time.Second
	maxEasypayResponseSize = 1 << 20 // 1MB
	maxEasypayErrorSummary = 512
	tradeStatusSuccess     = "TRADE_SUCCESS"
	signTypeMD5            = "MD5"
	ezfpSignTypeRSA        = "RSA"
	ezfpSubmitPath         = "/api/pay/submit"
	ezfpCreatePath         = "/api/pay/create"
	ezfpQueryPath          = "/api/pay/query"
	ezfpRefundPath         = "/api/pay/refund"
	ezfpRefundQueryPath    = "/api/pay/refundquery"
	paymentModePopup       = "popup"
	deviceMobile           = "mobile"
)

// EasyPay implements payment.Provider for the EasyPay aggregation platform.
type EasyPay struct {
	instanceID string
	config     map[string]string
	httpClient *http.Client
	legacy     bool
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

type easyPayCustomMethod struct {
	Type         string `json:"type"`
	UpstreamType string `json:"upstreamType"`
	DisplayName  string `json:"displayName"`
}

// NewEasyPay creates a new EasyPay provider.
//
// New ezfp configurations use pid, privateKey, publicKey, apiBase, notifyUrl,
// and returnUrl. pkey remains accepted as a legacy MD5 configuration so
// existing merchants can migrate without breaking pending orders.
func NewEasyPay(instanceID string, config map[string]string) (*EasyPay, error) {
	for _, k := range []string{"pid", "apiBase", "notifyUrl", "returnUrl"} {
		if strings.TrimSpace(config[k]) == "" {
			return nil, fmt.Errorf("easypay config missing required key: %s", k)
		}
	}
	cfg := make(map[string]string, len(config))
	for k, v := range config {
		cfg[k] = v
	}
	cfg["apiBase"] = normalizeEasyPayAPIBase(cfg["apiBase"])
	provider := &EasyPay{
		instanceID: instanceID,
		config:     cfg,
		httpClient: &http.Client{Timeout: easypayHTTPTimeout},
	}
	privateKeyValue := strings.TrimSpace(config["privateKey"])
	publicKeyValue := strings.TrimSpace(config["publicKey"])
	if privateKeyValue == "" && publicKeyValue == "" {
		if strings.TrimSpace(config["pkey"]) == "" {
			return nil, fmt.Errorf("easypay config missing required key: privateKey")
		}
		provider.legacy = true
		return provider, nil
	}
	if privateKeyValue == "" || publicKeyValue == "" {
		return nil, fmt.Errorf("easypay config requires both privateKey and publicKey")
	}
	var err error
	provider.privateKey, err = parseEasyPayPrivateKey(privateKeyValue)
	if err != nil {
		return nil, fmt.Errorf("easypay privateKey: %w", err)
	}
	provider.publicKey, err = parseEasyPayPublicKey(publicKeyValue)
	if err != nil {
		return nil, fmt.Errorf("easypay publicKey: %w", err)
	}
	cfg["privateKey"] = privateKeyValue
	cfg["publicKey"] = publicKeyValue
	return provider, nil
}

func normalizeEasyPayAPIBase(apiBase string) string {
	base := strings.TrimSpace(apiBase)
	if base == "" {
		return ""
	}
	if parsed, err := url.Parse(base); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		parsed.RawQuery = ""
		parsed.Fragment = ""
		parsed.RawPath = ""
		parsed.Path = trimEasyPayEndpointPath(parsed.Path)
		return strings.TrimRight(parsed.String(), "/")
	}
	return strings.TrimRight(trimEasyPayEndpointPath(base), "/")
}

func trimEasyPayEndpointPath(path string) string {
	path = strings.TrimRight(strings.TrimSpace(path), "/")
	lower := strings.ToLower(path)
	for _, endpoint := range []string{
		"/api/pay/submit", "/api/pay/create", "/api/pay/query", "/api/pay/refund",
		"/api/pay/refundquery", "/submit.php", "/mapi.php", "/api.php",
	} {
		if strings.HasSuffix(lower, endpoint) {
			return strings.TrimRight(path[:len(path)-len(endpoint)], "/")
		}
	}
	return path
}

func (e *EasyPay) apiBase() string {
	if e == nil {
		return ""
	}
	return normalizeEasyPayAPIBase(e.config["apiBase"])
}

func (e *EasyPay) Name() string        { return "EasyPay" }
func (e *EasyPay) ProviderKey() string { return payment.TypeEasyPay }
func (e *EasyPay) SupportedTypes() []payment.PaymentType {
	types := []payment.PaymentType{payment.TypeAlipay, payment.TypeWxpay}
	if !e.legacy {
		return types
	}
	for _, method := range e.customMethods() {
		if method.Type != "" {
			types = append(types, method.Type)
		}
	}
	return types
}

func (e *EasyPay) MerchantIdentityMetadata() map[string]string {
	if e == nil {
		return nil
	}
	pid := strings.TrimSpace(e.config["pid"])
	if pid == "" {
		return nil
	}
	return map[string]string{"pid": pid}
}

func (e *EasyPay) CreatePayment(ctx context.Context, req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	if !e.legacy {
		return e.createEzfpPayment(ctx, req)
	}
	// Payment mode determined by instance config, not payment type.
	// "popup" → hosted page (submit.php); "qrcode"/default → API call (mapi.php).
	mode := e.config["paymentMode"]
	if mode == paymentModePopup {
		return e.createRedirectPayment(req)
	}
	return e.createAPIPayment(ctx, req)
}

// createEzfpPayment uses the documented ezfp API. qrcode mode uses the
// server-side create endpoint; popup mode uses the browser-facing submit
// endpoint because that endpoint is designed to be opened directly by a user.
func (e *EasyPay) createEzfpPayment(ctx context.Context, req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	switch payment.GetBasePaymentType(req.PaymentType) {
	case payment.TypeAlipay, payment.TypeWxpay:
		req.PaymentType = payment.GetBasePaymentType(req.PaymentType)
	default:
		return nil, fmt.Errorf("ezfp EasyPay only supports alipay and wxpay")
	}
	if e.config["paymentMode"] == paymentModePopup {
		return e.createEzfpRedirectPayment(req)
	}
	params := e.ezfpPaymentParams(req)
	params["method"] = "web"
	if req.IsMobile {
		params["device"] = deviceMobile
	} else {
		params["device"] = "pc"
	}
	return e.createEzfpPaymentRequest(ctx, params)
}

func (e *EasyPay) createEzfpRedirectPayment(req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	params := e.ezfpPaymentParams(req)
	params["timestamp"] = strconv.FormatInt(time.Now().Unix(), 10)
	params["sign_type"] = ezfpSignTypeRSA
	sign, err := e.signEzfp(params)
	if err != nil {
		return nil, fmt.Errorf("easypay submit sign: %w", err)
	}
	params["sign"] = sign
	query := url.Values{}
	for key, value := range params {
		query.Set(key, value)
	}
	return &payment.CreatePaymentResponse{PayURL: e.ezfpEndpoint(ezfpSubmitPath) + "?" + query.Encode()}, nil
}

func (e *EasyPay) ezfpPaymentParams(req payment.CreatePaymentRequest) map[string]string {
	notifyURL, returnURL := e.resolveURLs(req)
	params := map[string]string{
		"pid":          e.config["pid"],
		"type":         e.upstreamPaymentType(req.PaymentType),
		"out_trade_no": req.OrderID,
		"notify_url":   notifyURL,
		"return_url":   returnURL,
		"name":         req.Subject,
		"money":        req.Amount,
		"clientip":     req.ClientIP,
	}
	if channelID := e.resolveChannelID(params["type"]); channelID != "" {
		params["channel_id"] = channelID
	}
	return params
}

func (e *EasyPay) createEzfpPaymentRequest(ctx context.Context, params map[string]string) (*payment.CreatePaymentResponse, error) {
	params["timestamp"] = strconv.FormatInt(time.Now().Unix(), 10)
	params["sign_type"] = ezfpSignTypeRSA
	sign, err := e.signEzfp(params)
	if err != nil {
		return nil, fmt.Errorf("easypay create sign: %w", err)
	}
	params["sign"] = sign
	body, err := e.post(ctx, e.ezfpEndpoint(ezfpCreatePath), params)
	if err != nil {
		return nil, fmt.Errorf("easypay create: %w", err)
	}
	response, err := e.verifyEzfpResponse(body, "create")
	if err != nil {
		return nil, err
	}
	if code := ezfpResponseCode(response); code != ezfpCodeSuccess {
		return nil, fmt.Errorf("easypay create error: %s", ezfpResponseMessage(response))
	}
	tradeNo := response["trade_no"]
	payInfo := strings.TrimSpace(response["pay_info"])
	switch strings.ToLower(strings.TrimSpace(response["pay_type"])) {
	case "qrcode":
		return &payment.CreatePaymentResponse{TradeNo: tradeNo, QRCode: resolveEasyPayReturnedRef(e.apiBase(), payInfo)}, nil
	case "jump", "urlscheme":
		return &payment.CreatePaymentResponse{TradeNo: tradeNo, PayURL: resolveEasyPayReturnedRef(e.apiBase(), payInfo)}, nil
	case "html":
		if strings.HasPrefix(payInfo, "http://") || strings.HasPrefix(payInfo, "https://") {
			return &payment.CreatePaymentResponse{TradeNo: tradeNo, PayURL: payInfo}, nil
		}
	}
	return nil, fmt.Errorf("easypay create returned unsupported pay_type %q", response["pay_type"])
}

// createRedirectPayment builds a submit.php URL for browser redirect.
// No server-side API call — the user is redirected to EasyPay's hosted page.
// TradeNo is empty; it arrives via the notify callback after payment.
func (e *EasyPay) createRedirectPayment(req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	notifyURL, returnURL := e.resolveURLs(req)
	paymentType := e.upstreamPaymentType(req.PaymentType)
	params := map[string]string{
		"pid": e.config["pid"], "type": paymentType,
		"out_trade_no": req.OrderID, "notify_url": notifyURL,
		"return_url": returnURL, "name": req.Subject,
		"money": req.Amount,
	}
	if cid := e.resolveCID(paymentType); cid != "" {
		params["cid"] = cid
	}
	if req.IsMobile {
		params["device"] = deviceMobile
	}
	params["sign"] = easyPaySign(params, e.config["pkey"])
	params["sign_type"] = signTypeMD5

	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	payURL := e.apiBase() + "/submit.php?" + q.Encode()
	return &payment.CreatePaymentResponse{PayURL: payURL}, nil
}

// createAPIPayment calls mapi.php to get payurl/qrcode (existing behavior).
func (e *EasyPay) createAPIPayment(ctx context.Context, req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	notifyURL, returnURL := e.resolveURLs(req)
	paymentType := e.upstreamPaymentType(req.PaymentType)
	params := map[string]string{
		"pid": e.config["pid"], "type": paymentType,
		"out_trade_no": req.OrderID, "notify_url": notifyURL,
		"return_url": returnURL, "name": req.Subject,
		"money": req.Amount, "clientip": req.ClientIP,
	}
	if cid := e.resolveCID(paymentType); cid != "" {
		params["cid"] = cid
	}
	if req.IsMobile {
		params["device"] = deviceMobile
	}
	params["sign"] = easyPaySign(params, e.config["pkey"])
	params["sign_type"] = signTypeMD5

	body, err := e.post(ctx, e.apiBase()+"/mapi.php", params)
	if err != nil {
		return nil, fmt.Errorf("easypay create: %w", err)
	}
	var resp struct {
		Code    int    `json:"code"`
		Msg     string `json:"msg"`
		TradeNo string `json:"trade_no"`
		PayURL  string `json:"payurl"`
		PayURL2 string `json:"payurl2"` // H5 mobile payment URL
		QRCode  string `json:"qrcode"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("easypay parse: %w", err)
	}
	if resp.Code != easypayCodeSuccess {
		return nil, fmt.Errorf("easypay error: %s", resp.Msg)
	}
	payURL := resp.PayURL
	if req.IsMobile && resp.PayURL2 != "" {
		payURL = resp.PayURL2
	}
	base := e.apiBase()
	return &payment.CreatePaymentResponse{
		TradeNo: resp.TradeNo,
		PayURL:  resolveEasyPayReturnedRef(base, payURL),
		QRCode:  resolveEasyPayReturnedRef(base, resp.QRCode),
	}, nil
}

// resolveEasyPayReturnedRef absolutizes a payurl/payurl2/qrcode reference that
// mapi.php returned, resolving it against the instance's configured apiBase.
//
// EasyPay-compatible upstreams disagree on this field: some answer with a full
// checkout URL, others with a site-root-relative path such as
// "/api/pay/toapp/<order>". createRedirectPayment already builds an absolute URL
// itself (apiBase + "/submit.php?..."), but the mapi.php branch stored whatever
// came back verbatim, and nothing downstream repairs it —
// sanitizeCreatePaymentResponseDetails only strips NUL bytes before the value is
// persisted to pay_url/qr_code. The frontend then feeds qr_code straight into
// QRCode.toCanvas (PaymentQRCodeView.renderQR), so a relative path becomes a QR
// whose payload is a bare path: WeChat renders it as text, and pay_url resolves
// against the gateway's own domain and 404s.
//
// That is precisely the outcome the Alipay provider already refuses to produce —
// "Setting it as QRCode would let the frontend render an unscannable image"
// (createPagePayTrade) — so EasyPay is brought in line with the same rule.
//
// Only references beginning with "/" are resolved. Anything carrying a scheme is
// already actionable and is returned untouched, which covers both absolute
// https:// checkout URLs and app deep links (weixin://, wxp://, alipays://). A
// reference without a leading slash is left alone as well: QR payloads are
// frequently opaque tokens rather than paths, and rewriting "OrderToken123" into
// "<apiBase>/OrderToken123" would break a payload that works today — strictly
// worse than the bug this fixes.
func resolveEasyPayReturnedRef(apiBase, ref string) string {
	trimmed := strings.TrimSpace(ref)
	if !strings.HasPrefix(trimmed, "/") {
		return ref
	}
	base, err := url.Parse(strings.TrimSpace(apiBase))
	if err != nil || base.Scheme == "" || base.Host == "" {
		return ref
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme != "" {
		return ref
	}
	return base.ResolveReference(parsed).String()
}

// resolveURLs returns (notifyURL, returnURL) preferring request values,
// falling back to instance config.
func (e *EasyPay) resolveURLs(req payment.CreatePaymentRequest) (string, string) {
	notifyURL := req.NotifyURL
	if notifyURL == "" {
		notifyURL = e.config["notifyUrl"]
	}
	returnURL := req.ReturnURL
	if returnURL == "" {
		returnURL = e.config["returnUrl"]
	}
	return notifyURL, returnURL
}

func (e *EasyPay) customMethods() []easyPayCustomMethod {
	if e == nil {
		return nil
	}
	raw := strings.TrimSpace(e.config["customMethods"])
	if raw == "" {
		return nil
	}
	var methods []easyPayCustomMethod
	if err := json.Unmarshal([]byte(raw), &methods); err != nil {
		return nil
	}
	result := make([]easyPayCustomMethod, 0, len(methods))
	for _, method := range methods {
		method.Type = strings.TrimSpace(method.Type)
		method.UpstreamType = strings.TrimSpace(method.UpstreamType)
		method.DisplayName = strings.TrimSpace(method.DisplayName)
		if method.Type == "" || method.UpstreamType == "" {
			continue
		}
		result = append(result, method)
	}
	return result
}

func (e *EasyPay) upstreamPaymentType(paymentType string) string {
	paymentType = strings.TrimSpace(paymentType)
	for _, method := range e.customMethods() {
		if paymentType == method.Type {
			return method.UpstreamType
		}
	}
	return paymentType
}

func (e *EasyPay) QueryOrder(ctx context.Context, tradeNo string) (*payment.QueryOrderResponse, error) {
	if !e.legacy {
		return e.queryEzfpOrder(ctx, tradeNo)
	}
	params := map[string]string{
		"act": "order", "pid": e.config["pid"],
		"key": e.config["pkey"], "out_trade_no": tradeNo,
	}
	body, err := e.post(ctx, e.apiBase()+"/api.php", params)
	if err != nil {
		return nil, fmt.Errorf("easypay query: %w", err)
	}
	type easyPayQueryData struct {
		TradeStatus *string `json:"trade_status"`
		Status      *int    `json:"status"`
		Money       *string `json:"money"`
		TradeNo     *string `json:"trade_no"`
	}
	var resp struct {
		Code        int              `json:"code"`
		Msg         string           `json:"msg"`
		TradeStatus *string          `json:"trade_status"`
		Status      *int             `json:"status"`
		Money       *string          `json:"money"`
		TradeNo     *string          `json:"trade_no"`
		Data        easyPayQueryData `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("easypay parse query: %w", err)
	}
	status := payment.ProviderStatusPending
	if resp.TradeStatus != nil {
		if *resp.TradeStatus == tradeStatusSuccess {
			status = payment.ProviderStatusPaid
		}
	} else if resp.Data.TradeStatus != nil {
		if *resp.Data.TradeStatus == tradeStatusSuccess {
			status = payment.ProviderStatusPaid
		}
	} else if resp.Status != nil {
		if *resp.Status == easypayStatusPaid {
			status = payment.ProviderStatusPaid
		}
	} else if resp.Data.Status != nil && *resp.Data.Status == easypayStatusPaid {
		status = payment.ProviderStatusPaid
	}

	money := ""
	if resp.Money != nil {
		money = *resp.Money
	} else if resp.Data.Money != nil {
		money = *resp.Data.Money
	}
	responseTradeNo := tradeNo
	if resp.TradeNo != nil {
		if *resp.TradeNo != "" {
			responseTradeNo = *resp.TradeNo
		}
	} else if resp.Data.TradeNo != nil && *resp.Data.TradeNo != "" {
		responseTradeNo = *resp.Data.TradeNo
	}

	amount, _ := strconv.ParseFloat(money, 64)
	return &payment.QueryOrderResponse{
		TradeNo:  responseTradeNo,
		Status:   status,
		Amount:   amount,
		Metadata: e.MerchantIdentityMetadata(),
	}, nil
}

func (e *EasyPay) queryEzfpOrder(ctx context.Context, tradeNo string) (*payment.QueryOrderResponse, error) {
	params := map[string]string{
		"pid":          e.config["pid"],
		"out_trade_no": tradeNo,
		"timestamp":    strconv.FormatInt(time.Now().Unix(), 10),
		"sign_type":    ezfpSignTypeRSA,
	}
	sign, err := e.signEzfp(params)
	if err != nil {
		return nil, fmt.Errorf("easypay query sign: %w", err)
	}
	params["sign"] = sign
	body, err := e.post(ctx, e.ezfpEndpoint(ezfpQueryPath), params)
	if err != nil {
		return nil, fmt.Errorf("easypay query: %w", err)
	}
	response, err := e.verifyEzfpResponse(body, "query")
	if err != nil {
		return nil, err
	}
	if ezfpResponseCode(response) != ezfpCodeSuccess {
		return &payment.QueryOrderResponse{
			TradeNo:  tradeNo,
			Status:   payment.ProviderStatusPending,
			Metadata: e.MerchantIdentityMetadata(),
		}, nil
	}
	status := payment.ProviderStatusPending
	queryStatus := strings.ToUpper(strings.TrimSpace(response["status"]))
	tradeStatus := strings.ToUpper(strings.TrimSpace(response["trade_status"]))
	switch queryStatus {
	case "1", "PAID", "TRADE_SUCCESS":
		status = payment.ProviderStatusPaid
	case "2", "REFUNDED", "TRADE_REFUND":
		status = payment.ProviderStatusRefunded
	}
	if tradeStatus == tradeStatusSuccess {
		status = payment.ProviderStatusPaid
	}
	responseTradeNo := strings.TrimSpace(response["trade_no"])
	if responseTradeNo == "" {
		responseTradeNo = tradeNo
	}
	amount, _ := strconv.ParseFloat(response["money"], 64)
	return &payment.QueryOrderResponse{
		TradeNo:  responseTradeNo,
		Status:   status,
		Amount:   amount,
		Metadata: e.MerchantIdentityMetadata(),
	}, nil
}

func (e *EasyPay) VerifyNotification(_ context.Context, rawBody string, _ map[string]string) (*payment.PaymentNotification, error) {
	if !e.legacy {
		return e.verifyEzfpNotification(rawBody)
	}
	values, err := url.ParseQuery(rawBody)
	if err != nil {
		return nil, fmt.Errorf("parse notify: %w", err)
	}
	// url.ParseQuery already decodes values — no additional decode needed.
	params := make(map[string]string)
	for k := range values {
		params[k] = values.Get(k)
	}
	sign := params["sign"]
	if sign == "" {
		return nil, fmt.Errorf("missing sign")
	}
	if !easyPayVerifySign(params, e.config["pkey"], sign) {
		return nil, fmt.Errorf("invalid signature")
	}
	status := payment.ProviderStatusFailed
	if params["trade_status"] == tradeStatusSuccess {
		status = payment.ProviderStatusSuccess
	}
	amount, _ := strconv.ParseFloat(params["money"], 64)

	metadata := e.MerchantIdentityMetadata()
	if pid := strings.TrimSpace(params["pid"]); pid != "" {
		if metadata == nil {
			metadata = map[string]string{}
		}
		metadata["pid"] = pid
	}
	return &payment.PaymentNotification{
		TradeNo: params["trade_no"], OrderID: params["out_trade_no"],
		Amount: amount, Status: status, RawData: rawBody, Metadata: metadata,
	}, nil
}

func (e *EasyPay) verifyEzfpNotification(rawBody string) (*payment.PaymentNotification, error) {
	values, err := url.ParseQuery(rawBody)
	if err != nil {
		return nil, fmt.Errorf("parse notify: %w", err)
	}
	params := make(map[string]string, len(values))
	for key := range values {
		params[key] = values.Get(key)
	}
	if sign := strings.TrimSpace(params["sign"]); sign == "" {
		return nil, fmt.Errorf("missing sign")
	} else if err := e.verifyEzfpSignature(params, sign); err != nil {
		return nil, err
	}
	pid := strings.TrimSpace(params["pid"])
	if pid == "" {
		return nil, fmt.Errorf("easypay notification missing pid")
	}
	if pid != strings.TrimSpace(e.config["pid"]) {
		return nil, fmt.Errorf("easypay notification pid mismatch")
	}
	status := payment.ProviderStatusFailed
	if strings.EqualFold(strings.TrimSpace(params["trade_status"]), tradeStatusSuccess) {
		status = payment.ProviderStatusSuccess
	}
	amount, _ := strconv.ParseFloat(params["money"], 64)
	return &payment.PaymentNotification{
		TradeNo:  params["trade_no"],
		OrderID:  params["out_trade_no"],
		Amount:   amount,
		Status:   status,
		RawData:  rawBody,
		Metadata: e.MerchantIdentityMetadata(),
	}, nil
}

func (e *EasyPay) Refund(ctx context.Context, req payment.RefundRequest) (*payment.RefundResponse, error) {
	if !e.legacy {
		return e.refundEzfp(ctx, req)
	}
	attempts := e.refundAttempts(req)
	if len(attempts) == 0 {
		return nil, fmt.Errorf("easypay refund missing order identifier")
	}
	var firstErr error
	for i, attempt := range attempts {
		body, status, err := e.postRaw(ctx, e.apiBase()+"/api.php?act=refund", attempt.params)
		if err != nil {
			return nil, fmt.Errorf("easypay refund request: %w", err)
		}
		if err := parseEasyPayRefundResponse(status, body); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			if i+1 < len(attempts) && isEasyPayRefundOrderNotFound(err) {
				continue
			}
			return nil, err
		}
		return &payment.RefundResponse{RefundID: attempt.refundID, Status: payment.ProviderStatusSuccess}, nil
	}
	return nil, firstErr
}

type easyPayRefundAttempt struct {
	params   map[string]string
	refundID string
}

func (e *EasyPay) refundAttempts(req payment.RefundRequest) []easyPayRefundAttempt {
	base := map[string]string{
		"pid": e.config["pid"], "key": e.config["pkey"], "money": req.Amount,
	}
	var attempts []easyPayRefundAttempt
	if orderID := strings.TrimSpace(req.OrderID); orderID != "" {
		params := cloneStringMap(base)
		params["out_trade_no"] = orderID
		attempts = append(attempts, easyPayRefundAttempt{params: params, refundID: orderID})
	}
	if tradeNo := strings.TrimSpace(req.TradeNo); tradeNo != "" {
		params := cloneStringMap(base)
		params["trade_no"] = tradeNo
		attempts = append(attempts, easyPayRefundAttempt{params: params, refundID: tradeNo})
	}
	return attempts
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func isEasyPayRefundOrderNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	return strings.Contains(msg, "订单编号不存在") ||
		strings.Contains(msg, "订单不存在") ||
		strings.Contains(lower, "order not found") ||
		strings.Contains(lower, "not exist")
}

func parseEasyPayRefundResponse(status int, body []byte) error {
	summary := summarizeEasyPayResponse(body)
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return fmt.Errorf("easypay refund HTTP %d: %s", status, summary)
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return fmt.Errorf("easypay refund empty response (HTTP %d): %s", status, summary)
	}

	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "<!doctype html") || strings.HasPrefix(lower, "<html") ||
		(strings.HasPrefix(lower, "<") && strings.Contains(lower, "html")) {
		return fmt.Errorf("easypay refund non-JSON response (HTTP %d): %s", status, summary)
	}

	var resp struct {
		Code any    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("easypay refund non-JSON response (HTTP %d): %s", status, summary)
	}
	if !easyPayResponseCodeIsSuccess(resp.Code) {
		msg := strings.TrimSpace(resp.Msg)
		if msg == "" {
			msg = summary
		}
		return fmt.Errorf("easypay refund failed (HTTP %d): %s", status, msg)
	}
	return nil
}

func easyPayResponseCodeIsSuccess(code any) bool {
	switch v := code.(type) {
	case float64:
		return int(v) == easypayCodeSuccess
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		return err == nil && n == easypayCodeSuccess
	default:
		return false
	}
}

func summarizeEasyPayResponse(body []byte) string {
	summary := strings.Join(strings.Fields(string(body)), " ")
	if summary == "" {
		return "<empty>"
	}
	if len(summary) > maxEasypayErrorSummary {
		truncated := summary[:maxEasypayErrorSummary]
		for len(truncated) > 0 && !utf8.ValidString(truncated) {
			truncated = truncated[:len(truncated)-1]
		}
		return truncated + "..."
	}
	return summary
}

func (e *EasyPay) resolveCID(paymentType string) string {
	if strings.HasPrefix(paymentType, "alipay") {
		if v := e.config["cidAlipay"]; v != "" {
			return v
		}
		return e.config["cid"]
	}
	if v := e.config["cidWxpay"]; v != "" {
		return v
	}
	return e.config["cid"]
}

func (e *EasyPay) resolveChannelID(paymentType string) string {
	if value := strings.TrimSpace(e.config["channelId"]); value != "" {
		return value
	}
	if strings.HasPrefix(paymentType, "alipay") {
		if value := strings.TrimSpace(e.config["cidAlipay"]); value != "" {
			return value
		}
	} else if strings.HasPrefix(paymentType, "wxpay") {
		if value := strings.TrimSpace(e.config["cidWxpay"]); value != "" {
			return value
		}
	}
	return strings.TrimSpace(e.config["cid"])
}

func (e *EasyPay) ezfpEndpoint(path string) string {
	return strings.TrimRight(e.apiBase(), "/") + path
}

type ezfpRefundAttempt struct {
	identifierKey string
	identifier    string
	refundID      string
}

func (e *EasyPay) refundEzfp(ctx context.Context, req payment.RefundRequest) (*payment.RefundResponse, error) {
	attempts := make([]ezfpRefundAttempt, 0, 2)
	if orderID := strings.TrimSpace(req.OrderID); orderID != "" {
		attempts = append(attempts, ezfpRefundAttempt{
			identifierKey: "out_trade_no",
			identifier:    orderID,
			refundID:      orderID + "-refund",
		})
	}
	if tradeNo := strings.TrimSpace(req.TradeNo); tradeNo != "" {
		attempts = append(attempts, ezfpRefundAttempt{
			identifierKey: "trade_no",
			identifier:    tradeNo,
			refundID:      tradeNo + "-refund",
		})
	}
	if len(attempts) == 0 {
		return nil, fmt.Errorf("easypay refund missing order identifier")
	}
	for index, attempt := range attempts {
		params := map[string]string{
			"pid":            e.config["pid"],
			"money":          req.Amount,
			"out_refund_no":  attempt.refundID,
			"timestamp":      strconv.FormatInt(time.Now().Unix(), 10),
			"sign_type":      ezfpSignTypeRSA,
			attempt.identifierKey: attempt.identifier,
		}
		sign, err := e.signEzfp(params)
		if err != nil {
			return nil, fmt.Errorf("easypay refund sign: %w", err)
		}
		params["sign"] = sign
		body, err := e.post(ctx, e.ezfpEndpoint(ezfpRefundPath), params)
		if err != nil {
			return nil, fmt.Errorf("easypay refund request: %w", err)
		}
		response, verifyErr := e.verifyEzfpResponse(body, "refund")
		if verifyErr != nil {
			return nil, verifyErr
		}
		if ezfpResponseCode(response) != ezfpCodeSuccess {
			err = fmt.Errorf("easypay refund failed: %s", ezfpResponseMessage(response))
			if index+1 < len(attempts) && isEasyPayRefundOrderNotFound(err) {
				continue
			}
			return nil, err
		}
		refundID := strings.TrimSpace(response["refund_no"])
		if refundID == "" {
			refundID = strings.TrimSpace(response["out_refund_no"])
		}
		if refundID == "" {
			refundID = attempt.refundID
		}
		return &payment.RefundResponse{RefundID: refundID, Status: payment.ProviderStatusSuccess}, nil
	}
	return nil, fmt.Errorf("easypay refund failed")
}

// QueryRefund implements ezfp's documented refundquery endpoint so pending
// refunds can be finalized from the admin console.
func (e *EasyPay) QueryRefund(ctx context.Context, req payment.RefundQueryRequest) (*payment.RefundResponse, error) {
	if e.legacy {
		return nil, fmt.Errorf("easypay legacy refund query is unsupported")
	}
	refundID := strings.TrimSpace(req.RefundID)
	if refundID == "" {
		refundID = strings.TrimSpace(req.OrderID) + "-refund"
	}
	if refundID == "-refund" {
		return nil, fmt.Errorf("easypay refund query missing refund identifier")
	}
	params := map[string]string{
		"pid":       e.config["pid"],
		"refund_no": refundID,
		"timestamp": strconv.FormatInt(time.Now().Unix(), 10),
		"sign_type": ezfpSignTypeRSA,
	}
	sign, err := e.signEzfp(params)
	if err != nil {
		return nil, fmt.Errorf("easypay refund query sign: %w", err)
	}
	params["sign"] = sign
	body, err := e.post(ctx, e.ezfpEndpoint(ezfpRefundQueryPath), params)
	if err != nil {
		return nil, fmt.Errorf("easypay refund query: %w", err)
	}
	response, err := e.verifyEzfpResponse(body, "refund query")
	if err != nil {
		return nil, err
	}
	if ezfpResponseCode(response) != ezfpCodeSuccess {
		return nil, fmt.Errorf("easypay refund query failed: %s", ezfpResponseMessage(response))
	}
	status := payment.ProviderStatusPending
	switch response["status"] {
	case "1":
		status = payment.ProviderStatusSuccess
	case "0":
		status = payment.ProviderStatusFailed
	}
	if responseID := strings.TrimSpace(response["refund_no"]); responseID != "" {
		refundID = responseID
	}
	return &payment.RefundResponse{RefundID: refundID, Status: status}, nil
}

func (e *EasyPay) post(ctx context.Context, endpoint string, params map[string]string) ([]byte, error) {
	body, _, err := e.postRaw(ctx, endpoint, params)
	return body, err
}

func (e *EasyPay) postRaw(ctx context.Context, endpoint string, params map[string]string) ([]byte, int, error) {
	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := e.httpClient
	if client == nil {
		client = &http.Client{Timeout: easypayHTTPTimeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxEasypayResponseSize))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func parseEasyPayPrivateKey(raw string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, fmt.Errorf("PEM block not found")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse PKCS1/PKCS8 private key: %w", err)
	}
	privateKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not RSA")
	}
	return privateKey, nil
}

func parseEasyPayPublicKey(raw string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, fmt.Errorf("PEM block not found")
	}
	if certificate, err := x509.ParseCertificate(block.Bytes); err == nil {
		if publicKey, ok := certificate.PublicKey.(*rsa.PublicKey); ok {
			return publicKey, nil
		}
		return nil, fmt.Errorf("certificate public key is not RSA")
	}
	if publicKey, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		if rsaKey, ok := publicKey.(*rsa.PublicKey); ok {
			return rsaKey, nil
		}
		return nil, fmt.Errorf("public key is not RSA")
	}
	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse PKIX/PKCS1 public key: %w", err)
	}
	return publicKey, nil
}

func (e *EasyPay) signEzfp(params map[string]string) (string, error) {
	if e == nil || e.privateKey == nil {
		return "", fmt.Errorf("RSA private key is not configured")
	}
	digest := sha256.Sum256([]byte(ezfpSignaturePayload(params)))
	signature, err := rsa.SignPKCS1v15(rand.Reader, e.privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func (e *EasyPay) verifyEzfpSignature(params map[string]string, sign string) error {
	if e == nil || e.publicKey == nil {
		return fmt.Errorf("easypay public key is not configured")
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(sign))
	if err != nil {
		return fmt.Errorf("easypay invalid signature encoding: %w", err)
	}
	digest := sha256.Sum256([]byte(ezfpSignaturePayload(params)))
	if err := rsa.VerifyPKCS1v15(e.publicKey, crypto.SHA256, digest[:], decoded); err != nil {
		return fmt.Errorf("easypay invalid signature")
	}
	return nil
}

func ezfpSignaturePayload(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for key, value := range params {
		if key == "sign" || key == "sign_type" || value == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, key+"="+params[key])
	}
	return strings.Join(values, "&")
}

func (e *EasyPay) verifyEzfpResponse(body []byte, operation string) (map[string]string, error) {
	params, err := ezfpJSONParams(body)
	if err != nil {
		return nil, fmt.Errorf("easypay parse %s: %w", operation, err)
	}
	sign := strings.TrimSpace(params["sign"])
	if sign == "" {
		return nil, fmt.Errorf("easypay %s response missing sign", operation)
	}
	if err := e.verifyEzfpSignature(params, sign); err != nil {
		return nil, fmt.Errorf("easypay %s response: %w", operation, err)
	}
	return params, nil
}

func ezfpJSONParams(body []byte) (map[string]string, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	params := make(map[string]string, len(raw))
	for key, value := range raw {
		if string(value) == "null" {
			continue
		}
		var stringValue string
		if err := json.Unmarshal(value, &stringValue); err == nil {
			params[key] = stringValue
			continue
		}
		params[key] = strings.TrimSpace(string(value))
	}
	return params, nil
}

func ezfpResponseCode(params map[string]string) int {
	code, _ := strconv.Atoi(strings.TrimSpace(params["code"]))
	return code
}

func ezfpResponseMessage(params map[string]string) string {
	if message := strings.TrimSpace(params["msg"]); message != "" {
		return message
	}
	return "unknown upstream error"
}

func easyPaySign(params map[string]string, pkey string) string {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if k == "sign" || k == "sign_type" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf strings.Builder
	for i, k := range keys {
		if i > 0 {
			_ = buf.WriteByte('&')
		}
		_, _ = buf.WriteString(k + "=" + params[k])
	}
	_, _ = buf.WriteString(pkey)
	hash := md5.Sum([]byte(buf.String()))
	return hex.EncodeToString(hash[:])
}

func easyPayVerifySign(params map[string]string, pkey string, sign string) bool {
	return hmac.Equal([]byte(easyPaySign(params, pkey)), []byte(sign))
}

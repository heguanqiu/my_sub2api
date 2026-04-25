package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

func TestEasyPayCreateAPIPayment_StripsNonBMPFromSubject(t *testing.T) {
	var gotName string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		gotName = r.Form.Get("name")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":     1,
			"msg":      "success",
			"trade_no": "trade-test",
			"payurl":   "https://example.com/pay",
			"qrcode":   "https://example.com/qr",
		})
	}))
	defer server.Close()

	provider, err := NewEasyPay("1", map[string]string{
		"pid":       "pid-test",
		"pkey":      "pkey-test",
		"apiBase":   server.URL,
		"notifyUrl": "https://merchant.example.com/notify",
		"returnUrl": "https://merchant.example.com/return",
		"cidAlipay": "cid-test",
	})
	if err != nil {
		t.Fatalf("NewEasyPay: %v", err)
	}

	_, err = provider.CreatePayment(context.Background(), payment.CreatePaymentRequest{
		OrderID:     "order-test",
		Amount:      "1.00",
		PaymentType: "alipay",
		Subject:     "龙币 1.00 🐲",
		ClientIP:    "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}

	if gotName != "龙币 1.00" {
		t.Fatalf("name = %q, want %q", gotName, "龙币 1.00")
	}
}

func TestEasyPayCreateRedirectPayment_StripsNonBMPFromSubject(t *testing.T) {
	provider, err := NewEasyPay("1", map[string]string{
		"pid":         "pid-test",
		"pkey":        "pkey-test",
		"apiBase":     "https://gateway.example.com",
		"notifyUrl":   "https://merchant.example.com/notify",
		"returnUrl":   "https://merchant.example.com/return",
		"paymentMode": paymentModePopup,
	})
	if err != nil {
		t.Fatalf("NewEasyPay: %v", err)
	}

	resp, err := provider.CreatePayment(context.Background(), payment.CreatePaymentRequest{
		OrderID:     "order-test",
		Amount:      "1.00",
		PaymentType: "alipay",
		Subject:     "龙币 1.00 🐲",
	})
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}

	parsed, err := url.Parse(resp.PayURL)
	if err != nil {
		t.Fatalf("parse pay url: %v", err)
	}
	if got := parsed.Query().Get("name"); got != "龙币 1.00" {
		t.Fatalf("name = %q, want %q", got, "龙币 1.00")
	}
}

func TestSanitizeEasyPaySubject_FallsBackWhenOnlyEmojiRemain(t *testing.T) {
	if got := sanitizeEasyPaySubject("🐲"); got != "Sub2API" {
		t.Fatalf("sanitizeEasyPaySubject = %q, want %q", got, "Sub2API")
	}
}

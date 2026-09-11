package upstream

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
)

func TestCardNetV2AdapterSignsRequestAndOmitsUnsupportedFields(t *testing.T) {
	const secret = "cardnet-secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/upstream/orders" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		timestamp, nonce, signature := r.Header.Get(HeaderTimestamp), r.Header.Get(HeaderNonce), r.Header.Get(HeaderSignature)
		if r.Header.Get(HeaderApiKey) != "cardnet-key" || nonce == "" {
			t.Fatalf("missing CardNet v2 headers: key=%q nonce=%q", r.Header.Get(HeaderApiKey), nonce)
		}
		if !VerifyV2(secret, r.Method, r.URL.Path, timestamp, nonce, signature, body) {
			t.Fatal("request does not have a valid CardNet v2 signature")
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		if payload["callback_url"] != "" {
			t.Fatalf("callback_url = %#v, want empty", payload["callback_url"])
		}
		if _, exists := payload["manual_form_data"]; exists {
			t.Fatal("manual_form_data must be omitted for CardNet")
		}
		_, _ = w.Write([]byte(`{"ok":true,"order_id":456,"order_no":"CARDNET-1","status":"completed","amount":"20.00","currency":"CNY"}`))
	}))
	defer server.Close()

	adapter := NewCardNetV2Adapter(&siteconnectiondomain.Connection{BaseURL: server.URL, ApiKey: "cardnet-key", ApiSecret: secret}, t.TempDir())
	result, err := adapter.CreateOrder(context.Background(), CreateUpstreamOrderReq{SKUID: 123, Quantity: 1, DownstreamOrderNo: "merchant-1", CallbackURL: "https://merchant.example/callback", ManualFormData: map[string]any{"email": "a@example.com"}})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if !result.OK || result.OrderID != 456 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

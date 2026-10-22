package upstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

func TestCardNetRequestPolicies(t *testing.T) {
	for _, tc := range []struct {
		status int
		code   string
		retry  bool
	}{
		{401, "authentication_failed", false}, {403, "credential_paused", false},
		{409, "idempotency_conflict", false}, {409, "insufficient_stock", false},
		{400, "product_not_supported", false}, {400, "", false},
		{429, "rate_limited", true}, {503, "service_unavailable", true},
		{409, "replay_detected", true},
	} {
		t.Run(fmt.Sprintf("%d/%s", tc.status, tc.code), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Retry-After", "90")
				w.WriteHeader(tc.status)
				_, _ = fmt.Fprintf(w, `{"ok":false,"error_code":%q,"error_message":"do not log card payload"}`, tc.code)
			}))
			defer server.Close()
			adapter := NewCardNetV2Adapter(&siteconnectiondomain.Connection{BaseURL: server.URL, ApiKey: t.Name()}, t.TempDir())
			result, err := adapter.CreateOrder(context.Background(), CreateUpstreamOrderReq{SKUID: 1, Quantity: 1, DownstreamOrderNo: "same-order"})
			if err != nil || result.Retryable == nil || *result.Retryable != tc.retry {
				t.Fatalf("result=%+v, err=%v", result, err)
			}
			if result.RetryAfter != 90*time.Second {
				t.Fatalf("retry delay=%v", result.RetryAfter)
			}
			if result.ErrorMessage == "do not log card payload" {
				t.Fatal("upstream error text leaked")
			}
		})
	}
}

func TestCardNetEncodesIncrementalTimeAndCapsPageSize(t *testing.T) {
	updated := time.Date(2026, 10, 22, 8, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("updated_after"); got != updated.Format(time.RFC3339) {
			t.Errorf("time=%q", got)
		}
		if r.URL.Query().Get("page_size") != "50" {
			t.Error("CardNet page size must be capped at 50")
		}
		if !VerifyV2("secret", r.Method, r.URL.Path, r.Header.Get(HeaderTimestamp), r.Header.Get(HeaderNonce), r.Header.Get(HeaderSignature), nil) {
			t.Error("invalid query request signature")
		}
		_, _ = w.Write([]byte(`{"ok":true,"items":[],"total":0}`))
	}))
	defer server.Close()
	adapter := NewCardNetV2Adapter(&siteconnectiondomain.Connection{BaseURL: server.URL, ApiKey: t.Name(), ApiSecret: "secret"}, t.TempDir())
	if _, err := adapter.ListProducts(context.Background(), ListProductsOpts{Page: 2, PageSize: 200, UpdatedAfter: &updated}); err != nil {
		t.Fatal(err)
	}
}

func TestCardNetQuantityLimitDoesNotSendRequest(t *testing.T) {
	adapter := NewCardNetV2Adapter(&siteconnectiondomain.Connection{BaseURL: "http://must-not-be-called.invalid", ApiKey: t.Name()}, t.TempDir())
	for _, quantity := range []int{0, 101} {
		result, err := adapter.CreateOrder(context.Background(), CreateUpstreamOrderReq{Quantity: quantity})
		if err != nil || result.OK || result.ErrorCode != "invalid_request" || result.Retryable == nil || *result.Retryable {
			t.Fatalf("result=%+v, err=%v", result, err)
		}
	}
}

func TestCardNetPausesAcrossAdaptersUntilManualPing(t *testing.T) {
	calls := 0
	authorized := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path == "/api/v1/upstream/ping" {
			authorized = true
			_, _ = w.Write([]byte(`{"ok":true,"balance":"10.00"}`))
			return
		}
		if authorized {
			_, _ = w.Write([]byte(`{"ok":true,"order_id":1}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"ok":false,"error_code":"authentication_failed"}`))
	}))
	defer server.Close()
	conn := &siteconnectiondomain.Connection{BaseURL: server.URL, ApiKey: t.Name()}
	adapter := NewCardNetV2Adapter(conn, t.TempDir())
	if _, err := adapter.GetOrder(context.Background(), 1); err == nil {
		t.Fatal("expected unauthorized")
	}
	other := NewCardNetV2Adapter(conn, t.TempDir())
	if _, err := other.GetOrder(context.Background(), 1); !errors.Is(err, ErrAuthenticationPaused) {
		t.Fatalf("error=%v", err)
	}
	if calls != 1 {
		t.Fatalf("paused adapter sent %d requests", calls)
	}
	if _, err := other.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
	other.ResumeRequests()
	if _, err := adapter.GetOrder(context.Background(), 1); err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	if calls != 3 {
		t.Fatalf("calls=%d", calls)
	}
}

func TestCardNetRetryPreservesBusinessParametersAndRefreshesNonce(t *testing.T) {
	var bodies [][]byte
	var nonces []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		bodies = append(bodies, body)
		nonces = append(nonces, r.Header.Get(HeaderNonce))
		if !VerifyV2("secret", r.Method, r.URL.Path, r.Header.Get(HeaderTimestamp), r.Header.Get(HeaderNonce), r.Header.Get(HeaderSignature), body) {
			t.Error("invalid signature")
		}
		if len(bodies) == 1 {
			w.WriteHeader(503)
			_, _ = w.Write([]byte(`{"ok":false,"error_code":"service_unavailable"}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"order_id":456,"status":"completed"}`))
	}))
	defer server.Close()
	adapter := NewCardNetV2Adapter(&siteconnectiondomain.Connection{BaseURL: server.URL, ApiKey: t.Name(), ApiSecret: "secret"}, t.TempDir())
	req := CreateUpstreamOrderReq{SKUID: 123, Quantity: 2, DownstreamOrderNo: "same-order", TraceID: "same-trace"}
	first, err := adapter.CreateOrder(context.Background(), req)
	if err != nil || first.OK || first.Retryable == nil || !*first.Retryable {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	second, err := adapter.CreateOrder(context.Background(), req)
	if err != nil || !second.OK {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	if string(bodies[0]) != string(bodies[1]) || nonces[0] == nonces[1] {
		t.Fatal("retry changed business parameters or reused a nonce")
	}
}

func TestCardNetLimitReservationsAndCancellation(t *testing.T) {
	now := time.Now()
	state, host := &cardNetLimitState{}, &cardNetLimitState{}
	if wait := state.reserve(now, host, true); wait != 0 {
		t.Fatal(wait)
	}
	if wait := state.reserve(now.Add(time.Second), host, false); wait != 100*time.Millisecond {
		t.Fatal(wait)
	}
	if wait := state.reserve(now.Add(1100*time.Millisecond), host, true); wait != time.Second {
		t.Fatal(wait)
	}
	other := &cardNetLimitState{}
	if wait := other.reserve(now, host, false); wait != cardNetHostSpacing {
		t.Fatal(wait)
	}
	state.reads = 8
	if wait := state.reserve(now.Add(time.Minute), host, false); wait == 0 {
		t.Fatal("read concurrency was not limited")
	}
	state.reads = 0
	state.writes = 2
	if wait := state.reserve(now.Add(time.Minute), host, true); wait == 0 {
		t.Fatal("write concurrency was not limited")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := acquireCardNetRequest(ctx, "https://example.com", t.Name(), http.MethodGet); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}

func TestCardNetRetryAfterSupportsDates(t *testing.T) {
	now := time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		value string
		want  time.Duration
	}{
		{"5", 5 * time.Second}, {now.Add(time.Minute).Format(http.TimeFormat), time.Minute}, {"invalid", 0}, {"-1", 0},
	} {
		if got := cardNetRetryAfter(tc.value, now); got != tc.want {
			t.Fatalf("Retry-After %q=%v", tc.value, got)
		}
	}
}

func TestCardNetRetryAfterBlocksOtherAdaptersWithoutReusingSignature(t *testing.T) {
	var calls int
	var nonces []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		nonces = append(nonces, r.Header.Get(HeaderNonce))
		w.Header().Set("Retry-After", "90")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"ok":false,"error_code":"service_unavailable"}`))
	}))
	defer server.Close()
	conn := &siteconnectiondomain.Connection{BaseURL: server.URL, ApiKey: t.Name()}
	adapter := NewCardNetV2Adapter(conn, t.TempDir())
	if _, err := adapter.GetOrder(context.Background(), 1); err == nil {
		t.Fatal("expected service unavailable")
	}
	other := NewCardNetV2Adapter(conn, t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := other.GetOrder(ctx, 1); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error=%v", err)
	}
	if calls != 1 || len(nonces) != 1 {
		t.Fatalf("request was sent during Retry-After: %d", calls)
	}
}

package upstream

import (
	"testing"
	"time"
)

func TestSignAndVerify(t *testing.T) {
	secret := "test-secret-key-12345"
	method := "POST"
	path := "/api/v1/upstream/orders"
	timestamp := time.Now().Unix()
	body := []byte(`{"sku_id":1,"quantity":1}`)

	sig := Sign(secret, method, path, timestamp, body)
	if sig == "" {
		t.Fatal("signature should not be empty")
	}

	// 正确的签名应该验证通过
	if !Verify(secret, method, path, sig, timestamp, body) {
		t.Fatal("signature verification should pass")
	}

	// 错误的 secret 应该验证失败
	if Verify("wrong-secret", method, path, sig, timestamp, body) {
		t.Fatal("signature verification should fail with wrong secret")
	}

	// 错误的 body 应该验证失败
	if Verify(secret, method, path, sig, timestamp, []byte(`{"sku_id":2}`)) {
		t.Fatal("signature verification should fail with different body")
	}

	// 错误的 path 应该验证失败
	if Verify(secret, method, "/api/v1/upstream/ping", sig, timestamp, body) {
		t.Fatal("signature verification should fail with different path")
	}
}

func TestSignEmptyBody(t *testing.T) {
	secret := "test-secret"
	sig1 := Sign(secret, "GET", "/test", 1000, nil)
	sig2 := Sign(secret, "GET", "/test", 1000, []byte{})

	// nil 和空 []byte 的 MD5 相同
	if sig1 != sig2 {
		t.Fatal("nil body and empty body should produce same signature")
	}
}

func TestSignV2CardNetDocumentVector(t *testing.T) {
	const secret = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	const want = "09cf5cedb04ba4b32552c7e9d8c533368eeb48dd9ba1b72fd1e2c151e9487819"

	got := SignV2(secret, "POST", "/api/v1/upstream/ping", "1709625600", "550e8400-e29b-41d4-a716-446655440000", nil)
	if got != want {
		t.Fatalf("CardNet v2 signature = %s, want %s", got, want)
	}
	if !VerifyV2(secret, "POST", "/api/v1/upstream/ping", "1709625600", "550e8400-e29b-41d4-a716-446655440000", got, nil) {
		t.Fatal("CardNet v2 signature should verify")
	}
}

func TestIsTimestampValid(t *testing.T) {
	now := time.Now().Unix()

	if !IsTimestampValid(now) {
		t.Fatal("current timestamp should be valid")
	}

	if !IsTimestampValid(now - (MaxTimestampSkew / 2)) {
		t.Fatalf("timestamp within skew window should be valid: skew=%d", MaxTimestampSkew)
	}

	if IsTimestampValid(now - (MaxTimestampSkew + 1)) {
		t.Fatalf("timestamp older than skew window should be invalid: skew=%d", MaxTimestampSkew)
	}

	if IsTimestampValid(now + (MaxTimestampSkew + 1)) {
		t.Fatalf("future timestamp beyond skew window should be invalid: skew=%d", MaxTimestampSkew)
	}
}

func TestParseTimestamp(t *testing.T) {
	ts, err := ParseTimestamp("1709625600")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts != 1709625600 {
		t.Fatalf("expected 1709625600, got %d", ts)
	}

	_, err = ParseTimestamp("not-a-number")
	if err == nil {
		t.Fatal("expected error for non-numeric string")
	}
}

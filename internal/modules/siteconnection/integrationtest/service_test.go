package siteconnection_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	siteconnectionapp "github.com/dujiao-next/internal/modules/siteconnection/application"
	siteconnectioncontract "github.com/dujiao-next/internal/modules/siteconnection/contract"
	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/crypto"
	"github.com/dujiao-next/internal/upstream"

	"github.com/shopspring/decimal"
)

func TestSiteConnectionServicePingReturnsAdapterCreationError(t *testing.T) {
	appSecretKey := "test-secret-key"
	encrypted, err := crypto.Encrypt(crypto.DeriveKey(appSecretKey), "upstream-secret")
	if err != nil {
		t.Fatalf("encrypt secret failed: %v", err)
	}
	repo := &siteConnectionRepoStub{
		conn: &siteconnectiondomain.Connection{
			ID:        1,
			Name:      "unsupported upstream",
			BaseURL:   "https://upstream.example.com",
			ApiKey:    "upstream-key",
			ApiSecret: encrypted,
			Protocol:  "unsupported-protocol",
			Status:    constants.ConnectionStatusPending,
		},
	}
	svc := siteconnectionapp.NewService(repo, appSecretKey, t.TempDir())

	result, err := svc.Ping(1)

	if err == nil {
		t.Fatalf("expected adapter creation error")
	}
	if result != nil {
		t.Fatalf("expected nil ping result, got %#v", result)
	}
	if !strings.Contains(err.Error(), "unsupported protocol") {
		t.Fatalf("expected unsupported protocol error, got %v", err)
	}
	if repo.updated {
		t.Fatalf("connection should not be updated when adapter creation fails")
	}
}

func TestCardNetAuthenticationPauseSurvivesServiceRestartAndRequiresPing(t *testing.T) {
	var unauthorized atomic.Bool
	unauthorized.Store(true)
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if unauthorized.Load() {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"ok":false,"error_code":"authentication_failed"}`))
			return
		}
		if r.URL.Path == "/api/v1/upstream/ping" {
			_, _ = w.Write([]byte(`{"ok":true,"balance":"10.00"}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"order_id":1,"status":"completed"}`))
	}))
	defer server.Close()
	repo := &siteConnectionRepoStub{}
	service := siteconnectionapp.NewService(repo, "test-key", t.TempDir())
	conn, err := service.Create(siteconnectionapp.CreateInput{Name: "CardNet", BaseURL: server.URL, ApiKey: t.Name(), ApiSecret: "secret", Protocol: constants.ConnectionProtocolCardNetV2})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := service.GetAdapter(conn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.GetOrder(context.Background(), 1); err == nil {
		t.Fatal("expected forbidden")
	}
	if !repo.conn.AuthenticationPaused || repo.conn.Status != constants.ConnectionStatusDisabled {
		t.Fatalf("pause not persisted: %+v", repo.conn)
	}
	service = siteconnectionapp.NewService(repo, "test-key", t.TempDir())
	other, err := service.GetAdapter(repo.conn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.GetOrder(context.Background(), 1); !errors.Is(err, upstream.ErrAuthenticationPaused) {
		t.Fatalf("pause lost: %v", err)
	}
	if err := service.SetStatus(conn.ID, constants.ConnectionStatusActive); !errors.Is(err, siteconnectioncontract.ErrAuthenticationPaused) {
		t.Fatalf("status change bypassed ping: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatal("paused connection contacted upstream")
	}
	unauthorized.Store(false)
	if _, err := service.Ping(conn.ID); err != nil {
		t.Fatal(err)
	}
	if repo.conn.AuthenticationPaused || repo.conn.Status != constants.ConnectionStatusActive {
		t.Fatal("ping did not recover connection")
	}
	if _, err := adapter.GetOrder(context.Background(), 1); err != nil {
		t.Fatalf("existing adapter did not recover: %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

type fakeMarkupReapplier struct {
	calls []uint
}

func (f *fakeMarkupReapplier) ReapplyMarkup(connectionID uint) (int, error) {
	f.calls = append(f.calls, connectionID)
	return 0, nil
}

func newReapplyTestConn() *siteconnectiondomain.Connection {
	return &siteconnectiondomain.Connection{
		ID:                 7,
		Name:               "conn",
		BaseURL:            "https://up.example.com",
		ApiKey:             "key",
		Protocol:           "dujiao-next",
		Status:             constants.ConnectionStatusActive,
		ExchangeRate:       decimal.NewFromInt(1),
		PriceMarkupPercent: decimal.Zero,
		PriceRoundingMode:  "none",
	}
}

func TestSiteConnectionServiceUpdateTriggersReapplyWhenExchangeRateChanges(t *testing.T) {
	repo := &siteConnectionRepoStub{conn: newReapplyTestConn()}
	svc := siteconnectionapp.NewService(repo, "test-secret-key", t.TempDir())
	reapplier := &fakeMarkupReapplier{}
	svc.SetMarkupReapplier(reapplier)

	rate := 6.9
	if _, err := svc.Update(7, siteconnectionapp.UpdateInput{ExchangeRate: &rate}); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if len(reapplier.calls) != 1 || reapplier.calls[0] != 7 {
		t.Fatalf("expected reapply called once for conn 7, got %#v", reapplier.calls)
	}
}

func TestSiteConnectionServiceUpdateSkipsReapplyWhenPriceConfigUnchanged(t *testing.T) {
	repo := &siteConnectionRepoStub{conn: newReapplyTestConn()}
	svc := siteconnectionapp.NewService(repo, "test-secret-key", t.TempDir())
	reapplier := &fakeMarkupReapplier{}
	svc.SetMarkupReapplier(reapplier)

	// 只改名字，汇率传入与现值相同的 1 → 定价配置未变，不应触发重算。
	name := "renamed"
	rate := 1.0
	if _, err := svc.Update(7, siteconnectionapp.UpdateInput{Name: name, ExchangeRate: &rate}); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if len(reapplier.calls) != 0 {
		t.Fatalf("expected no reapply when price config unchanged, got %#v", reapplier.calls)
	}
}

type siteConnectionRepoStub struct {
	conn    *siteconnectiondomain.Connection
	updated bool
}

func (r *siteConnectionRepoStub) GetByID(id uint) (*siteconnectiondomain.Connection, error) {
	if r.conn != nil && r.conn.ID == id {
		copy := *r.conn
		return &copy, nil
	}
	return nil, nil
}

func (r *siteConnectionRepoStub) GetByApiKey(apiKey string) (*siteconnectiondomain.Connection, error) {
	if r.conn != nil && r.conn.ApiKey == apiKey {
		copy := *r.conn
		return &copy, nil
	}
	return nil, nil
}

func (r *siteConnectionRepoStub) Create(conn *siteconnectiondomain.Connection) error {
	copy := *conn
	r.conn = &copy
	return nil
}

func (r *siteConnectionRepoStub) Update(conn *siteconnectiondomain.Connection) error {
	r.updated = true
	copy := *conn
	r.conn = &copy
	return nil
}

func (r *siteConnectionRepoStub) UpdateAuthenticationState(conn *siteconnectiondomain.Connection, paused bool) (bool, error) {
	if r.conn == nil || r.conn.ApiKey != conn.ApiKey || r.conn.ApiSecret != conn.ApiSecret || r.conn.BaseURL != conn.BaseURL {
		return false, nil
	}
	r.conn.AuthenticationPaused = paused
	r.conn.LastPingOK = !paused
	if paused {
		r.conn.Status = constants.ConnectionStatusDisabled
	} else if conn.AuthenticationPaused || conn.Status == constants.ConnectionStatusPending {
		r.conn.Status = constants.ConnectionStatusActive
	}
	return true, nil
}

func (r *siteConnectionRepoStub) RecordPingFailure(conn *siteconnectiondomain.Connection) error {
	if r.conn != nil && r.conn.ApiSecret == conn.ApiSecret && r.conn.ApiKey == conn.ApiKey {
		r.conn.LastPingOK = false
	}
	return nil
}

func (r *siteConnectionRepoStub) Delete(id uint) error {
	if r.conn != nil && r.conn.ID == id {
		r.conn = nil
	}
	return nil
}

func (r *siteConnectionRepoStub) List(siteconnectioncontract.ListFilter) ([]siteconnectiondomain.Connection, int64, error) {
	if r.conn == nil {
		return nil, 0, nil
	}
	return []siteconnectiondomain.Connection{*r.conn}, 1, nil
}

func (r *siteConnectionRepoStub) ListActive() ([]siteconnectiondomain.Connection, error) {
	if r.conn == nil || r.conn.Status != constants.ConnectionStatusActive {
		return nil, nil
	}
	return []siteconnectiondomain.Connection{*r.conn}, nil
}

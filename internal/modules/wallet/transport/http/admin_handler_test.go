package wallethttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	walletdomain "github.com/dujiao-next/internal/modules/wallet/domain"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

func TestAdjustUserWalletRequiresRemarkWithDedicatedMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("admin_id", uint(1))
	ctx.Params = gin.Params{{Key: "id", Value: "10"}}
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/users/10/wallet/adjust",
		strings.NewReader(`{"operation":"add","amount":"10.00","remark":"   "}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.Header.Set("Accept-Language", "en-US")

	(&AdminHandler{}).AdjustUserWallet(ctx)

	if !strings.Contains(recorder.Body.String(), "A remark is required for wallet balance adjustments") {
		t.Fatalf("expected dedicated remark validation message, got %s", recorder.Body.String())
	}
}

func TestBindManualRechargeRequestNormalizesEscapedUnderscores(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/wallet/manual-recharges", strings.NewReader(`{"channel\_id":1,"amount":"1","transaction\_no":"TXN-1","contact\_type":"tg","contact\_value":"@example","proof\_url":"/uploads/proof.jpg"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	var req manualRechargeRequest
	if err := bindManualRechargeRequest(ctx, &req); err != nil {
		t.Fatalf("bindManualRechargeRequest returned error: %v", err)
	}
	if req.ChannelID != 1 || req.TransactionNo != "TXN-1" || req.ProofURL != "/uploads/proof.jpg" {
		t.Fatalf("unexpected normalized request: %#v", req)
	}
}

func TestGetWalletReturnsManualRechargeSettingWithBalance(t *testing.T) {
	for _, tc := range []struct {
		name    string
		setting SiteCurrencyReader
		want    bool
	}{
		{name: "enabled", setting: manualRechargeSettingStub{enabled: true}, want: true},
		{name: "disabled", setting: manualRechargeSettingStub{}},
		{name: "missing settings"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Set("user_id", uint(1))
			handler := &UserHandler{wallets: walletAccountStub{}, settings: tc.setting}
			handler.GetWallet(ctx)

			var result struct {
				StatusCode int `json:"status_code"`
				Data       struct {
					Balance string `json:"balance"`
					Enabled *bool  `json:"manual_recharge_enabled"`
				} `json:"data"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != http.StatusOK || result.StatusCode != 0 || result.Data.Balance != "12.00" || result.Data.Enabled == nil || *result.Data.Enabled != tc.want {
				t.Fatalf("unexpected wallet response: %s", recorder.Body.String())
			}
		})
	}
}

type walletAccountStub struct{ WalletService }

func (walletAccountStub) GetAccount(uint) (*walletdomain.Account, error) {
	return &walletdomain.Account{Balance: money.FromDecimal(decimal.NewFromInt(12))}, nil
}

type manualRechargeSettingStub struct {
	SiteCurrencyReader
	enabled bool
}

func (s manualRechargeSettingStub) GetManualRechargeEnabled() bool { return s.enabled }

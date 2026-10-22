package wallethttp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
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

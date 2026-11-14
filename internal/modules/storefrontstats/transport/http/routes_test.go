package storefrontstatshttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	reseller "github.com/dujiao-next/internal/modules/reseller/contract"
	"github.com/dujiao-next/internal/modules/storefrontstats/domain"
	"github.com/gin-gonic/gin"
)

type visitorStoreStub struct {
	siteID uint
	key    string
}

func (s *visitorStoreStub) RecordHomeVisit(_ context.Context, siteID uint, key, _ string) error {
	s.siteID, s.key = siteID, key
	return nil
}

func (s *visitorStoreStub) HomeVisitorStats(context.Context, string) (domain.HomeVisitorStats, error) {
	return domain.HomeVisitorStats{Today: 3, Total: 25}, nil
}

func TestPublicVisitDoesNotExposeStatistics(t *testing.T) {
	router := gin.New()
	store := &visitorStoreStub{}
	RegisterPublicRoutes(router.Group("/public"), store, func(c *gin.Context) { c.Next() })
	const visitor = "4a53cab4e98f4ae0a22591cc923af21a"
	tenant := reseller.ResellerTenantContext("reseller.example", 17, 1, "reseller.example")
	req := httptest.NewRequest(http.MethodPost, "/public/home-visitors", strings.NewReader(`{"visitor_id":"`+visitor+`"}`))
	req = req.WithContext(reseller.WithTenantContext(req.Context(), tenant))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	var result struct {
		StatusCode int              `json:"status_code"`
		Data       *json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusOK || result.StatusCode != 0 || result.Data != nil {
		t.Fatalf("public response = %d, %s", recorder.Code, recorder.Body.String())
	}
	if store.siteID != 17 || len(store.key) != 64 {
		t.Fatalf("stored visit = site %d, key length %d", store.siteID, len(store.key))
	}
}

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeadersProtectSensitiveResponses(t *testing.T) {
	for _, tc := range []struct {
		path   string
		auth   string
		status int
	}{
		{"/api/v1/admin/card-secrets", "Bearer token", 200},
		{"/api/v1/guest/orders/ORDER/fulfillment/download", "Guest credentials", 200},
		{"/api/v1/admin/login", "", 401},
		{"/api/missing", "", 404},
		{"/custom-callback", "Bearer token", 200},
	} {
		t.Run(tc.path, func(t *testing.T) {
			r := gin.New()
			r.Use(SecurityHeadersMiddleware())
			r.GET(tc.path, func(c *gin.Context) { c.String(tc.status, "sensitive") })
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("Authorization", tc.auth)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			for header, want := range map[string]string{
				"Cache-Control": "no-store", "Pragma": "no-cache", "Expires": "0",
				"X-Content-Type-Options": "nosniff", "Referrer-Policy": "no-referrer", "X-Frame-Options": "DENY",
			} {
				if got := w.Header().Get(header); got != want {
					t.Errorf("%s = %q, want %q", header, got, want)
				}
			}
		})
	}
}

func TestSecurityHeadersPreservePublicAssetCaching(t *testing.T) {
	r := gin.New()
	r.Use(SecurityHeadersMiddleware())
	r.GET("/assets/app.js", func(c *gin.Context) { c.String(200, "asset") })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))
	if got := w.Header().Get("Cache-Control"); got != "" {
		t.Fatalf("public asset Cache-Control = %q", got)
	}
}

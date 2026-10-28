package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware prevents sensitive API responses from being stored
// by browsers and shared caches, including authentication failures and downloads.
// Register globally before callback dispatch, which can handle requests early.
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "no-referrer")
		if c.Request.URL.Path == "/api" || strings.HasPrefix(c.Request.URL.Path, "/api/") || c.GetHeader("Authorization") != "" {
			c.Header("Cache-Control", "no-store")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
			c.Header("X-Frame-Options", "DENY")
		}
		c.Next()
	}
}

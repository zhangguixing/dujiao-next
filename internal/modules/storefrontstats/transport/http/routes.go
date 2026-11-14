package storefrontstatshttp

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	reseller "github.com/dujiao-next/internal/modules/reseller/contract"
	"github.com/dujiao-next/internal/modules/storefrontstats/contract"
	"github.com/dujiao-next/internal/platform/http/ginutil"
	"github.com/dujiao-next/internal/platform/http/response"
	"github.com/gin-gonic/gin"
)

func RegisterPublicRoutes(public gin.IRoutes, store contract.Store, limit gin.HandlerFunc) {
	public.POST("/home-visitors", limit, func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 256)
		var input struct {
			VisitorID string `json:"visitor_id"`
		}
		if err := c.ShouldBindJSON(&input); err != nil || len(input.VisitorID) != 32 {
			ginutil.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
			return
		}
		visitor, err := hex.DecodeString(input.VisitorID)
		if err != nil {
			ginutil.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
			return
		}
		digest := sha256.Sum256(visitor)
		ctx := c.Request.Context()
		err = store.RecordHomeVisit(ctx, reseller.RequestResellerID(ctx), hex.EncodeToString(digest[:]), time.Now().Format(time.DateOnly))
		if err != nil {
			ginutil.RespondError(c, response.CodeInternal, "error.internal_error", err)
			return
		}
		response.Success(c, nil)
	})
}

func RegisterAdminRoutes(admin gin.IRoutes, store contract.Store) {
	admin.GET("/dashboard/home-visitors", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		stats, err := store.HomeVisitorStats(c.Request.Context(), time.Now().Format(time.DateOnly))
		if err != nil {
			ginutil.RespondError(c, response.CodeInternal, "error.dashboard_fetch_failed", err)
			return
		}
		response.Success(c, stats)
	})
}

package uploadhttp

import "github.com/gin-gonic/gin"

func RegisterAdminRoutes(admin gin.IRoutes, handler *AdminHandler) {
	admin.POST("/upload", handler.UploadFile)
}

func RegisterUserRoutes(user gin.IRoutes, handler *UserHandler) {
	if user == nil || handler == nil {
		panic("upload user routes: required dependency is nil")
	}
	user.POST("/uploads/manual-recharge-proof", handler.UploadManualRechargeProof)
}

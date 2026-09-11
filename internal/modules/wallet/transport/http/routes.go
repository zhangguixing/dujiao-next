package wallethttp

import "github.com/gin-gonic/gin"

// RegisterUserRoutes 注册用户钱包端点。
func RegisterUserRoutes(user gin.IRoutes, handler *UserHandler) {
	if user == nil || handler == nil {
		panic("wallet user routes: required dependency is nil")
	}
	user.GET("/wallet", handler.GetWallet)
	user.GET("/wallet/transactions", handler.GetTransactions)
	user.POST("/wallet/payment-channels", handler.GetPaymentChannels)
	user.GET("/wallet/manual-recharge-channels", handler.GetManualRechargeChannels)
	user.POST("/wallet/manual-recharges", handler.CreateManualRecharge)
	user.GET("/wallet/manual-recharges", handler.ListManualRecharges)
	user.GET("/wallet/manual-recharges/:request_no", handler.GetManualRecharge)
	user.POST("/wallet/manual-recharges/:request_no/cancel", handler.CancelManualRecharge)
	user.POST("/wallet/recharge", handler.Recharge)
	user.GET("/wallet/recharges", handler.ListRecharges)
	user.GET("/wallet/recharges/stats", handler.RechargeStats)
	user.GET("/wallet/recharges/:recharge_no", handler.GetRecharge)
	user.POST("/wallet/recharge/payments/:id/capture", handler.CaptureRechargePayment)
}

// RegisterAdminRoutes 注册后台钱包端点（须挂在 paymentProtected 分组）。
func RegisterAdminRoutes(paymentProtected gin.IRoutes, handler *AdminHandler) {
	if paymentProtected == nil || handler == nil {
		panic("wallet admin routes: required dependency is nil")
	}
	paymentProtected.GET("/users/:id/wallet", handler.GetUserWallet)
	paymentProtected.GET("/users/:id/wallet/transactions", handler.GetUserTransactions)
	paymentProtected.POST("/users/:id/wallet/adjust", handler.AdjustUserWallet)
	paymentProtected.GET("/wallet/recharges", handler.GetRecharges)
	paymentProtected.GET("/manual-recharge-channels", handler.GetManualRechargeChannels)
	paymentProtected.POST("/manual-recharge-channels", handler.SaveManualRechargeChannel)
	paymentProtected.PUT("/manual-recharge-channels/:id", handler.SaveManualRechargeChannel)
	paymentProtected.DELETE("/manual-recharge-channels/:id", handler.DeleteManualRechargeChannel)
	paymentProtected.GET("/wallet/manual-recharges", handler.GetManualRecharges)
	paymentProtected.POST("/wallet/manual-recharges/:id/approve", handler.ApproveManualRecharge)
	paymentProtected.POST("/wallet/manual-recharges/:id/reject", handler.RejectManualRecharge)
}

// RegisterChannelRoutes 注册渠道钱包端点。
func RegisterChannelRoutes(channel gin.IRoutes, handler *ChannelHandler) {
	if channel == nil || handler == nil {
		panic("wallet channel routes: required dependency is nil")
	}
	channel.GET("/wallet", handler.GetWallet)
	channel.GET("/wallet/transactions", handler.GetWalletTransactions)
	channel.POST("/wallet/recharge", handler.CreateWalletRecharge)
}

package walletbootstrap

import (
	"errors"
	"fmt"

	notificationcontract "github.com/dujiao-next/internal/modules/notification/contract"
	paymentapp "github.com/dujiao-next/internal/modules/payment/application"

	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"

	"github.com/dujiao-next/internal/constants"
	userdomain "github.com/dujiao-next/internal/modules/identity/user/domain"
	orderapp "github.com/dujiao-next/internal/modules/order/application"
	walletapp "github.com/dujiao-next/internal/modules/wallet/application"
	walletcontract "github.com/dujiao-next/internal/modules/wallet/contract"
	walletdomain "github.com/dujiao-next/internal/modules/wallet/domain"
	wallettransport "github.com/dujiao-next/internal/modules/wallet/transport/http"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/money"
)

// walletTransportAdapter composes wallet use cases with payment capabilities
// needed by the wallet HTTP surface.
type walletTransportAdapter struct {
	wallets  *walletapp.Service
	payments *paymentapp.PaymentService
}

type manualRechargeNotifier struct {
	notifications notificationcontract.NotificationEnqueuer
}

func (n manualRechargeNotifier) NotifyManualRechargePending(request *walletdomain.ManualRechargeRequest, user *userdomain.User) error {
	if n.notifications == nil || request == nil {
		return nil
	}
	label, email := "", ""
	if user != nil {
		label = user.DisplayName
		email = user.Email
	}
	return n.notifications.Enqueue(notificationcontract.EnqueueInput{EventType: constants.NotificationEventManualRechargePending, BizType: constants.NotificationBizTypeManualRecharge, BizID: request.ID, Data: jsonmap.JSON{"request_no": request.RequestNo, "customer_label": label, "customer_email": email, "amount": request.Amount.StringFixed(2), "currency": request.Currency, "channel_id": fmt.Sprintf("%d", request.ChannelID), "transaction_no": request.TransactionNo, "contact_type": request.ContactType, "contact_value": request.ContactValue}})
}

func (a walletTransportAdapter) GetAccount(userID uint) (*walletdomain.Account, error) {
	account, err := a.wallets.GetAccount(userID)
	return account, mapWalletTransportError(err)
}

func (a walletTransportAdapter) ListTransactions(userID uint, page, pageSize int) ([]walletdomain.Transaction, int64, error) {
	transactions, total, err := a.wallets.ListTransactions(walletcontract.TransactionListFilter{
		Page: page, PageSize: pageSize, UserID: userID,
	})
	return transactions, total, mapWalletTransportError(err)
}

func (a walletTransportAdapter) ListAdminTransactions(userID uint, page, pageSize int, typ, direction string) ([]walletdomain.Transaction, int64, error) {
	transactions, total, err := a.wallets.ListTransactions(walletcontract.TransactionListFilter{
		Page: page, PageSize: pageSize, UserID: userID, Type: typ, Direction: direction,
	})
	return transactions, total, mapWalletTransportError(err)
}

func (a walletTransportAdapter) ListRechargeOrdersAdmin(filter wallettransport.AdminRechargeListFilter) ([]walletdomain.RechargeOrder, int64, error) {
	orders, total, err := a.wallets.ListRechargeOrdersAdmin(walletcontract.RechargeListFilter{
		Page:         filter.Page,
		PageSize:     filter.PageSize,
		RechargeNo:   filter.RechargeNo,
		UserID:       filter.UserID,
		UserKeyword:  filter.UserKeyword,
		PaymentID:    filter.PaymentID,
		ChannelID:    filter.ChannelID,
		ProviderType: filter.ProviderType,
		ChannelType:  filter.ChannelType,
		Status:       filter.Status,
		CreatedFrom:  filter.CreatedFrom,
		CreatedTo:    filter.CreatedTo,
		PaidFrom:     filter.PaidFrom,
		PaidTo:       filter.PaidTo,
	})
	return orders, total, mapWalletTransportError(err)
}

func (a walletTransportAdapter) AdminAdjustBalance(input wallettransport.AdjustBalanceInput) (*walletdomain.Account, *walletdomain.Transaction, error) {
	account, txn, err := a.wallets.AdminAdjustBalance(walletcontract.AdjustBalanceInput{
		UserID:          input.UserID,
		OperatorAdminID: input.OperatorAdminID,
		Delta:           input.Delta,
		Currency:        input.Currency,
		Remark:          input.Remark,
	})
	return account, txn, mapWalletTransportError(err)
}

func (a walletTransportAdapter) ListUserRechargeOrders(userID uint, page, pageSize int, status, rechargeNo string) ([]walletdomain.RechargeOrder, int64, error) {
	orders, total, err := a.wallets.ListUserRechargeOrders(userID, page, pageSize, status, rechargeNo)
	return orders, total, mapWalletTransportError(err)
}

func (a walletTransportAdapter) StatsUserRechargeOrders(userID uint, rechargeNo string) (map[string]int64, error) {
	stats, err := a.wallets.StatsUserRechargeOrders(userID, rechargeNo)
	return stats, mapWalletTransportError(err)
}

func (a walletTransportAdapter) GetRechargeOrderByRechargeNo(userID uint, rechargeNo string) (*walletdomain.RechargeOrder, error) {
	order, err := a.wallets.GetRechargeOrderByRechargeNo(userID, rechargeNo)
	return order, mapWalletTransportError(err)
}

func (a walletTransportAdapter) GetRechargeOrderByPaymentIDAndUser(paymentID uint, userID uint) (*walletdomain.RechargeOrder, error) {
	order, err := a.wallets.GetRechargeOrderByPaymentIDAndUser(paymentID, userID)
	return order, mapWalletTransportError(err)
}

func (a walletTransportAdapter) GetAvailableWalletRechargeChannels(amount money.Amount, user *userdomain.User) ([]map[string]interface{}, error) {
	channels, err := a.payments.GetAvailableChannels(paymentapp.AvailablePaymentChannelFilter{
		TargetAmount: &amount, User: user, PaymentType: constants.PaymentTypeWallet,
	})
	return channels, mapWalletTransportError(err)
}

func (a walletTransportAdapter) CreateWalletRechargePayment(input wallettransport.CreateRechargePaymentInput) (*wallettransport.CreateRechargePaymentResult, error) {
	result, err := a.payments.CreateWalletRechargePayment(paymentapp.CreateWalletRechargePaymentInput{
		UserID: input.UserID, ChannelID: input.ChannelID, Amount: input.Amount, Currency: input.Currency,
		Remark: input.Remark, ClientIP: input.ClientIP, Context: input.Context, RequestScheme: input.RequestScheme,
	})
	if err != nil {
		return nil, mapWalletTransportError(err)
	}
	return &wallettransport.CreateRechargePaymentResult{Recharge: result.Recharge, Payment: result.Payment}, nil
}

func (a walletTransportAdapter) GetPayment(id uint) (*paymentdomain.Payment, error) {
	payment, err := a.payments.GetPayment(id)
	return payment, mapWalletTransportError(err)
}

func (a walletTransportAdapter) CapturePayment(input wallettransport.CapturePaymentInput) (*paymentdomain.Payment, error) {
	payment, err := a.payments.CapturePayment(paymentapp.CapturePaymentInput{PaymentID: input.PaymentID, Context: input.Context})
	return payment, mapWalletTransportError(err)
}

func (a walletTransportAdapter) ListManualRechargeChannels(activeOnly bool) ([]walletdomain.ManualRechargeChannel, error) {
	rows, err := a.wallets.ListManualRechargeChannels(activeOnly)
	return rows, mapWalletTransportError(err)
}
func (a walletTransportAdapter) SaveManualRechargeChannel(channel *walletdomain.ManualRechargeChannel) (*walletdomain.ManualRechargeChannel, error) {
	row, err := a.wallets.SaveManualRechargeChannel(channel)
	return row, mapWalletTransportError(err)
}
func (a walletTransportAdapter) DeleteManualRechargeChannel(id uint) error {
	return mapWalletTransportError(a.wallets.DeleteManualRechargeChannel(id))
}
func (a walletTransportAdapter) CreateManualRechargeRequest(input walletcontract.ManualRechargeCreateInput) (*walletdomain.ManualRechargeRequest, error) {
	row, err := a.wallets.CreateManualRechargeRequest(input)
	return row, mapWalletTransportError(err)
}
func (a walletTransportAdapter) ListManualRechargeRequests(filter walletcontract.ManualRechargeListFilter) ([]walletdomain.ManualRechargeRequest, int64, error) {
	rows, total, err := a.wallets.ListManualRechargeRequests(filter)
	return rows, total, mapWalletTransportError(err)
}
func (a walletTransportAdapter) GetManualRechargeRequestByNo(userID uint, no string) (*walletdomain.ManualRechargeRequest, error) {
	row, err := a.wallets.GetManualRechargeRequestByNo(userID, no)
	return row, mapWalletTransportError(err)
}
func (a walletTransportAdapter) CancelManualRechargeRequest(userID uint, no string) (*walletdomain.ManualRechargeRequest, error) {
	row, err := a.wallets.CancelManualRechargeRequest(userID, no)
	return row, mapWalletTransportError(err)
}
func (a walletTransportAdapter) ApproveManualRechargeRequest(adminID, requestID uint, note string) (*walletdomain.ManualRechargeRequest, error) {
	row, err := a.wallets.ApproveManualRechargeRequest(adminID, requestID, note)
	return row, mapWalletTransportError(err)
}
func (a walletTransportAdapter) RejectManualRechargeRequest(adminID, requestID uint, note string) (*walletdomain.ManualRechargeRequest, error) {
	row, err := a.wallets.RejectManualRechargeRequest(adminID, requestID, note)
	return row, mapWalletTransportError(err)
}

func mapWalletTransportError(err error) error {
	if err == nil {
		return nil
	}
	for _, mapping := range []struct {
		source error
		target error
	}{
		{walletcontract.ErrInvalidAmount, wallettransport.ErrInvalidAmount},
		{walletcontract.ErrInsufficientBalance, wallettransport.ErrInsufficientBalance},
		{walletcontract.ErrNotSupportedForGuest, wallettransport.ErrNotSupportedForGuest},
		{walletcontract.ErrRechargeNotFound, wallettransport.ErrRechargeNotFound},
		{paymentapp.ErrPaymentInvalid, wallettransport.ErrPaymentInvalid},
		{paymentapp.ErrPaymentNotFound, wallettransport.ErrPaymentNotFound},
		{orderapp.ErrOrderNotFound, wallettransport.ErrOrderNotFound},
		{orderapp.ErrOrderStatusInvalid, wallettransport.ErrOrderStatusInvalid},
		{paymentapp.ErrPaymentChannelNotFound, wallettransport.ErrPaymentChannelNotFound},
		{paymentapp.ErrPaymentChannelInactive, wallettransport.ErrPaymentChannelInactive},
		{paymentapp.ErrPaymentProviderNotSupported, wallettransport.ErrPaymentProviderNotSupported},
		{paymentapp.ErrPaymentChannelConfigInvalid, wallettransport.ErrPaymentChannelConfigInvalid},
		{paymentapp.ErrPaymentGatewayRequestFailed, wallettransport.ErrPaymentGatewayRequestFailed},
		{paymentapp.ErrPaymentGatewayResponseInvalid, wallettransport.ErrPaymentGatewayResponseInvalid},
		{paymentapp.ErrPaymentCurrencyMismatch, wallettransport.ErrPaymentCurrencyMismatch},
		{paymentapp.ErrPaymentChannelNotAllowedForProduct, wallettransport.ErrPaymentChannelNotAllowedProduct},
		{paymentapp.ErrPaymentChannelNotAllowedForRecharge, wallettransport.ErrPaymentChannelNotAllowedRecharge},
		{walletcontract.ErrOnlyPaymentRequired, wallettransport.ErrWalletOnlyPaymentRequired},
		{paymentapp.ErrPaymentStatusInvalid, wallettransport.ErrPaymentStatusInvalid},
		{paymentapp.ErrPaymentAmountMismatch, wallettransport.ErrPaymentAmountMismatch},
	} {
		if errors.Is(err, mapping.source) {
			return fmt.Errorf("%w: %v", mapping.target, err)
		}
	}
	return err
}

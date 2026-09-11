package application

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	walletcontract "github.com/dujiao-next/internal/modules/wallet/contract"
	walletdomain "github.com/dujiao-next/internal/modules/wallet/domain"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/shopspring/decimal"
)

var (
	ErrManualRechargeNotFound      = errors.New("manual recharge request not found")
	ErrManualRechargeActive        = errors.New("manual recharge request already active")
	ErrManualRechargeInvalid       = errors.New("manual recharge request invalid")
	ErrManualRechargeStatusInvalid = errors.New("manual recharge request status invalid")
)

func (s *Service) ListManualRechargeChannels(activeOnly bool) ([]walletdomain.ManualRechargeChannel, error) {
	return s.repository.ListManualRechargeChannels(activeOnly)
}

func (s *Service) SaveManualRechargeChannel(channel *walletdomain.ManualRechargeChannel) (*walletdomain.ManualRechargeChannel, error) {
	if channel == nil || strings.TrimSpace(channel.Name) == "" || (channel.Type != "wechat" && channel.Type != "alipay") || strings.TrimSpace(channel.QRCodeURL) == "" {
		return nil, ErrManualRechargeInvalid
	}
	if channel.MinAmount.Decimal.LessThan(decimal.Zero) || channel.MaxAmount.Decimal.LessThan(decimal.Zero) || (channel.MaxAmount.Decimal.GreaterThan(decimal.Zero) && channel.MaxAmount.Decimal.LessThan(channel.MinAmount.Decimal)) {
		return nil, ErrManualRechargeInvalid
	}
	channel.Name = strings.TrimSpace(channel.Name)
	channel.QRCodeURL = strings.TrimSpace(channel.QRCodeURL)
	channel.AccountName = strings.TrimSpace(channel.AccountName)
	channel.Instructions = strings.TrimSpace(channel.Instructions)
	channel.UpdatedAt = time.Now()
	if channel.ID == 0 {
		channel.CreatedAt = channel.UpdatedAt
		if err := s.repository.CreateManualRechargeChannel(channel); err != nil {
			return nil, err
		}
		return channel, nil
	}
	previous, err := s.repository.GetManualRechargeChannel(channel.ID)
	if err != nil {
		return nil, err
	}
	if previous == nil {
		return nil, ErrManualRechargeNotFound
	}
	channel.CreatedAt = previous.CreatedAt
	if err := s.repository.UpdateManualRechargeChannel(channel); err != nil {
		return nil, err
	}
	return channel, nil
}

func (s *Service) DeleteManualRechargeChannel(id uint) error {
	channel, err := s.repository.GetManualRechargeChannel(id)
	if err != nil {
		return err
	}
	if channel == nil {
		return ErrManualRechargeNotFound
	}
	now := time.Now()
	channel.DeletedAt = &now
	return s.repository.UpdateManualRechargeChannel(channel)
}

func (s *Service) CreateManualRechargeRequest(input walletcontract.ManualRechargeCreateInput) (*walletdomain.ManualRechargeRequest, error) {
	if input.UserID == 0 || input.ChannelID == 0 || input.Amount.Decimal.Round(2).LessThanOrEqual(decimal.Zero) || strings.TrimSpace(input.TransactionNo) == "" || strings.TrimSpace(input.ProofURL) == "" {
		return nil, ErrManualRechargeInvalid
	}
	contactType := strings.ToLower(strings.TrimSpace(input.ContactType))
	if (contactType != "email" && contactType != "tg") || strings.TrimSpace(input.ContactValue) == "" {
		return nil, ErrManualRechargeInvalid
	}
	if s.transactions == nil {
		return nil, walletcontract.ErrTransactionRequired
	}
	var result *walletdomain.ManualRechargeRequest
	err := s.transactions.WithinTransaction(func(tx walletcontract.Transaction) error {
		repo := tx.Wallets()
		channel, err := repo.GetManualRechargeChannelForUpdate(input.ChannelID)
		if err != nil {
			return err
		}
		if channel == nil || !channel.Enabled {
			return ErrManualRechargeNotFound
		}
		amount := input.Amount.Decimal.Round(2)
		if (channel.MinAmount.Decimal.GreaterThan(decimal.Zero) && amount.LessThan(channel.MinAmount.Decimal)) || (channel.MaxAmount.Decimal.GreaterThan(decimal.Zero) && amount.GreaterThan(channel.MaxAmount.Decimal)) {
			return ErrManualRechargeInvalid
		}
		active, err := repo.GetActiveManualRechargeRequest(input.UserID)
		if err != nil {
			return err
		}
		if active != nil {
			// Old versions could leave active_user_id populated after a request had
			// already reached a terminal status. Release that stale unique-key claim
			// here so a cancelled/rejected/completed request never blocks a new one.
			if active.Status == constants.ManualRechargeStatusPending || active.Status == constants.ManualRechargeStatusProcessing {
				return ErrManualRechargeActive
			}
			active.ActiveUserID = nil
			active.UpdatedAt = time.Now()
			if err := repo.UpdateManualRechargeRequest(active); err != nil {
				return err
			}
		}
		now := time.Now()
		userID := input.UserID
		request := &walletdomain.ManualRechargeRequest{RequestNo: fmt.Sprintf("MR%d%d", now.UnixMilli(), input.UserID), UserID: input.UserID, ChannelID: input.ChannelID, ActiveUserID: &userID, Amount: money.FromDecimal(amount), Currency: normalizeCurrency(input.Currency), TransactionNo: strings.TrimSpace(input.TransactionNo), ContactType: contactType, ContactValue: strings.TrimSpace(input.ContactValue), ProofURL: strings.TrimSpace(input.ProofURL), Remark: strings.TrimSpace(input.Remark), Status: constants.ManualRechargeStatusPending, CreatedAt: now, UpdatedAt: now}
		if err := repo.CreateManualRechargeRequest(request); err != nil {
			// The active_user_id unique index wins races even if both requests passed the read.
			return ErrManualRechargeActive
		}
		result = request
		return nil
	})
	return result, err
}

func (s *Service) ListManualRechargeRequests(filter walletcontract.ManualRechargeListFilter) ([]walletdomain.ManualRechargeRequest, int64, error) {
	return s.repository.ListManualRechargeRequests(filter)
}
func (s *Service) GetManualRechargeRequestByNo(userID uint, no string) (*walletdomain.ManualRechargeRequest, error) {
	r, err := s.repository.GetManualRechargeRequestByNo(userID, no)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrManualRechargeNotFound
	}
	return r, nil
}

func (s *Service) CancelManualRechargeRequest(userID uint, no string) (*walletdomain.ManualRechargeRequest, error) {
	r, err := s.GetManualRechargeRequestByNo(userID, no)
	if err != nil {
		return nil, err
	}
	if r.Status != constants.ManualRechargeStatusPending {
		return nil, ErrManualRechargeStatusInvalid
	}
	r.Status = constants.ManualRechargeStatusCancelled
	r.ActiveUserID = nil
	r.UpdatedAt = time.Now()
	err = s.repository.UpdateManualRechargeRequest(r)
	return r, err
}

func (s *Service) ApproveManualRechargeRequest(adminID, requestID uint, note string) (*walletdomain.ManualRechargeRequest, error) {
	if adminID == 0 || requestID == 0 {
		return nil, ErrManualRechargeInvalid
	}
	var result *walletdomain.ManualRechargeRequest
	err := s.transactions.WithinTransaction(func(tx walletcontract.Transaction) error {
		repo := tx.Wallets()
		request, err := repo.GetManualRechargeRequestForUpdate(requestID)
		if err != nil {
			return err
		}
		if request == nil {
			return ErrManualRechargeNotFound
		}
		if request.Status == constants.ManualRechargeStatusApproved {
			result = request
			return nil
		}
		if request.Status != constants.ManualRechargeStatusPending && request.Status != constants.ManualRechargeStatusProcessing {
			return ErrManualRechargeStatusInvalid
		}
		now := time.Now()
		operatorID := adminID
		_, transaction, err := s.CreditInTransaction(tx, walletcontract.CreditInput{UserID: request.UserID, Amount: request.Amount, Currency: request.Currency, Type: constants.WalletTxnTypeManualRecharge, Reference: fmt.Sprintf("manual_recharge:%d:approved", request.ID), Remark: cleanRemark(note, "扫码人工充值到账"), OperatorAdminID: &operatorID})
		if err != nil {
			return err
		}
		request.Status = constants.ManualRechargeStatusApproved
		request.ActiveUserID = nil
		request.ReviewedBy = &operatorID
		request.ReviewedAt = &now
		request.CreditedAt = &now
		request.WalletTransactionID = &transaction.ID
		request.ReviewNote = strings.TrimSpace(note)
		request.UpdatedAt = now
		if err := repo.UpdateManualRechargeRequest(request); err != nil {
			return err
		}
		result = request
		return nil
	})
	return result, err
}

func (s *Service) RejectManualRechargeRequest(adminID, requestID uint, note string) (*walletdomain.ManualRechargeRequest, error) {
	if adminID == 0 || requestID == 0 || strings.TrimSpace(note) == "" {
		return nil, ErrManualRechargeInvalid
	}
	request, err := s.repository.GetManualRechargeRequest(requestID)
	if err != nil {
		return nil, err
	}
	if request == nil {
		return nil, ErrManualRechargeNotFound
	}
	if request.Status != constants.ManualRechargeStatusPending && request.Status != constants.ManualRechargeStatusProcessing {
		return nil, ErrManualRechargeStatusInvalid
	}
	now := time.Now()
	request.Status = constants.ManualRechargeStatusRejected
	request.ActiveUserID = nil
	request.ReviewedBy = &adminID
	request.ReviewedAt = &now
	request.ReviewNote = strings.TrimSpace(note)
	request.UpdatedAt = now
	return request, s.repository.UpdateManualRechargeRequest(request)
}

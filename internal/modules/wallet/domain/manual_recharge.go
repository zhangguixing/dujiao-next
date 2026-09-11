package domain

import (
	"time"

	"github.com/dujiao-next/internal/shared/money"
)

// ManualRechargeChannel is an operator-managed QR-code collection method.
type ManualRechargeChannel struct {
	ID           uint         `gorm:"primarykey" json:"id"`
	Name         string       `gorm:"type:varchar(80);not null" json:"name"`
	Type         string       `gorm:"type:varchar(20);index;not null" json:"type"`
	QRCodeURL    string       `gorm:"type:varchar(500);not null" json:"qr_code_url"`
	AccountName  string       `gorm:"type:varchar(120)" json:"account_name"`
	Instructions string       `gorm:"type:text" json:"instructions"`
	MinAmount    money.Amount `gorm:"type:decimal(20,2);not null;default:0" json:"min_amount"`
	MaxAmount    money.Amount `gorm:"type:decimal(20,2);not null;default:0" json:"max_amount"`
	Enabled      bool         `gorm:"not null;default:true;index" json:"enabled"`
	Sort         int          `gorm:"not null;default:0" json:"sort"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	DeletedAt    *time.Time   `gorm:"index" json:"-"`
}

func (ManualRechargeChannel) TableName() string { return "manual_recharge_channels" }

// ManualRechargeRequest represents a submitted offline transfer waiting for review.
// ActiveUserID is non-nil only while the request is open; its unique index is the
// database-level guard that permits one active request per user.
type ManualRechargeRequest struct {
	ID                  uint         `gorm:"primarykey" json:"id"`
	RequestNo           string       `gorm:"type:varchar(40);uniqueIndex;not null" json:"request_no"`
	UserID              uint         `gorm:"index;not null" json:"user_id"`
	ChannelID           uint         `gorm:"index;not null;uniqueIndex:idx_manual_recharge_channel_txn" json:"channel_id"`
	ActiveUserID        *uint        `gorm:"uniqueIndex" json:"-"`
	Amount              money.Amount `gorm:"type:decimal(20,2);not null" json:"amount"`
	Currency            string       `gorm:"type:varchar(16);not null;default:'CNY'" json:"currency"`
	TransactionNo       string       `gorm:"type:varchar(160);not null;uniqueIndex:idx_manual_recharge_channel_txn" json:"transaction_no"`
	ContactType         string       `gorm:"type:varchar(16);not null" json:"contact_type"`
	ContactValue        string       `gorm:"type:varchar(180);not null" json:"contact_value"`
	ProofURL            string       `gorm:"type:varchar(500);not null" json:"proof_url"`
	Remark              string       `gorm:"type:varchar(500)" json:"remark"`
	Status              string       `gorm:"type:varchar(20);index;not null" json:"status"`
	ReviewNote          string       `gorm:"type:varchar(500)" json:"review_note"`
	ReviewedBy          *uint        `gorm:"index" json:"reviewed_by,omitempty"`
	ReviewedAt          *time.Time   `gorm:"index" json:"reviewed_at,omitempty"`
	WalletTransactionID *uint        `gorm:"uniqueIndex" json:"wallet_transaction_id,omitempty"`
	CreditedAt          *time.Time   `gorm:"index" json:"credited_at,omitempty"`
	CreatedAt           time.Time    `gorm:"index" json:"created_at"`
	UpdatedAt           time.Time    `json:"updated_at"`
	DeletedAt           *time.Time   `gorm:"index" json:"-"`
}

func (ManualRechargeRequest) TableName() string { return "manual_recharge_requests" }

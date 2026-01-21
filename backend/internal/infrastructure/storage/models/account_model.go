package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	pkgmodels "github.com/smilemakc/mbflow/pkg/models"
)

// BillingAccountModel represents a billing account in the database
type BillingAccountModel struct {
	bun.BaseModel `bun:"table:mbflow_billing_accounts,alias:ba"`

	ID        uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID `bun:"user_id,notnull,type:uuid" json:"user_id" validate:"required"`
	Balance   float64   `bun:"balance,notnull,default:0" json:"balance" validate:"min=0"`
	Currency  string    `bun:"currency,notnull,default:'USD'" json:"currency" validate:"required,len=3"`
	Status    string    `bun:"status,notnull,default:'active'" json:"status" validate:"required,oneof=active suspended closed"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`

	// Relations
	User         *UserModel          `bun:"rel:belongs-to,join:user_id=id" json:"user,omitempty"`
	Transactions []*TransactionModel `bun:"rel:has-many,join:id=account_id" json:"transactions,omitempty"`
}

// TableName returns the table name for BillingAccountModel
func (BillingAccountModel) TableName() string {
	return "mbflow_billing_accounts"
}

// BeforeInsert hook to set timestamps and defaults
func (b *BillingAccountModel) BeforeInsert(ctx interface{}) error {
	now := time.Now()
	b.CreatedAt = now
	b.UpdatedAt = now
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	if b.Currency == "" {
		b.Currency = "USD"
	}
	if b.Status == "" {
		b.Status = string(pkgmodels.AccountStatusActive)
	}
	return nil
}

// BeforeUpdate hook to update timestamp
func (b *BillingAccountModel) BeforeUpdate(ctx interface{}) error {
	b.UpdatedAt = time.Now()
	return nil
}

// IsActive returns true if account is active
func (b *BillingAccountModel) IsActive() bool {
	return b.Status == string(pkgmodels.AccountStatusActive)
}

// HasSufficientBalance checks if the account has sufficient balance
func (b *BillingAccountModel) HasSufficientBalance(amount float64) bool {
	return b.Balance >= amount
}

// CanCharge checks if the account can be charged the specified amount
func (b *BillingAccountModel) CanCharge(amount float64) bool {
	return b.IsActive() && b.HasSufficientBalance(amount)
}

// TransactionModel represents a financial transaction in the database
type TransactionModel struct {
	bun.BaseModel `bun:"table:mbflow_transactions,alias:t"`

	ID             uuid.UUID  `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	AccountID      uuid.UUID  `bun:"account_id,notnull,type:uuid" json:"account_id" validate:"required"`
	Type           string     `bun:"type,notnull" json:"type" validate:"required,oneof=deposit charge refund adjustment"`
	Amount         float64    `bun:"amount,notnull" json:"amount" validate:"required,gt=0"`
	Currency       string     `bun:"currency,notnull" json:"currency" validate:"required,len=3"`
	Status         string     `bun:"status,notnull,default:'completed'" json:"status" validate:"required,oneof=pending completed failed reversed"`
	Description    string     `bun:"description" json:"description,omitempty" validate:"max=500"`
	ReferenceType  string     `bun:"reference_type" json:"reference_type,omitempty" validate:"max=100"`
	ReferenceID    *uuid.UUID `bun:"reference_id,type:uuid" json:"reference_id,omitempty"`
	IdempotencyKey string     `bun:"idempotency_key,notnull" json:"idempotency_key" validate:"required,max=255"`
	BalanceBefore  float64    `bun:"balance_before,notnull" json:"balance_before" validate:"min=0"`
	BalanceAfter   float64    `bun:"balance_after,notnull" json:"balance_after" validate:"min=0"`
	Metadata       JSONBMap   `bun:"metadata,type:jsonb,default:'{}'" json:"metadata,omitempty"`
	CreatedAt      time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`

	// Relations
	Account *BillingAccountModel `bun:"rel:belongs-to,join:account_id=id" json:"account,omitempty"`
}

// TableName returns the table name for TransactionModel
func (TransactionModel) TableName() string {
	return "mbflow_transactions"
}

// BeforeInsert hook to set timestamps and defaults
func (t *TransactionModel) BeforeInsert(ctx interface{}) error {
	t.CreatedAt = time.Now()
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	if t.Metadata == nil {
		t.Metadata = make(JSONBMap)
	}
	if t.Status == "" {
		t.Status = string(pkgmodels.TransactionStatusCompleted)
	}
	return nil
}

// IsCompleted returns true if transaction is completed
func (t *TransactionModel) IsCompleted() bool {
	return t.Status == string(pkgmodels.TransactionStatusCompleted)
}

// IsPending returns true if transaction is pending
func (t *TransactionModel) IsPending() bool {
	return t.Status == string(pkgmodels.TransactionStatusPending)
}

// IsFailed returns true if transaction failed
func (t *TransactionModel) IsFailed() bool {
	return t.Status == string(pkgmodels.TransactionStatusFailed)
}

// IsReversed returns true if transaction was reversed
func (t *TransactionModel) IsReversed() bool {
	return t.Status == string(pkgmodels.TransactionStatusReversed)
}

// IsTerminal returns true if transaction is in a terminal state
func (t *TransactionModel) IsTerminal() bool {
	return t.Status == string(pkgmodels.TransactionStatusCompleted) ||
		t.Status == string(pkgmodels.TransactionStatusFailed) ||
		t.Status == string(pkgmodels.TransactionStatusReversed)
}

// ============================================================================
// Domain Model Conversions
// ============================================================================

// ToAccountDomain converts BillingAccountModel to domain Account
func ToAccountDomain(a *BillingAccountModel) *pkgmodels.Account {
	if a == nil {
		return nil
	}

	return &pkgmodels.Account{
		ID:        a.ID.String(),
		UserID:    a.UserID.String(),
		Balance:   a.Balance,
		Currency:  a.Currency,
		Status:    pkgmodels.AccountStatus(a.Status),
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
}

// ToTransactionDomain converts TransactionModel to domain Transaction
func ToTransactionDomain(t *TransactionModel) *pkgmodels.Transaction {
	if t == nil {
		return nil
	}

	var metadata map[string]interface{}
	if t.Metadata != nil {
		metadata = t.Metadata
	}

	var referenceID string
	if t.ReferenceID != nil {
		referenceID = t.ReferenceID.String()
	}

	return &pkgmodels.Transaction{
		ID:             t.ID.String(),
		AccountID:      t.AccountID.String(),
		Type:           pkgmodels.TransactionType(t.Type),
		Amount:         t.Amount,
		Currency:       t.Currency,
		Status:         pkgmodels.TransactionStatus(t.Status),
		Description:    t.Description,
		ReferenceType:  t.ReferenceType,
		ReferenceID:    referenceID,
		IdempotencyKey: t.IdempotencyKey,
		BalanceBefore:  t.BalanceBefore,
		BalanceAfter:   t.BalanceAfter,
		Metadata:       metadata,
		CreatedAt:      t.CreatedAt,
	}
}

// FromAccountDomain converts domain Account to BillingAccountModel
func FromAccountDomain(account *pkgmodels.Account) *BillingAccountModel {
	if account == nil {
		return nil
	}

	var accountID uuid.UUID
	if account.ID != "" {
		accountID = uuid.MustParse(account.ID)
	}

	var userID uuid.UUID
	if account.UserID != "" {
		userID = uuid.MustParse(account.UserID)
	}

	return &BillingAccountModel{
		ID:        accountID,
		UserID:    userID,
		Balance:   account.Balance,
		Currency:  account.Currency,
		Status:    string(account.Status),
		CreatedAt: account.CreatedAt,
		UpdatedAt: account.UpdatedAt,
	}
}

// FromTransactionDomain converts domain Transaction to TransactionModel
func FromTransactionDomain(transaction *pkgmodels.Transaction) *TransactionModel {
	if transaction == nil {
		return nil
	}

	var transactionID uuid.UUID
	if transaction.ID != "" {
		transactionID = uuid.MustParse(transaction.ID)
	}

	var accountID uuid.UUID
	if transaction.AccountID != "" {
		accountID = uuid.MustParse(transaction.AccountID)
	}

	var referenceID *uuid.UUID
	if transaction.ReferenceID != "" {
		refID := uuid.MustParse(transaction.ReferenceID)
		referenceID = &refID
	}

	var metadata JSONBMap
	if transaction.Metadata != nil {
		metadata = JSONBMap(transaction.Metadata)
	}

	return &TransactionModel{
		ID:             transactionID,
		AccountID:      accountID,
		Type:           string(transaction.Type),
		Amount:         transaction.Amount,
		Currency:       transaction.Currency,
		Status:         string(transaction.Status),
		Description:    transaction.Description,
		ReferenceType:  transaction.ReferenceType,
		ReferenceID:    referenceID,
		IdempotencyKey: transaction.IdempotencyKey,
		BalanceBefore:  transaction.BalanceBefore,
		BalanceAfter:   transaction.BalanceAfter,
		Metadata:       metadata,
		CreatedAt:      transaction.CreatedAt,
	}
}

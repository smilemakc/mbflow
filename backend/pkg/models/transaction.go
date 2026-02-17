package models

import "time"

// TransactionType defines the type of financial transaction
type TransactionType string

const (
	TransactionTypeDeposit    TransactionType = "deposit"
	TransactionTypeCharge     TransactionType = "charge"
	TransactionTypeRefund     TransactionType = "refund"
	TransactionTypeAdjustment TransactionType = "adjustment"
)

// TransactionStatus defines the status of a transaction
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusCompleted TransactionStatus = "completed"
	TransactionStatusFailed    TransactionStatus = "failed"
	TransactionStatusReversed  TransactionStatus = "reversed"
)

// Transaction represents a financial operation on an account
type Transaction struct {
	ID             string            `json:"id"`
	AccountID      string            `json:"account_id"`
	Type           TransactionType   `json:"type"`
	Amount         float64           `json:"amount"`
	Currency       string            `json:"currency"`
	Status         TransactionStatus `json:"status"`
	Description    string            `json:"description,omitempty"`
	ReferenceType  string            `json:"reference_type,omitempty"`
	ReferenceID    string            `json:"reference_id,omitempty"`
	IdempotencyKey string            `json:"idempotency_key"`
	BalanceBefore  float64           `json:"balance_before"`
	BalanceAfter   float64           `json:"balance_after"`
	Metadata       map[string]any    `json:"metadata,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
}

// Validate validates the transaction structure
func (t *Transaction) Validate() error {
	if t.AccountID == "" {
		return &ValidationError{Field: "account_id", Message: "account ID is required"}
	}
	if t.Amount <= 0 {
		return &ValidationError{Field: "amount", Message: "amount must be positive"}
	}
	if t.IdempotencyKey == "" {
		return &ValidationError{Field: "idempotency_key", Message: "idempotency key is required"}
	}
	if t.Currency == "" {
		return &ValidationError{Field: "currency", Message: "currency is required"}
	}
	return nil
}

// IsCompleted checks if the transaction is completed
func (t *Transaction) IsCompleted() bool {
	return t.Status == TransactionStatusCompleted
}

// IsPending checks if the transaction is pending
func (t *Transaction) IsPending() bool {
	return t.Status == TransactionStatusPending
}

// IsFailed checks if the transaction failed
func (t *Transaction) IsFailed() bool {
	return t.Status == TransactionStatusFailed
}

// IsReversed checks if the transaction was reversed
func (t *Transaction) IsReversed() bool {
	return t.Status == TransactionStatusReversed
}

// IsTerminal checks if the transaction is in a terminal state
func (t *Transaction) IsTerminal() bool {
	return t.Status == TransactionStatusCompleted ||
		t.Status == TransactionStatusFailed ||
		t.Status == TransactionStatusReversed
}

// Complete marks the transaction as completed
func (t *Transaction) Complete() {
	t.Status = TransactionStatusCompleted
}

// Fail marks the transaction as failed
func (t *Transaction) Fail() {
	t.Status = TransactionStatusFailed
}

// Reverse marks the transaction as reversed
func (t *Transaction) Reverse() {
	t.Status = TransactionStatusReversed
}

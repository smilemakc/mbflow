package models

import "time"

// AccountStatus defines the status of a billing account
type AccountStatus string

const (
	AccountStatusActive    AccountStatus = "active"
	AccountStatusSuspended AccountStatus = "suspended"
	AccountStatusClosed    AccountStatus = "closed"
)

// Account represents a billing account for a user
type Account struct {
	ID        string        `json:"id"`
	UserID    string        `json:"user_id"`
	Balance   float64       `json:"balance"`
	Currency  string        `json:"currency"`
	Status    AccountStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// NewAccount creates a new account for a user with zero balance
func NewAccount(userID string) *Account {
	now := time.Now()
	return &Account{
		UserID:    userID,
		Balance:   0,
		Currency:  "USD",
		Status:    AccountStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Validate validates the account structure
func (a *Account) Validate() error {
	if a.UserID == "" {
		return &ValidationError{Field: "user_id", Message: "user ID is required"}
	}
	if a.Currency == "" {
		return &ValidationError{Field: "currency", Message: "currency is required"}
	}
	return nil
}

// HasSufficientBalance checks if the account has sufficient balance
func (a *Account) HasSufficientBalance(amount float64) bool {
	return a.Balance >= amount
}

// IsActive checks if the account is active
func (a *Account) IsActive() bool {
	return a.Status == AccountStatusActive
}

// CanCharge checks if the account can be charged the specified amount
func (a *Account) CanCharge(amount float64) bool {
	return a.IsActive() && a.HasSufficientBalance(amount)
}

// Deposit adds funds to the account
func (a *Account) Deposit(amount float64) error {
	if amount <= 0 {
		return &ValidationError{Field: "amount", Message: "deposit amount must be positive"}
	}
	a.Balance += amount
	a.UpdatedAt = time.Now()
	return nil
}

// Charge deducts funds from the account
func (a *Account) Charge(amount float64) error {
	if amount <= 0 {
		return &ValidationError{Field: "amount", Message: "charge amount must be positive"}
	}
	if !a.CanCharge(amount) {
		return ErrInsufficientBalance
	}
	a.Balance -= amount
	a.UpdatedAt = time.Now()
	return nil
}

// Suspend suspends the account
func (a *Account) Suspend() {
	a.Status = AccountStatusSuspended
	a.UpdatedAt = time.Now()
}

// Activate activates the account
func (a *Account) Activate() {
	a.Status = AccountStatusActive
	a.UpdatedAt = time.Now()
}

// Close closes the account
func (a *Account) Close() {
	a.Status = AccountStatusClosed
	a.UpdatedAt = time.Now()
}

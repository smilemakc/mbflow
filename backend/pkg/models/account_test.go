package models

import (
	"testing"
)

func TestNewAccount(t *testing.T) {
	account := NewAccount("user-123")

	if account.UserID != "user-123" {
		t.Errorf("UserID = %v, want %v", account.UserID, "user-123")
	}
	if account.Balance != 0 {
		t.Errorf("Balance = %v, want %v", account.Balance, 0)
	}
	if account.Currency != "USD" {
		t.Errorf("Currency = %v, want %v", account.Currency, "USD")
	}
	if account.Status != AccountStatusActive {
		t.Errorf("Status = %v, want %v", account.Status, AccountStatusActive)
	}
}

func TestAccount_Validate(t *testing.T) {
	tests := []struct {
		name    string
		account *Account
		wantErr bool
	}{
		{
			name:    "valid account",
			account: NewAccount("user-123"),
			wantErr: false,
		},
		{
			name: "missing user ID",
			account: &Account{
				Currency: "USD",
			},
			wantErr: true,
		},
		{
			name: "missing currency",
			account: &Account{
				UserID: "user-123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.account.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Account.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAccount_HasSufficientBalance(t *testing.T) {
	tests := []struct {
		name     string
		balance  float64
		amount   float64
		expected bool
	}{
		{
			name:     "sufficient balance",
			balance:  100,
			amount:   50,
			expected: true,
		},
		{
			name:     "exact balance",
			balance:  100,
			amount:   100,
			expected: true,
		},
		{
			name:     "insufficient balance",
			balance:  100,
			amount:   150,
			expected: false,
		},
		{
			name:     "zero balance",
			balance:  0,
			amount:   1,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := NewAccount("user-123")
			account.Balance = tt.balance

			got := account.HasSufficientBalance(tt.amount)
			if got != tt.expected {
				t.Errorf("HasSufficientBalance() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAccount_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   AccountStatus
		expected bool
	}{
		{
			name:     "active account",
			status:   AccountStatusActive,
			expected: true,
		},
		{
			name:     "suspended account",
			status:   AccountStatusSuspended,
			expected: false,
		},
		{
			name:     "closed account",
			status:   AccountStatusClosed,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := NewAccount("user-123")
			account.Status = tt.status

			got := account.IsActive()
			if got != tt.expected {
				t.Errorf("IsActive() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAccount_CanCharge(t *testing.T) {
	tests := []struct {
		name     string
		balance  float64
		status   AccountStatus
		amount   float64
		expected bool
	}{
		{
			name:     "can charge active account with balance",
			balance:  100,
			status:   AccountStatusActive,
			amount:   50,
			expected: true,
		},
		{
			name:     "cannot charge suspended account",
			balance:  100,
			status:   AccountStatusSuspended,
			amount:   50,
			expected: false,
		},
		{
			name:     "cannot charge with insufficient balance",
			balance:  100,
			status:   AccountStatusActive,
			amount:   150,
			expected: false,
		},
		{
			name:     "cannot charge closed account",
			balance:  100,
			status:   AccountStatusClosed,
			amount:   50,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := NewAccount("user-123")
			account.Balance = tt.balance
			account.Status = tt.status

			got := account.CanCharge(tt.amount)
			if got != tt.expected {
				t.Errorf("CanCharge() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAccount_Deposit(t *testing.T) {
	tests := []struct {
		name            string
		initialBalance  float64
		amount          float64
		wantErr         bool
		expectedBalance float64
	}{
		{
			name:            "deposit positive amount",
			initialBalance:  100,
			amount:          50,
			wantErr:         false,
			expectedBalance: 150,
		},
		{
			name:            "deposit to zero balance",
			initialBalance:  0,
			amount:          100,
			wantErr:         false,
			expectedBalance: 100,
		},
		{
			name:            "cannot deposit zero",
			initialBalance:  100,
			amount:          0,
			wantErr:         true,
			expectedBalance: 100,
		},
		{
			name:            "cannot deposit negative",
			initialBalance:  100,
			amount:          -50,
			wantErr:         true,
			expectedBalance: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := NewAccount("user-123")
			account.Balance = tt.initialBalance

			err := account.Deposit(tt.amount)
			if (err != nil) != tt.wantErr {
				t.Errorf("Deposit() error = %v, wantErr %v", err, tt.wantErr)
			}
			if account.Balance != tt.expectedBalance {
				t.Errorf("Balance = %v, want %v", account.Balance, tt.expectedBalance)
			}
		})
	}
}

func TestAccount_Charge(t *testing.T) {
	tests := []struct {
		name            string
		initialBalance  float64
		status          AccountStatus
		amount          float64
		wantErr         bool
		expectedBalance float64
	}{
		{
			name:            "charge active account",
			initialBalance:  100,
			status:          AccountStatusActive,
			amount:          50,
			wantErr:         false,
			expectedBalance: 50,
		},
		{
			name:            "charge exact balance",
			initialBalance:  100,
			status:          AccountStatusActive,
			amount:          100,
			wantErr:         false,
			expectedBalance: 0,
		},
		{
			name:            "cannot charge insufficient balance",
			initialBalance:  100,
			status:          AccountStatusActive,
			amount:          150,
			wantErr:         true,
			expectedBalance: 100,
		},
		{
			name:            "cannot charge suspended account",
			initialBalance:  100,
			status:          AccountStatusSuspended,
			amount:          50,
			wantErr:         true,
			expectedBalance: 100,
		},
		{
			name:            "cannot charge zero",
			initialBalance:  100,
			status:          AccountStatusActive,
			amount:          0,
			wantErr:         true,
			expectedBalance: 100,
		},
		{
			name:            "cannot charge negative",
			initialBalance:  100,
			status:          AccountStatusActive,
			amount:          -50,
			wantErr:         true,
			expectedBalance: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := NewAccount("user-123")
			account.Balance = tt.initialBalance
			account.Status = tt.status

			err := account.Charge(tt.amount)
			if (err != nil) != tt.wantErr {
				t.Errorf("Charge() error = %v, wantErr %v", err, tt.wantErr)
			}
			if account.Balance != tt.expectedBalance {
				t.Errorf("Balance = %v, want %v", account.Balance, tt.expectedBalance)
			}
		})
	}
}

func TestAccount_StatusTransitions(t *testing.T) {
	account := NewAccount("user-123")

	if account.Status != AccountStatusActive {
		t.Errorf("Initial status = %v, want %v", account.Status, AccountStatusActive)
	}

	account.Suspend()
	if account.Status != AccountStatusSuspended {
		t.Errorf("Status after Suspend() = %v, want %v", account.Status, AccountStatusSuspended)
	}

	account.Activate()
	if account.Status != AccountStatusActive {
		t.Errorf("Status after Activate() = %v, want %v", account.Status, AccountStatusActive)
	}

	account.Close()
	if account.Status != AccountStatusClosed {
		t.Errorf("Status after Close() = %v, want %v", account.Status, AccountStatusClosed)
	}
}

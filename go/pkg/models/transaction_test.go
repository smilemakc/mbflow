package models

import (
	"testing"
	"time"
)

func TestTransaction_Validate(t *testing.T) {
	tests := []struct {
		name        string
		transaction *Transaction
		wantErr     bool
	}{
		{
			name: "valid transaction",
			transaction: &Transaction{
				AccountID:      "acc-123",
				Type:           TransactionTypeDeposit,
				Amount:         100,
				Currency:       "USD",
				IdempotencyKey: "key-123",
			},
			wantErr: false,
		},
		{
			name: "missing account ID",
			transaction: &Transaction{
				Type:           TransactionTypeDeposit,
				Amount:         100,
				Currency:       "USD",
				IdempotencyKey: "key-123",
			},
			wantErr: true,
		},
		{
			name: "zero amount",
			transaction: &Transaction{
				AccountID:      "acc-123",
				Type:           TransactionTypeDeposit,
				Amount:         0,
				Currency:       "USD",
				IdempotencyKey: "key-123",
			},
			wantErr: true,
		},
		{
			name: "negative amount",
			transaction: &Transaction{
				AccountID:      "acc-123",
				Type:           TransactionTypeDeposit,
				Amount:         -100,
				Currency:       "USD",
				IdempotencyKey: "key-123",
			},
			wantErr: true,
		},
		{
			name: "missing idempotency key",
			transaction: &Transaction{
				AccountID: "acc-123",
				Type:      TransactionTypeDeposit,
				Amount:    100,
				Currency:  "USD",
			},
			wantErr: true,
		},
		{
			name: "missing currency",
			transaction: &Transaction{
				AccountID:      "acc-123",
				Type:           TransactionTypeDeposit,
				Amount:         100,
				IdempotencyKey: "key-123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.transaction.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Transaction.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransaction_StatusChecks(t *testing.T) {
	transaction := &Transaction{
		AccountID:      "acc-123",
		Type:           TransactionTypeDeposit,
		Amount:         100,
		Currency:       "USD",
		Status:         TransactionStatusPending,
		IdempotencyKey: "key-123",
		CreatedAt:      time.Now(),
	}

	if !transaction.IsPending() {
		t.Error("IsPending() should return true for pending transaction")
	}
	if transaction.IsCompleted() {
		t.Error("IsCompleted() should return false for pending transaction")
	}
	if transaction.IsFailed() {
		t.Error("IsFailed() should return false for pending transaction")
	}
	if transaction.IsReversed() {
		t.Error("IsReversed() should return false for pending transaction")
	}
	if transaction.IsTerminal() {
		t.Error("IsTerminal() should return false for pending transaction")
	}

	transaction.Complete()
	if transaction.Status != TransactionStatusCompleted {
		t.Errorf("Status = %v, want %v", transaction.Status, TransactionStatusCompleted)
	}
	if !transaction.IsCompleted() {
		t.Error("IsCompleted() should return true after Complete()")
	}
	if !transaction.IsTerminal() {
		t.Error("IsTerminal() should return true for completed transaction")
	}

	transaction.Status = TransactionStatusPending
	transaction.Fail()
	if transaction.Status != TransactionStatusFailed {
		t.Errorf("Status = %v, want %v", transaction.Status, TransactionStatusFailed)
	}
	if !transaction.IsFailed() {
		t.Error("IsFailed() should return true after Fail()")
	}
	if !transaction.IsTerminal() {
		t.Error("IsTerminal() should return true for failed transaction")
	}

	transaction.Status = TransactionStatusPending
	transaction.Reverse()
	if transaction.Status != TransactionStatusReversed {
		t.Errorf("Status = %v, want %v", transaction.Status, TransactionStatusReversed)
	}
	if !transaction.IsReversed() {
		t.Error("IsReversed() should return true after Reverse()")
	}
	if !transaction.IsTerminal() {
		t.Error("IsTerminal() should return true for reversed transaction")
	}
}

func TestTransaction_Types(t *testing.T) {
	types := []TransactionType{
		TransactionTypeDeposit,
		TransactionTypeCharge,
		TransactionTypeRefund,
		TransactionTypeAdjustment,
	}

	for _, txType := range types {
		transaction := &Transaction{
			AccountID:      "acc-123",
			Type:           txType,
			Amount:         100,
			Currency:       "USD",
			IdempotencyKey: "key-123",
		}

		if err := transaction.Validate(); err != nil {
			t.Errorf("Valid transaction with type %v failed validation: %v", txType, err)
		}
	}
}

package repository

import (
	"context"

	"github.com/smilemakc/mbflow/pkg/models"
)

// AccountRepository defines the interface for billing account operations
type AccountRepository interface {
	// Create creates a new billing account for a user
	Create(ctx context.Context, account *models.Account) error

	// GetByID retrieves an account by its ID
	GetByID(ctx context.Context, id string) (*models.Account, error)

	// GetByUserID retrieves an account by user ID
	GetByUserID(ctx context.Context, userID string) (*models.Account, error)

	// Update updates an existing account
	Update(ctx context.Context, account *models.Account) error

	// UpdateBalance atomically updates account balance
	UpdateBalance(ctx context.Context, id string, newBalance float64) error

	// Suspend suspends an account
	Suspend(ctx context.Context, id string) error

	// Activate activates an account
	Activate(ctx context.Context, id string) error

	// Close closes an account
	Close(ctx context.Context, id string) error
}

// TransactionRepository defines the interface for transaction operations
type TransactionRepository interface {
	// Create creates a new transaction
	Create(ctx context.Context, tx *models.Transaction) error

	// GetByID retrieves a transaction by ID
	GetByID(ctx context.Context, id string) (*models.Transaction, error)

	// GetByIdempotencyKey retrieves a transaction by idempotency key
	GetByIdempotencyKey(ctx context.Context, key string) (*models.Transaction, error)

	// GetByAccountID retrieves transactions for an account with pagination
	GetByAccountID(ctx context.Context, accountID string, limit, offset int) ([]*models.Transaction, error)

	// GetByAccountIDAndType retrieves transactions of specific type for an account
	GetByAccountIDAndType(ctx context.Context, accountID string, txType models.TransactionType, limit, offset int) ([]*models.Transaction, error)

	// GetByReference retrieves transactions by reference (resource, execution, etc.)
	GetByReference(ctx context.Context, referenceType string, referenceID string) ([]*models.Transaction, error)

	// CountByAccountID counts total transactions for an account
	CountByAccountID(ctx context.Context, accountID string) (int64, error)
}

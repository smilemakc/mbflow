package storage

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/smilemakc/mbflow/go/testutil"
)

func rcCreateAccount(t *testing.T, db bun.IDB, balance float64) *pkgmodels.Account {
	t.Helper()
	userID := createCredentialsTestUser(t, db)
	acc := &pkgmodels.Account{
		UserID:   userID,
		Balance:  balance,
		Currency: "USD",
		Status:   pkgmodels.AccountStatusActive,
	}
	require.NoError(t, NewAccountRepository(db).Create(context.Background(), acc))
	require.NotEmpty(t, acc.ID)
	return acc
}

func TestRC_AccountRepository_CRUDAndStatus(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewAccountRepository(db)

	acc := rcCreateAccount(t, db, 10)
	_, err := uuid.Parse(acc.ID)
	require.NoError(t, err)
	assert.False(t, acc.CreatedAt.IsZero(), "created_at must be returned from the DB")

	got, err := repo.GetByID(ctx, acc.ID)
	require.NoError(t, err)
	assert.Equal(t, acc.UserID, got.UserID)
	assert.Equal(t, 10.0, got.Balance)
	assert.Equal(t, "USD", got.Currency)
	assert.Equal(t, pkgmodels.AccountStatusActive, got.Status)

	byUser, err := repo.GetByUserID(ctx, acc.UserID)
	require.NoError(t, err)
	assert.Equal(t, acc.ID, byUser.ID)

	// one account per user
	dup := &pkgmodels.Account{UserID: acc.UserID, Currency: "USD", Status: pkgmodels.AccountStatusActive}
	assert.Error(t, repo.Create(ctx, dup))

	got.Balance = 42
	got.Currency = "EUR"
	got.Status = pkgmodels.AccountStatusSuspended
	require.NoError(t, repo.Update(ctx, got))
	got, err = repo.GetByID(ctx, acc.ID)
	require.NoError(t, err)
	assert.Equal(t, 42.0, got.Balance)
	assert.Equal(t, "EUR", got.Currency)
	assert.Equal(t, pkgmodels.AccountStatusSuspended, got.Status)

	require.NoError(t, repo.UpdateBalance(ctx, acc.ID, 7.5))
	require.NoError(t, repo.Activate(ctx, acc.ID))
	got, _ = repo.GetByID(ctx, acc.ID)
	assert.Equal(t, 7.5, got.Balance)
	assert.Equal(t, pkgmodels.AccountStatusActive, got.Status)

	require.NoError(t, repo.Suspend(ctx, acc.ID))
	got, _ = repo.GetByID(ctx, acc.ID)
	assert.Equal(t, pkgmodels.AccountStatusSuspended, got.Status)

	require.NoError(t, repo.Close(ctx, acc.ID))
	got, _ = repo.GetByID(ctx, acc.ID)
	assert.Equal(t, pkgmodels.AccountStatusClosed, got.Status)

	// invalid status violates the CHECK constraint
	got.Status = "bogus"
	assert.Error(t, repo.Update(ctx, got))
}

func TestRC_AccountRepository_NotFoundAndInvalidIDs(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewAccountRepository(db)

	_, err := repo.GetByID(ctx, uuid.NewString())
	assert.ErrorIs(t, err, pkgmodels.ErrAccountNotFound)
	_, err = repo.GetByUserID(ctx, uuid.NewString())
	assert.ErrorIs(t, err, pkgmodels.ErrAccountNotFound)

	_, err = repo.GetByID(ctx, "bad")
	assert.ErrorIs(t, err, pkgmodels.ErrInvalidID)
	_, err = repo.GetByUserID(ctx, "bad")
	assert.ErrorIs(t, err, pkgmodels.ErrInvalidID)
	assert.ErrorIs(t, repo.Update(ctx, &pkgmodels.Account{ID: "bad"}), pkgmodels.ErrInvalidID)
	assert.ErrorIs(t, repo.UpdateBalance(ctx, "bad", 1), pkgmodels.ErrInvalidID)
	assert.ErrorIs(t, repo.Suspend(ctx, "bad"), pkgmodels.ErrInvalidID)
	assert.ErrorIs(t, repo.Activate(ctx, "bad"), pkgmodels.ErrInvalidID)
	assert.ErrorIs(t, repo.Close(ctx, "bad"), pkgmodels.ErrInvalidID)

	// FK violation: user does not exist
	err = repo.Create(ctx, &pkgmodels.Account{UserID: uuid.NewString(), Currency: "USD", Status: pkgmodels.AccountStatusActive})
	assert.Error(t, err)
}

func TestRC_TransactionRepository_BalanceFlow(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	accRepo := NewAccountRepository(db)
	repo := NewTransactionRepository(db)

	acc := rcCreateAccount(t, db, 0)
	refID := uuid.NewString()

	newTx := func(typ pkgmodels.TransactionType, amount float64, key string) *pkgmodels.Transaction {
		return &pkgmodels.Transaction{
			AccountID:      acc.ID,
			Type:           typ,
			Amount:         amount,
			Currency:       "USD",
			IdempotencyKey: key,
			ReferenceType:  "execution",
			ReferenceID:    refID,
			Metadata:       map[string]any{"k": key},
		}
	}

	dep := newTx(pkgmodels.TransactionTypeDeposit, 100, "dep-1")
	require.NoError(t, repo.Create(ctx, dep))
	assert.NotEmpty(t, dep.ID)
	assert.Equal(t, 0.0, dep.BalanceBefore)
	assert.Equal(t, 100.0, dep.BalanceAfter)
	assert.Equal(t, pkgmodels.TransactionStatusCompleted, dep.Status)

	charge := newTx(pkgmodels.TransactionTypeCharge, 30, "chg-1")
	require.NoError(t, repo.Create(ctx, charge))
	assert.Equal(t, 100.0, charge.BalanceBefore)
	assert.Equal(t, 70.0, charge.BalanceAfter)

	refund := newTx(pkgmodels.TransactionTypeRefund, 5, "ref-1")
	require.NoError(t, repo.Create(ctx, refund))
	adj := newTx(pkgmodels.TransactionTypeAdjustment, 1, "adj-1")
	require.NoError(t, repo.Create(ctx, adj))

	got, err := accRepo.GetByID(ctx, acc.ID)
	require.NoError(t, err)
	assert.Equal(t, 76.0, got.Balance)

	// insufficient balance: rolled back, balance unchanged
	err = repo.Create(ctx, newTx(pkgmodels.TransactionTypeCharge, 1000, "chg-2"))
	assert.ErrorIs(t, err, pkgmodels.ErrInsufficientBalance)
	// unknown type
	err = repo.Create(ctx, newTx("weird", 1, "weird-1"))
	assert.ErrorIs(t, err, pkgmodels.ErrInvalidInput)
	// duplicate idempotency key violates unique index and rolls back the balance change
	err = repo.Create(ctx, newTx(pkgmodels.TransactionTypeDeposit, 1, "dep-1"))
	assert.Error(t, err)
	// account errors
	err = repo.Create(ctx, &pkgmodels.Transaction{AccountID: "bad"})
	assert.ErrorIs(t, err, pkgmodels.ErrInvalidID)
	err = repo.Create(ctx, &pkgmodels.Transaction{AccountID: uuid.NewString(), Type: pkgmodels.TransactionTypeDeposit})
	assert.ErrorIs(t, err, pkgmodels.ErrAccountNotFound)

	got, _ = accRepo.GetByID(ctx, acc.ID)
	assert.Equal(t, 76.0, got.Balance, "failed transactions must not change the balance")

	// lookups
	byID, err := repo.GetByID(ctx, charge.ID)
	require.NoError(t, err)
	assert.Equal(t, pkgmodels.TransactionTypeCharge, byID.Type)
	assert.Equal(t, 30.0, byID.Amount)
	assert.Equal(t, refID, byID.ReferenceID)
	assert.Equal(t, "chg-1", byID.Metadata["k"])

	_, err = repo.GetByID(ctx, uuid.NewString())
	assert.ErrorIs(t, err, pkgmodels.ErrTransactionNotFound)
	_, err = repo.GetByID(ctx, "bad")
	assert.ErrorIs(t, err, pkgmodels.ErrInvalidID)

	byKey, err := repo.GetByIdempotencyKey(ctx, "ref-1")
	require.NoError(t, err)
	require.NotNil(t, byKey)
	assert.Equal(t, refund.ID, byKey.ID)
	none, err := repo.GetByIdempotencyKey(ctx, "missing")
	require.NoError(t, err)
	assert.Nil(t, none)

	all, err := repo.GetByAccountID(ctx, acc.ID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, all, 4)
	page, err := repo.GetByAccountID(ctx, acc.ID, 2, 3)
	require.NoError(t, err)
	assert.Len(t, page, 1)
	_, err = repo.GetByAccountID(ctx, "bad", 10, 0)
	assert.ErrorIs(t, err, pkgmodels.ErrInvalidID)

	deposits, err := repo.GetByAccountIDAndType(ctx, acc.ID, pkgmodels.TransactionTypeDeposit, 10, 0)
	require.NoError(t, err)
	require.Len(t, deposits, 1)
	assert.Equal(t, dep.ID, deposits[0].ID)
	_, err = repo.GetByAccountIDAndType(ctx, "bad", pkgmodels.TransactionTypeDeposit, 10, 0)
	assert.ErrorIs(t, err, pkgmodels.ErrInvalidID)

	byRef, err := repo.GetByReference(ctx, "execution", refID)
	require.NoError(t, err)
	assert.Len(t, byRef, 4)
	byRef, err = repo.GetByReference(ctx, "other", refID)
	require.NoError(t, err)
	assert.Empty(t, byRef)

	count, err := repo.CountByAccountID(ctx, acc.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(4), count)
	_, err = repo.CountByAccountID(ctx, "bad")
	assert.ErrorIs(t, err, pkgmodels.ErrInvalidID)
}

func TestRC_PricingPlanRepository_SeededPlans(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewPricingPlanRepository(db)

	all, err := repo.GetAll(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, all, "migrations seed pricing plans")

	active, err := repo.GetActive(ctx)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(active), len(all))
	for _, p := range active {
		assert.True(t, p.IsActive)
	}

	rental, err := repo.GetByResourceType(ctx, pkgmodels.ResourceTypeRentalKey)
	require.NoError(t, err)
	require.Len(t, rental, 3)
	for i := 1; i < len(rental); i++ {
		assert.LessOrEqual(t, rental[i-1].PricePerUnit, rental[i].PricePerUnit, "ordered by price")
	}

	free, err := repo.GetFreePlan(ctx, pkgmodels.ResourceTypeFileStorage)
	require.NoError(t, err)
	assert.True(t, free.IsFree)
	assert.Equal(t, pkgmodels.ResourceTypeFileStorage, free.ResourceType)

	byID, err := repo.GetByID(ctx, free.ID)
	require.NoError(t, err)
	assert.Equal(t, free.Name, byID.Name)

	_, err = repo.GetByID(ctx, uuid.NewString())
	assert.ErrorIs(t, err, pkgmodels.ErrPricingPlanNotFound)
	_, err = repo.GetByID(ctx, "bad")
	assert.ErrorIs(t, err, pkgmodels.ErrInvalidID)
	_, err = repo.GetFreePlan(ctx, "no-such-type")
	assert.ErrorIs(t, err, pkgmodels.ErrPricingPlanNotFound)

	none, err := repo.GetByResourceType(ctx, "no-such-type")
	require.NoError(t, err)
	assert.Empty(t, none)
}

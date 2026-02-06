package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	pkgmodels "github.com/smilemakc/mbflow/pkg/models"
)

var _ repository.AccountRepository = (*AccountRepositoryImpl)(nil)
var _ repository.TransactionRepository = (*TransactionRepositoryImpl)(nil)

type AccountRepositoryImpl struct {
	db bun.IDB
}

func NewAccountRepository(db bun.IDB) *AccountRepositoryImpl {
	return &AccountRepositoryImpl{db: db}
}

func (r *AccountRepositoryImpl) Create(ctx context.Context, account *pkgmodels.Account) error {
	accountModel := models.FromAccountDomain(account)

	_, err := r.db.NewInsert().Model(accountModel).Exec(ctx)
	if err != nil {
		return err
	}

	account.ID = accountModel.ID.String()
	account.CreatedAt = accountModel.CreatedAt
	account.UpdatedAt = accountModel.UpdatedAt

	return nil
}

func (r *AccountRepositoryImpl) GetByID(ctx context.Context, id string) (*pkgmodels.Account, error) {
	accountID, err := uuid.Parse(id)
	if err != nil {
		return nil, pkgmodels.ErrInvalidID
	}

	accountModel := new(models.BillingAccountModel)
	err = r.db.NewSelect().
		Model(accountModel).
		Where("id = ?", accountID).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, pkgmodels.ErrAccountNotFound
		}
		return nil, err
	}

	return models.ToAccountDomain(accountModel), nil
}

func (r *AccountRepositoryImpl) GetByUserID(ctx context.Context, userID string) (*pkgmodels.Account, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, pkgmodels.ErrInvalidID
	}

	accountModel := new(models.BillingAccountModel)
	err = r.db.NewSelect().
		Model(accountModel).
		Where("user_id = ?", userUUID).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, pkgmodels.ErrAccountNotFound
		}
		return nil, err
	}

	return models.ToAccountDomain(accountModel), nil
}

func (r *AccountRepositoryImpl) Update(ctx context.Context, account *pkgmodels.Account) error {
	accountID, err := uuid.Parse(account.ID)
	if err != nil {
		return pkgmodels.ErrInvalidID
	}

	_, err = r.db.NewUpdate().
		Model((*models.BillingAccountModel)(nil)).
		Set("balance = ?", account.Balance).
		Set("currency = ?", account.Currency).
		Set("status = ?", string(account.Status)).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", accountID).
		Exec(ctx)

	return err
}

func (r *AccountRepositoryImpl) UpdateBalance(ctx context.Context, id string, newBalance float64) error {
	accountID, err := uuid.Parse(id)
	if err != nil {
		return pkgmodels.ErrInvalidID
	}

	_, err = r.db.NewUpdate().
		Model((*models.BillingAccountModel)(nil)).
		Set("balance = ?", newBalance).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", accountID).
		Exec(ctx)

	return err
}

func (r *AccountRepositoryImpl) Suspend(ctx context.Context, id string) error {
	accountID, err := uuid.Parse(id)
	if err != nil {
		return pkgmodels.ErrInvalidID
	}

	_, err = r.db.NewUpdate().
		Model((*models.BillingAccountModel)(nil)).
		Set("status = ?", string(pkgmodels.AccountStatusSuspended)).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", accountID).
		Exec(ctx)

	return err
}

func (r *AccountRepositoryImpl) Activate(ctx context.Context, id string) error {
	accountID, err := uuid.Parse(id)
	if err != nil {
		return pkgmodels.ErrInvalidID
	}

	_, err = r.db.NewUpdate().
		Model((*models.BillingAccountModel)(nil)).
		Set("status = ?", string(pkgmodels.AccountStatusActive)).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", accountID).
		Exec(ctx)

	return err
}

func (r *AccountRepositoryImpl) Close(ctx context.Context, id string) error {
	accountID, err := uuid.Parse(id)
	if err != nil {
		return pkgmodels.ErrInvalidID
	}

	_, err = r.db.NewUpdate().
		Model((*models.BillingAccountModel)(nil)).
		Set("status = ?", string(pkgmodels.AccountStatusClosed)).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", accountID).
		Exec(ctx)

	return err
}

type TransactionRepositoryImpl struct {
	db bun.IDB
}

func NewTransactionRepository(db bun.IDB) *TransactionRepositoryImpl {
	return &TransactionRepositoryImpl{db: db}
}

func (r *TransactionRepositoryImpl) Create(ctx context.Context, tx *pkgmodels.Transaction) error {
	return r.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, dbTx bun.Tx) error {
		accountID, err := uuid.Parse(tx.AccountID)
		if err != nil {
			return pkgmodels.ErrInvalidID
		}

		accountModel := new(models.BillingAccountModel)
		err = dbTx.NewSelect().
			Model(accountModel).
			Where("id = ?", accountID).
			For("UPDATE").
			Scan(ctx)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return pkgmodels.ErrAccountNotFound
			}
			return err
		}

		tx.BalanceBefore = accountModel.Balance

		switch tx.Type {
		case pkgmodels.TransactionTypeDeposit:
			accountModel.Balance += tx.Amount
		case pkgmodels.TransactionTypeCharge:
			if accountModel.Balance < tx.Amount {
				return pkgmodels.ErrInsufficientBalance
			}
			accountModel.Balance -= tx.Amount
		case pkgmodels.TransactionTypeRefund:
			accountModel.Balance += tx.Amount
		case pkgmodels.TransactionTypeAdjustment:
			accountModel.Balance += tx.Amount
		default:
			return pkgmodels.ErrInvalidInput
		}

		tx.BalanceAfter = accountModel.Balance
		tx.Status = pkgmodels.TransactionStatusCompleted

		txModel := models.FromTransactionDomain(tx)

		_, err = dbTx.NewInsert().Model(txModel).Exec(ctx)
		if err != nil {
			return err
		}

		_, err = dbTx.NewUpdate().
			Model(accountModel).
			Column("balance", "updated_at").
			Where("id = ?", accountID).
			Exec(ctx)

		if err != nil {
			return err
		}

		tx.ID = txModel.ID.String()
		tx.CreatedAt = txModel.CreatedAt

		return nil
	})
}

func (r *TransactionRepositoryImpl) GetByID(ctx context.Context, id string) (*pkgmodels.Transaction, error) {
	txID, err := uuid.Parse(id)
	if err != nil {
		return nil, pkgmodels.ErrInvalidID
	}

	txModel := new(models.TransactionModel)
	err = r.db.NewSelect().
		Model(txModel).
		Where("id = ?", txID).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, pkgmodels.ErrTransactionNotFound
		}
		return nil, err
	}

	return models.ToTransactionDomain(txModel), nil
}

func (r *TransactionRepositoryImpl) GetByIdempotencyKey(ctx context.Context, key string) (*pkgmodels.Transaction, error) {
	txModel := new(models.TransactionModel)
	err := r.db.NewSelect().
		Model(txModel).
		Where("idempotency_key = ?", key).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return models.ToTransactionDomain(txModel), nil
}

func (r *TransactionRepositoryImpl) GetByAccountID(ctx context.Context, accountID string, limit, offset int) ([]*pkgmodels.Transaction, error) {
	accID, err := uuid.Parse(accountID)
	if err != nil {
		return nil, pkgmodels.ErrInvalidID
	}

	var txModels []*models.TransactionModel
	err = r.db.NewSelect().
		Model(&txModels).
		Where("account_id = ?", accID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	transactions := make([]*pkgmodels.Transaction, len(txModels))
	for i, tm := range txModels {
		transactions[i] = models.ToTransactionDomain(tm)
	}

	return transactions, nil
}

func (r *TransactionRepositoryImpl) GetByAccountIDAndType(ctx context.Context, accountID string, txType pkgmodels.TransactionType, limit, offset int) ([]*pkgmodels.Transaction, error) {
	accID, err := uuid.Parse(accountID)
	if err != nil {
		return nil, pkgmodels.ErrInvalidID
	}

	var txModels []*models.TransactionModel
	err = r.db.NewSelect().
		Model(&txModels).
		Where("account_id = ? AND type = ?", accID, string(txType)).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	transactions := make([]*pkgmodels.Transaction, len(txModels))
	for i, tm := range txModels {
		transactions[i] = models.ToTransactionDomain(tm)
	}

	return transactions, nil
}

func (r *TransactionRepositoryImpl) GetByReference(ctx context.Context, referenceType string, referenceID string) ([]*pkgmodels.Transaction, error) {
	var txModels []*models.TransactionModel
	err := r.db.NewSelect().
		Model(&txModels).
		Where("reference_type = ? AND reference_id = ?", referenceType, referenceID).
		Order("created_at DESC").
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	transactions := make([]*pkgmodels.Transaction, len(txModels))
	for i, tm := range txModels {
		transactions[i] = models.ToTransactionDomain(tm)
	}

	return transactions, nil
}

func (r *TransactionRepositoryImpl) CountByAccountID(ctx context.Context, accountID string) (int64, error) {
	accID, err := uuid.Parse(accountID)
	if err != nil {
		return 0, pkgmodels.ErrInvalidID
	}

	count, err := r.db.NewSelect().
		Model((*models.TransactionModel)(nil)).
		Where("account_id = ?", accID).
		Count(ctx)

	if err != nil {
		return 0, err
	}

	return int64(count), nil
}

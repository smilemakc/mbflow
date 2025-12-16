package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/pkg/models"
)

// AccountHandlers handles billing account operations
type AccountHandlers struct {
	accountRepo     repository.AccountRepository
	transactionRepo repository.TransactionRepository
	logger          *logger.Logger
}

// NewAccountHandlers creates a new AccountHandlers instance
func NewAccountHandlers(accountRepo repository.AccountRepository, transactionRepo repository.TransactionRepository, log *logger.Logger) *AccountHandlers {
	return &AccountHandlers{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
		logger:          log,
	}
}

// GetAccount returns current user's billing account
// GET /api/v1/account
func (h *AccountHandlers) GetAccount(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	account, err := h.accountRepo.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		if err == models.ErrAccountNotFound {
			respondError(c, http.StatusNotFound, "account not found")
			return
		}
		h.logger.Error("Failed to get account", "error", err, "user_id", userID)
		respondError(c, http.StatusInternalServerError, "failed to get account")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         account.ID,
		"user_id":    account.UserID,
		"balance":    account.Balance,
		"currency":   account.Currency,
		"status":     account.Status,
		"created_at": account.CreatedAt,
		"updated_at": account.UpdatedAt,
	})
}

// DepositRequest represents a deposit request
type DepositRequest struct {
	Amount         float64 `json:"amount" binding:"required,gt=0"`
	IdempotencyKey string  `json:"idempotency_key" binding:"required"`
	Description    string  `json:"description" binding:"max=500"`
}

// Deposit adds funds to account
// POST /api/v1/account/deposit
func (h *AccountHandlers) Deposit(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req DepositRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	account, err := h.accountRepo.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		if err == models.ErrAccountNotFound {
			respondError(c, http.StatusNotFound, "account not found")
			return
		}
		h.logger.Error("Failed to get account for deposit", "error", err, "user_id", userID)
		respondError(c, http.StatusInternalServerError, "failed to get account")
		return
	}

	if !account.IsActive() {
		respondError(c, http.StatusForbidden, "account is not active")
		return
	}

	existingTx, err := h.transactionRepo.GetByIdempotencyKey(c.Request.Context(), req.IdempotencyKey)
	if err == nil && existingTx != nil {
		h.logger.Info("Duplicate deposit detected, returning existing transaction",
			"idempotency_key", req.IdempotencyKey,
			"transaction_id", existingTx.ID,
		)
		c.JSON(http.StatusOK, gin.H{
			"id":                existingTx.ID,
			"account_id":        existingTx.AccountID,
			"type":              existingTx.Type,
			"amount":            existingTx.Amount,
			"currency":          existingTx.Currency,
			"status":            existingTx.Status,
			"description":       existingTx.Description,
			"idempotency_key":   existingTx.IdempotencyKey,
			"balance_before":    existingTx.BalanceBefore,
			"balance_after":     existingTx.BalanceAfter,
			"created_at":        existingTx.CreatedAt,
			"duplicate_request": true,
		})
		return
	}

	// Create transaction - this atomically updates balance and creates transaction record
	transaction := &models.Transaction{
		AccountID:      account.ID,
		Type:           models.TransactionTypeDeposit,
		Amount:         req.Amount,
		Currency:       account.Currency,
		Description:    req.Description,
		IdempotencyKey: req.IdempotencyKey,
	}

	if err := h.transactionRepo.Create(c.Request.Context(), transaction); err != nil {
		h.logger.Error("Failed to create deposit transaction", "error", err, "account_id", account.ID)
		respondError(c, http.StatusInternalServerError, "failed to process deposit")
		return
	}

	h.logger.Info("Deposit successful",
		"account_id", account.ID,
		"user_id", userID,
		"amount", req.Amount,
		"transaction_id", transaction.ID,
		"balance_after", transaction.BalanceAfter,
	)

	c.JSON(http.StatusOK, gin.H{
		"id":              transaction.ID,
		"account_id":      transaction.AccountID,
		"type":            transaction.Type,
		"amount":          transaction.Amount,
		"currency":        transaction.Currency,
		"status":          transaction.Status,
		"description":     transaction.Description,
		"idempotency_key": transaction.IdempotencyKey,
		"balance_before":  transaction.BalanceBefore,
		"balance_after":   transaction.BalanceAfter,
		"created_at":      transaction.CreatedAt,
	})
}

// ListTransactions returns transaction history with pagination
// GET /api/v1/account/transactions?limit=20&offset=0&type=deposit
func (h *AccountHandlers) ListTransactions(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	account, err := h.accountRepo.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		if err == models.ErrAccountNotFound {
			respondError(c, http.StatusNotFound, "account not found")
			return
		}
		h.logger.Error("Failed to get account for transactions", "error", err, "user_id", userID)
		respondError(c, http.StatusInternalServerError, "failed to get account")
		return
	}

	limit := getQueryInt(c, "limit", 20)
	offset := getQueryInt(c, "offset", 0)
	txType := c.Query("type")

	var transactions []*models.Transaction
	if txType != "" {
		transactions, err = h.transactionRepo.GetByAccountIDAndType(c.Request.Context(), account.ID, models.TransactionType(txType), limit, offset)
	} else {
		transactions, err = h.transactionRepo.GetByAccountID(c.Request.Context(), account.ID, limit, offset)
	}

	if err != nil {
		h.logger.Error("Failed to get transactions", "error", err, "account_id", account.ID)
		respondError(c, http.StatusInternalServerError, "failed to get transactions")
		return
	}

	total, err := h.transactionRepo.CountByAccountID(c.Request.Context(), account.ID)
	if err != nil {
		h.logger.Warn("Failed to count transactions", "error", err, "account_id", account.ID)
		total = int64(len(transactions))
	}

	response := make([]gin.H, len(transactions))
	for i, tx := range transactions {
		response[i] = gin.H{
			"id":              tx.ID,
			"account_id":      tx.AccountID,
			"type":            tx.Type,
			"amount":          tx.Amount,
			"currency":        tx.Currency,
			"status":          tx.Status,
			"description":     tx.Description,
			"reference_type":  tx.ReferenceType,
			"reference_id":    tx.ReferenceID,
			"idempotency_key": tx.IdempotencyKey,
			"balance_before":  tx.BalanceBefore,
			"balance_after":   tx.BalanceAfter,
			"metadata":        tx.Metadata,
			"created_at":      tx.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": response,
		"total":        total,
		"limit":        limit,
		"offset":       offset,
	})
}

// GetTransaction returns a specific transaction by ID
// GET /api/v1/account/transactions/:id
func (h *AccountHandlers) GetTransaction(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	transactionID, ok := getParam(c, "id")
	if !ok {
		return
	}

	account, err := h.accountRepo.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		if err == models.ErrAccountNotFound {
			respondError(c, http.StatusNotFound, "account not found")
			return
		}
		h.logger.Error("Failed to get account", "error", err, "user_id", userID)
		respondError(c, http.StatusInternalServerError, "failed to get account")
		return
	}

	transaction, err := h.transactionRepo.GetByID(c.Request.Context(), transactionID)
	if err != nil {
		if err == models.ErrTransactionNotFound {
			respondError(c, http.StatusNotFound, "transaction not found")
			return
		}
		h.logger.Error("Failed to get transaction", "error", err, "transaction_id", transactionID)
		respondError(c, http.StatusInternalServerError, "failed to get transaction")
		return
	}

	if transaction.AccountID != account.ID {
		respondError(c, http.StatusForbidden, "access denied")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":              transaction.ID,
		"account_id":      transaction.AccountID,
		"type":            transaction.Type,
		"amount":          transaction.Amount,
		"currency":        transaction.Currency,
		"status":          transaction.Status,
		"description":     transaction.Description,
		"reference_type":  transaction.ReferenceType,
		"reference_id":    transaction.ReferenceID,
		"idempotency_key": transaction.IdempotencyKey,
		"balance_before":  transaction.BalanceBefore,
		"balance_after":   transaction.BalanceAfter,
		"metadata":        transaction.Metadata,
		"created_at":      transaction.CreatedAt,
	})
}

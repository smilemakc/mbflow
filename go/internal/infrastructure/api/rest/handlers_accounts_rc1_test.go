package rest

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
)

// rc1TxRepoCountFails wraps a transaction repository whose count query fails.
type rc1TxRepoCountFails struct {
	repository.TransactionRepository
}

func (rc1TxRepoCountFails) CountByAccountID(context.Context, string) (int64, error) {
	return 0, errors.New("count failed")
}

func rc1AccountRouter(accounts repository.AccountRepository, txs repository.TransactionRepository) *gin.Engine {
	h := NewAccountHandlers(accounts, txs, rc1Logger())
	r := gin.New()
	g := r.Group("/api/v1/account", rc1UserFromHeader())
	g.GET("", h.GetAccount)
	g.POST("/deposit", h.Deposit)
	g.GET("/transactions", h.ListTransactions)
	g.GET("/transactions/:id", h.GetTransaction)
	g.GET("/tx-noparam", h.GetTransaction)
	return r
}

func TestRC1Accounts_DepositAndTransactions(t *testing.T) {
	t.Parallel()
	db := rc1DB(t)
	accounts := storage.NewAccountRepository(db)
	txs := storage.NewTransactionRepository(db)
	r := rc1AccountRouter(accounts, txs)
	alice := rc1CreateUser(t, db, "alice")
	bob := rc1CreateUser(t, db, "bob")
	asAlice := map[string]string{"X-Test-User": alice.String()}
	asBob := map[string]string{"X-Test-User": bob.String()}

	// Unauthenticated access is rejected everywhere.
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/account"},
		{http.MethodPost, "/api/v1/account/deposit"},
		{http.MethodGet, "/api/v1/account/transactions"},
		{http.MethodGet, "/api/v1/account/transactions/" + uuid.NewString()},
	} {
		w := rc1Do(t, r, tc.method, tc.path, nil, nil)
		assert.Equal(t, http.StatusUnauthorized, w.Code, tc.path)
	}

	// The account is created lazily on first access.
	w := rc1Do(t, r, http.MethodGet, "/api/v1/account", nil, asAlice)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	acc := rc1JSON(t, w)
	assert.Equal(t, alice.String(), acc["user_id"])
	assert.EqualValues(t, 0, acc["balance"])
	assert.Equal(t, "active", acc["status"])

	// Deposit validation.
	w = rc1Do(t, r, http.MethodPost, "/api/v1/account/deposit", map[string]any{"amount": -1, "idempotency_key": "k"}, asAlice)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = rc1Do(t, r, http.MethodPost, "/api/v1/account/deposit", map[string]any{"amount": 5}, asAlice)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = rc1Do(t, r, http.MethodPost, "/api/v1/account/deposit", map[string]any{"amount": 10.5, "idempotency_key": "alice-1", "description": "top up"}, asAlice)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	dep := rc1JSON(t, w)
	assert.EqualValues(t, 0, dep["balance_before"])
	assert.EqualValues(t, 10.5, dep["balance_after"])
	assert.Equal(t, "deposit", dep["type"])
	txID := dep["id"].(string)

	// Replaying the same key returns the original transaction without charging twice.
	w = rc1Do(t, r, http.MethodPost, "/api/v1/account/deposit", map[string]any{"amount": 10.5, "idempotency_key": "alice-1"}, asAlice)
	require.Equal(t, http.StatusOK, w.Code)
	dup := rc1JSON(t, w)
	assert.Equal(t, true, dup["duplicate_request"])
	assert.Equal(t, txID, dup["id"])
	w = rc1Do(t, r, http.MethodGet, "/api/v1/account", nil, asAlice)
	assert.EqualValues(t, 10.5, rc1JSON(t, w)["balance"])

	// Another user's idempotency key must not leak their transaction.
	w = rc1Do(t, r, http.MethodPost, "/api/v1/account/deposit", map[string]any{"amount": 1, "idempotency_key": "alice-1"}, asBob)
	assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	assert.NotContains(t, w.Body.String(), txID)

	w = rc1Do(t, r, http.MethodPost, "/api/v1/account/deposit", map[string]any{"amount": 2, "idempotency_key": "bob-1"}, asBob)
	require.Equal(t, http.StatusOK, w.Code)
	bobTxID := rc1JSON(t, w)["id"].(string)

	// Listing.
	w = rc1Do(t, r, http.MethodGet, "/api/v1/account/transactions?limit=5", nil, asAlice)
	require.Equal(t, http.StatusOK, w.Code)
	list := rc1JSON(t, w)
	assert.EqualValues(t, 1, list["total"])
	assert.EqualValues(t, 5, list["limit"])
	require.Len(t, list["transactions"], 1)
	assert.Equal(t, txID, list["transactions"].([]any)[0].(map[string]any)["id"])
	w = rc1Do(t, r, http.MethodGet, "/api/v1/account/transactions?type=charge", nil, asAlice)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, rc1JSON(t, w)["transactions"], 0)
	w = rc1Do(t, r, http.MethodGet, "/api/v1/account/transactions?type=deposit", nil, asAlice)
	assert.Len(t, rc1JSON(t, w)["transactions"], 1)

	// Get single transaction: ownership enforced.
	w = rc1Do(t, r, http.MethodGet, "/api/v1/account/transactions/"+txID, nil, asAlice)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "alice-1", rc1JSON(t, w)["idempotency_key"])
	w = rc1Do(t, r, http.MethodGet, "/api/v1/account/transactions/"+bobTxID, nil, asAlice)
	assert.Equal(t, http.StatusForbidden, w.Code)
	w = rc1Do(t, r, http.MethodGet, "/api/v1/account/transactions/"+uuid.NewString(), nil, asAlice)
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = rc1Do(t, r, http.MethodGet, "/api/v1/account/transactions/not-a-uuid", nil, asAlice)
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = rc1Do(t, r, http.MethodGet, "/api/v1/account/tx-noparam", nil, asAlice)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Suspended accounts cannot deposit.
	w = rc1Do(t, r, http.MethodGet, "/api/v1/account", nil, asBob)
	require.NoError(t, accounts.Suspend(context.Background(), rc1JSON(t, w)["id"].(string)))
	w = rc1Do(t, r, http.MethodPost, "/api/v1/account/deposit", map[string]any{"amount": 2, "idempotency_key": "bob-2"}, asBob)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// A failing count falls back to the page length.
	r2 := rc1AccountRouter(accounts, rc1TxRepoCountFails{txs})
	w = rc1Do(t, r2, http.MethodGet, "/api/v1/account/transactions", nil, asAlice)
	require.Equal(t, http.StatusOK, w.Code)
	assert.EqualValues(t, 1, rc1JSON(t, w)["total"])
}

func TestRC1Accounts_UnknownUserHasNoAccount(t *testing.T) {
	t.Parallel()
	db := rc1DB(t)
	r := rc1AccountRouter(storage.NewAccountRepository(db), storage.NewTransactionRepository(db))
	ghost := map[string]string{"X-Test-User": uuid.NewString()}

	for _, tc := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, "/api/v1/account", nil},
		{http.MethodPost, "/api/v1/account/deposit", map[string]any{"amount": 1, "idempotency_key": "g"}},
		{http.MethodGet, "/api/v1/account/transactions", nil},
		{http.MethodGet, "/api/v1/account/transactions/" + uuid.NewString(), nil},
	} {
		w := rc1Do(t, r, tc.method, tc.path, tc.body, ghost)
		assert.Equal(t, http.StatusNotFound, w.Code, "%s %s: %s", tc.method, tc.path, w.Body.String())
		assert.Equal(t, "account not found", rc1JSON(t, w)["message"])
	}
}

func TestRC1Accounts_StorageFailures(t *testing.T) {
	t.Parallel()
	closed := rc1ClosedDB(t)
	r := rc1AccountRouter(storage.NewAccountRepository(closed), storage.NewTransactionRepository(closed))
	u := map[string]string{"X-Test-User": uuid.NewString()}
	for _, tc := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, "/api/v1/account", nil},
		{http.MethodPost, "/api/v1/account/deposit", map[string]any{"amount": 1, "idempotency_key": "x"}},
		{http.MethodGet, "/api/v1/account/transactions", nil},
		{http.MethodGet, "/api/v1/account/transactions/" + uuid.NewString(), nil},
	} {
		w := rc1Do(t, r, tc.method, tc.path, tc.body, u)
		assert.Equal(t, http.StatusInternalServerError, w.Code, "%s %s", tc.method, tc.path)
	}

	// Live accounts, broken transaction store.
	db := rc1DB(t)
	user := rc1CreateUser(t, db, "u")
	r = rc1AccountRouter(storage.NewAccountRepository(db), storage.NewTransactionRepository(closed))
	asUser := map[string]string{"X-Test-User": user.String()}
	for _, tc := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPost, "/api/v1/account/deposit", map[string]any{"amount": 1, "idempotency_key": "x"}},
		{http.MethodGet, "/api/v1/account/transactions", nil},
		{http.MethodGet, "/api/v1/account/transactions?type=deposit", nil},
		{http.MethodGet, "/api/v1/account/transactions/" + uuid.NewString(), nil},
	} {
		w := rc1Do(t, r, tc.method, tc.path, tc.body, asUser)
		assert.Equal(t, http.StatusInternalServerError, w.Code, "%s %s", tc.method, tc.path)
	}
}

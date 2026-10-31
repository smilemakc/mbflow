package storage

import (
	"context"
	"errors"
	"testing"
	"testing/fstest"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/migrations"
	"github.com/smilemakc/mbflow/go/pkg/crypto"
	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/smilemakc/mbflow/go/testutil"
)

func TestRC_DefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, 20, cfg.MaxOpenConns)
	assert.Equal(t, 5, cfg.MaxIdleConns)
	assert.Equal(t, time.Hour, cfg.ConnMaxLifetime)
	assert.Equal(t, 10*time.Minute, cfg.ConnMaxIdleTime)
	assert.False(t, cfg.Debug)
	assert.Empty(t, cfg.DSN)
}

func TestRC_NewDB_PingFailure(t *testing.T) {
	t.Parallel()
	// Port 1 on loopback is never a PostgreSQL server: the connection is refused immediately.
	db, err := NewDB(&Config{DSN: "postgres://u:p@127.0.0.1:1/db?sslmode=disable&dial_timeout=1s"})
	assert.Error(t, err)
	assert.Nil(t, db)
	assert.Contains(t, err.Error(), "failed to ping database")
}

func TestRC_DBHelpers(t *testing.T) {
	t.Parallel()
	db, err := NewDB(&Config{DSN: testutil.SetupTestDSN(t), MaxOpenConns: 2, MaxIdleConns: 1, Debug: true})
	require.NoError(t, err)
	t.Cleanup(func() { _ = Close(db) })
	ctx := context.Background()

	require.NoError(t, Ping(ctx, db))
	assert.Equal(t, 2, Stats(db).MaxOpenConnections)
	assert.NoError(t, Close(nil))

	// committed transaction
	wfID := uuid.New()
	err = WithTransaction(ctx, db, func(tx bun.Tx) error {
		_, err := tx.NewInsert().Model(&models.WorkflowModel{ID: wfID, Name: "tx", Status: "draft", Version: 1}).Exec(ctx)
		return err
	})
	require.NoError(t, err)
	exists, err := db.NewSelect().Model((*models.WorkflowModel)(nil)).Where("id = ?", wfID).Exists(ctx)
	require.NoError(t, err)
	assert.True(t, exists)

	// rolled back transaction
	boom := errors.New("boom")
	wfID2 := uuid.New()
	err = WithTransaction(ctx, db, func(tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(&models.WorkflowModel{ID: wfID2, Name: "tx2", Status: "draft", Version: 1}).Exec(ctx); err != nil {
			return err
		}
		return boom
	})
	assert.ErrorIs(t, err, boom)
	exists, err = db.NewSelect().Model((*models.WorkflowModel)(nil)).Where("id = ?", wfID2).Exists(ctx)
	require.NoError(t, err)
	assert.False(t, exists, "transaction must be rolled back")
}

func TestRC_Migrator_StatusResetAndAccessWrapper(t *testing.T) {
	t.Parallel()
	db, err := NewDB(&Config{DSN: testutil.SetupTestDSN(t), MaxOpenConns: 2, MaxIdleConns: 1})
	require.NoError(t, err)
	t.Cleanup(func() { _ = Close(db) })
	ctx := context.Background()

	m, err := NewMigratorWithAccess(db, migrations.FS)
	require.NoError(t, err)
	require.NoError(t, m.CreateMigrationTable(ctx))
	require.NoError(t, m.Status(ctx))

	ms, err := m.MigrationsWithStatus(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, ms)
	for _, mig := range ms {
		assert.Positive(t, mig.GroupID, "template DB has every migration applied: %s", mig.Name)
	}

	// nothing pending
	group, err := m.Migrate(ctx)
	require.NoError(t, err)
	assert.True(t, group.IsZero())
	require.NoError(t, m.Up(ctx))

	// rollback the single group, then nothing more to roll back
	group, err = m.Rollback(ctx)
	require.NoError(t, err)
	assert.False(t, group.IsZero())
	require.NoError(t, m.Down(ctx), "down with nothing applied is a no-op")

	ms, err = m.MigrationsWithStatus(ctx)
	require.NoError(t, err)
	for _, mig := range ms {
		assert.Zero(t, mig.GroupID, "migration %s should be pending", mig.Name)
	}

	group, err = m.Migrate(ctx)
	require.NoError(t, err)
	assert.False(t, group.IsZero())

	// Reset rolls everything back
	require.NoError(t, m.Reset(ctx))
	var tableExists bool
	require.NoError(t, db.NewRaw("SELECT to_regclass('mbflow_workflows') IS NOT NULL").Scan(ctx, &tableExists))
	assert.False(t, tableExists)
	require.NoError(t, m.Up(ctx))
	require.NoError(t, db.NewRaw("SELECT to_regclass('mbflow_workflows') IS NOT NULL").Scan(ctx, &tableExists))
	assert.True(t, tableExists)
}

func TestRC_Migrator_DiscoverFailure(t *testing.T) {
	t.Parallel()
	badFS := fstest.MapFS{"bad-name.up.sql": &fstest.MapFile{Data: []byte("SELECT 1")}}
	_, err := NewMigrator(nil, badFS)
	assert.Error(t, err)
	_, err = NewMigratorWithAccess(nil, badFS)
	assert.Error(t, err)
}

// rcClosedDB returns a *bun.DB whose connection pool has been closed, so every
// query fails. It is used to verify that repositories propagate driver errors.
func rcClosedDB(t *testing.T) *bun.DB {
	t.Helper()
	db, err := NewDB(&Config{DSN: testutil.SetupTestDSN(t), MaxOpenConns: 1, MaxIdleConns: 1})
	require.NoError(t, err)
	require.NoError(t, db.Close())
	return db
}

func TestRC_Migrator_ClosedDB(t *testing.T) {
	t.Parallel()
	db := rcClosedDB(t)
	ctx := context.Background()
	m, err := NewMigratorWithAccess(db, migrations.FS)
	require.NoError(t, err)

	assert.Error(t, m.Up(ctx))
	assert.Error(t, m.Down(ctx))
	assert.Error(t, m.Status(ctx))
	assert.Error(t, m.Reset(ctx))
	_, err = m.Migrate(ctx)
	assert.Error(t, err)
	_, err = m.Rollback(ctx)
	assert.Error(t, err)
	_, err = m.MigrationsWithStatus(ctx)
	assert.Error(t, err)
	assert.Error(t, Ping(ctx, db))
}

// TestRC_Repositories_PropagateDBErrors calls every repository method against a
// closed connection pool and checks that the failure is reported, never
// swallowed or turned into a "not found" result.
func TestRC_Repositories_PropagateDBErrors(t *testing.T) {
	t.Parallel()
	db := rcClosedDB(t)
	ctx := context.Background()
	id := uuid.New()
	sid := id.String()
	now := time.Now()

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	enc, err := crypto.NewEncryptionService(key)
	require.NoError(t, err)

	acc := NewAccountRepository(db)
	txr := NewTransactionRepository(db)
	plans := NewPricingPlanRepository(db)
	res := NewResourceRepository(db)
	rk := NewRentalKeyRepository(db, enc)
	cred := NewCredentialsRepository(db)
	sk := NewServiceKeyRepository(db)
	syk := NewSystemKeyRepo(db)
	audit := NewServiceAuditLogRepo(db)
	users := NewUserRepository(db)
	wf := NewWorkflowRepository(db)
	ex := NewExecutionRepository(db)
	ev := NewEventRepository(db)
	tr := NewTriggerRepository(db).(*TriggerRepository)
	files := NewFileRepository(db)
	cfgs := NewStorageConfigRepository(db)
	api := NewAPIClientRepository(db)

	errOnly := func(_ any, err error) error { return err }
	errOnly3 := func(_ any, _ any, err error) error { return err }
	lock := now.Format(time.RFC3339)
	fs := rcNewFileStorage(sid, "x")
	fs.ID = sid
	credRes := rcNewCredentials(sid, "x", "p")
	credRes.ID = sid
	rental := rcNewRentalKey(sid, "x", "openai")
	rental.ID = sid

	calls := map[string]func() error{
		// accounts
		"Account.Create":       func() error { return acc.Create(ctx, &pkgmodels.Account{UserID: sid}) },
		"Account.GetByID":      func() error { return errOnly(acc.GetByID(ctx, sid)) },
		"Account.GetByUserID":  func() error { return errOnly(acc.GetByUserID(ctx, sid)) },
		"Account.Update":       func() error { return acc.Update(ctx, &pkgmodels.Account{ID: sid}) },
		"Account.UpdateBal":    func() error { return acc.UpdateBalance(ctx, sid, 1) },
		"Account.Suspend":      func() error { return acc.Suspend(ctx, sid) },
		"Account.Activate":     func() error { return acc.Activate(ctx, sid) },
		"Account.Close":        func() error { return acc.Close(ctx, sid) },
		"Tx.Create":            func() error { return txr.Create(ctx, &pkgmodels.Transaction{AccountID: sid}) },
		"Tx.GetByID":           func() error { return errOnly(txr.GetByID(ctx, sid)) },
		"Tx.GetByIdempotency":  func() error { return errOnly(txr.GetByIdempotencyKey(ctx, "k")) },
		"Tx.GetByAccountID":    func() error { return errOnly(txr.GetByAccountID(ctx, sid, 1, 0)) },
		"Tx.GetByAccountType":  func() error { return errOnly(txr.GetByAccountIDAndType(ctx, sid, "deposit", 1, 0)) },
		"Tx.GetByReference":    func() error { return errOnly(txr.GetByReference(ctx, "t", sid)) },
		"Tx.CountByAccountID":  func() error { return errOnly(txr.CountByAccountID(ctx, sid)) },
		"Plan.GetByID":         func() error { return errOnly(plans.GetByID(ctx, sid)) },
		"Plan.GetByType":       func() error { return errOnly(plans.GetByResourceType(ctx, "file_storage")) },
		"Plan.GetFree":         func() error { return errOnly(plans.GetFreePlan(ctx, "file_storage")) },
		"Plan.GetAll":          func() error { return errOnly(plans.GetAll(ctx)) },
		"Plan.GetActive":       func() error { return errOnly(plans.GetActive(ctx)) },
		"Resource.Create":      func() error { return res.Create(ctx, rcNewFileStorage(sid, "x")) },
		"Resource.GetByID":     func() error { return errOnly(res.GetByID(ctx, sid)) },
		"Resource.GetByOwner":  func() error { return errOnly(res.GetByOwner(ctx, sid)) },
		"Resource.GetByType":   func() error { return errOnly(res.GetByOwnerAndType(ctx, sid, "file_storage")) },
		"Resource.Update":      func() error { return res.Update(ctx, fs) },
		"Resource.Delete":      func() error { return res.Delete(ctx, sid) },
		"Resource.HardDelete":  func() error { return res.HardDelete(ctx, sid) },
		"Resource.UpdateUsage": func() error { return res.UpdateUsage(ctx, sid, 1, 1) },
		"Resource.IncUsage":    func() error { return res.IncrementUsage(ctx, sid, 1) },
		"Resource.DecUsage":    func() error { return res.DecrementUsage(ctx, sid, 1) },
		// rental keys
		"Rental.Create":         func() error { return rk.CreateRentalKey(ctx, rcNewRentalKey(sid, "x", "openai"), "k") },
		"Rental.Get":            func() error { return errOnly(rk.GetRentalKey(ctx, sid)) },
		"Rental.ByOwner":        func() error { return errOnly(rk.GetRentalKeysByOwner(ctx, sid)) },
		"Rental.ByProvider":     func() error { return errOnly(rk.GetRentalKeysByProvider(ctx, sid, "openai")) },
		"Rental.Update":         func() error { return rk.UpdateRentalKey(ctx, rental) },
		"Rental.Delete":         func() error { return rk.DeleteRentalKey(ctx, sid) },
		"Rental.Decrypt":        func() error { return errOnly(rk.GetDecryptedAPIKey(ctx, sid)) },
		"Rental.Rotate":         func() error { return rk.RotateAPIKey(ctx, sid, "k") },
		"Rental.RecordUsage":    func() error { return rk.RecordUsage(ctx, sid, &pkgmodels.RentalKeyUsageRecord{}) },
		"Rental.History":        func() error { return errOnly(rk.GetUsageHistory(ctx, sid, 1, 1)) },
		"Rental.HistoryByRange": func() error { return errOnly(rk.GetUsageHistoryByTimeRange(ctx, sid, "", "")) },
		"Rental.Summary": func() error {
			_, _, _, err := rk.GetUsageSummary(ctx, sid)
			return err
		},
		"Rental.ResetDaily":   func() error { return rk.ResetDailyUsage(ctx) },
		"Rental.ResetMonthly": func() error { return rk.ResetMonthlyUsage(ctx) },
		"Rental.All":          func() error { return errOnly3(rk.GetAllRentalKeys(ctx, repository.RentalKeyFilter{})) },
		"Rental.AllCount":     func() error { return errOnly(rk.GetAllRentalKeysCount(ctx, repository.RentalKeyFilter{})) },
		// credentials
		"Cred.Create":       func() error { return cred.CreateCredentials(ctx, rcNewCredentials(sid, "x", "p")) },
		"Cred.Get":          func() error { return errOnly(cred.GetCredentials(ctx, sid)) },
		"Cred.ByOwner":      func() error { return errOnly(cred.GetCredentialsByOwner(ctx, sid)) },
		"Cred.ByProvider":   func() error { return errOnly(cred.GetCredentialsByProvider(ctx, sid, "p")) },
		"Cred.Update":       func() error { return cred.UpdateCredentials(ctx, credRes) },
		"Cred.UpdateData":   func() error { return cred.UpdateEncryptedData(ctx, sid, nil) },
		"Cred.Delete":       func() error { return cred.DeleteCredentials(ctx, sid) },
		"Cred.IncUsage":     func() error { return cred.IncrementUsageCount(ctx, sid) },
		"Cred.LogAccess":    func() error { return cred.LogCredentialAccess(ctx, sid, "a", sid, "user", nil) },
		"SK.Create":         func() error { return sk.Create(ctx, &pkgmodels.ServiceKey{UserID: sid, CreatedBy: sid}) },
		"SK.FindByID":       func() error { return errOnly(sk.FindByID(ctx, id)) },
		"SK.FindByPrefix":   func() error { return errOnly(sk.FindByPrefix(ctx, "p")) },
		"SK.FindByUserID":   func() error { return errOnly(sk.FindByUserID(ctx, id)) },
		"SK.FindAll":        func() error { return errOnly3(sk.FindAll(ctx, repository.ServiceKeyFilter{})) },
		"SK.Update":         func() error { return sk.Update(ctx, &pkgmodels.ServiceKey{ID: sid, UserID: sid, CreatedBy: sid}) },
		"SK.Delete":         func() error { return sk.Delete(ctx, id) },
		"SK.Revoke":         func() error { return sk.Revoke(ctx, id) },
		"SK.UpdateLastUsed": func() error { return sk.UpdateLastUsed(ctx, id) },
		"SK.CountByUserID":  func() error { return errOnly(sk.CountByUserID(ctx, id)) },
		"SYK.Create":        func() error { return syk.Create(ctx, &pkgmodels.SystemKey{CreatedBy: sid}) },
		"SYK.FindByID":      func() error { return errOnly(syk.FindByID(ctx, id)) },
		"SYK.FindByPrefix":  func() error { return errOnly(syk.FindByPrefix(ctx, "p")) },
		"SYK.FindAll":       func() error { return errOnly3(syk.FindAll(ctx, repository.SystemKeyFilter{})) },
		"SYK.Update":        func() error { return syk.Update(ctx, &pkgmodels.SystemKey{ID: sid, CreatedBy: sid}) },
		"SYK.Delete":        func() error { return syk.Delete(ctx, id) },
		"SYK.Revoke":        func() error { return syk.Revoke(ctx, id) },
		"SYK.UpdateLastUsed": func() error {
			return syk.UpdateLastUsed(ctx, id)
		},
		"SYK.Count":           func() error { return errOnly(syk.Count(ctx)) },
		"Audit.Create":        func() error { return audit.Create(ctx, &pkgmodels.ServiceAuditLog{SystemKeyID: sid}) },
		"Audit.FindAll":       func() error { return errOnly3(audit.FindAll(ctx, repository.ServiceAuditLogFilter{})) },
		"Audit.DeleteOlder":   func() error { return errOnly(audit.DeleteOlderThan(ctx, now)) },
		"User.Create":         func() error { return users.Create(ctx, &models.UserModel{}) },
		"User.Update":         func() error { return users.Update(ctx, &models.UserModel{ID: id}) },
		"User.Delete":         func() error { return users.Delete(ctx, id) },
		"User.HardDelete":     func() error { return users.HardDelete(ctx, id) },
		"User.FindByID":       func() error { return errOnly(users.FindByID(ctx, id)) },
		"User.FindByEmail":    func() error { return errOnly(users.FindByEmail(ctx, "e")) },
		"User.FindByUsername": func() error { return errOnly(users.FindByUsername(ctx, "u")) },
		"User.FindWithRoles":  func() error { return errOnly(users.FindByIDWithRoles(ctx, id)) },
		"User.FindAll":        func() error { return errOnly(users.FindAll(ctx, 1, 0)) },
		"User.FindAllActive":  func() error { return errOnly(users.FindAllActive(ctx, 1, 0)) },
		"User.ExistsEmail":    func() error { return errOnly(users.ExistsByEmail(ctx, "e")) },
		"User.ExistsUsername": func() error { return errOnly(users.ExistsByUsername(ctx, "u")) },
		"User.Count":          func() error { return errOnly(users.Count(ctx)) },
		"User.CountActive":    func() error { return errOnly(users.CountActive(ctx)) },
		"User.LastLogin":      func() error { return users.UpdateLastLogin(ctx, id) },
		"User.IncFailed":      func() error { return users.IncrementFailedAttempts(ctx, id) },
		"User.ResetFailed":    func() error { return users.ResetFailedAttempts(ctx, id) },
		"User.Lock":           func() error { return users.LockAccount(ctx, id, &lock) },
		"User.Unlock":         func() error { return users.UnlockAccount(ctx, id) },
		"User.CreateSession":  func() error { return users.CreateSession(ctx, &models.SessionModel{}) },
		"User.SessionByToken": func() error { return errOnly(users.FindSessionByToken(ctx, "t")) },
		"User.SessionByRef":   func() error { return errOnly(users.FindSessionByRefreshToken(ctx, "t")) },
		"User.SessionsByUser": func() error { return errOnly(users.FindSessionsByUserID(ctx, id)) },
		"User.DelSession":     func() error { return users.DeleteSession(ctx, "t") },
		"User.DelSessionByID": func() error { return users.DeleteSessionByID(ctx, id) },
		"User.DelSessions":    func() error { return users.DeleteSessionsByUserID(ctx, id) },
		"User.DelExpired":     func() error { return errOnly(users.DeleteExpiredSessions(ctx)) },
		"User.SessionAct":     func() error { return users.UpdateSessionActivity(ctx, "t") },
		"User.RoleByID":       func() error { return errOnly(users.FindRoleByID(ctx, id)) },
		"User.RoleByName":     func() error { return errOnly(users.FindRoleByName(ctx, "r")) },
		"User.AllRoles":       func() error { return errOnly(users.FindAllRoles(ctx)) },
		"User.CreateRole":     func() error { return users.CreateRole(ctx, &models.RoleModel{}) },
		"User.UpdateRole":     func() error { return users.UpdateRole(ctx, &models.RoleModel{}) },
		"User.DeleteRole":     func() error { return users.DeleteRole(ctx, id) },
		"User.AssignRole":     func() error { return users.AssignRole(ctx, id, id, nil) },
		"User.RemoveRole":     func() error { return users.RemoveRole(ctx, id, id) },
		"User.GetRoles":       func() error { return errOnly(users.GetUserRoles(ctx, id)) },
		"User.GetPerms":       func() error { return errOnly(users.GetUserPermissions(ctx, id)) },
		"User.HasPerm":        func() error { return errOnly(users.HasPermission(ctx, id, "p")) },
		"User.CreateAudit":    func() error { return users.CreateAuditLog(ctx, &models.AuditLogModel{}) },
		"User.FindAudit":      func() error { return errOnly(users.FindAuditLogs(ctx, &id, "a", 1, 0)) },
		// workflows
		"WF.Create":            func() error { return wf.Create(ctx, &models.WorkflowModel{}) },
		"WF.Update":            func() error { return wf.Update(ctx, &models.WorkflowModel{}) },
		"WF.Delete":            func() error { return wf.Delete(ctx, id) },
		"WF.HardDelete":        func() error { return wf.HardDelete(ctx, id) },
		"WF.FindByID":          func() error { return errOnly(wf.FindByID(ctx, id)) },
		"WF.FindWithRelations": func() error { return errOnly(wf.FindByIDWithRelations(ctx, id)) },
		"WF.FindByName":        func() error { return errOnly(wf.FindByName(ctx, "n", 1)) },
		"WF.FindAll":           func() error { return errOnly(wf.FindAll(ctx, 1, 0)) },
		"WF.FindByStatus":      func() error { return errOnly(wf.FindByStatus(ctx, "s", 1, 0)) },
		"WF.Count":             func() error { return errOnly(wf.Count(ctx)) },
		"WF.CountByStatus":     func() error { return errOnly(wf.CountByStatus(ctx, "s")) },
		"WF.FindFiltered":      func() error { return errOnly(wf.FindAllWithFilters(ctx, repository.WorkflowFilters{}, 1, 0)) },
		"WF.CountFiltered":     func() error { return errOnly(wf.CountWithFilters(ctx, repository.WorkflowFilters{})) },
		"WF.CreateNode":        func() error { return wf.CreateNode(ctx, &models.NodeModel{}) },
		"WF.UpdateNode":        func() error { return wf.UpdateNode(ctx, &models.NodeModel{}) },
		"WF.DeleteNode":        func() error { return wf.DeleteNode(ctx, id) },
		"WF.FindNode":          func() error { return errOnly(wf.FindNodeByID(ctx, id)) },
		"WF.FindNodes":         func() error { return errOnly(wf.FindNodesByWorkflowID(ctx, id)) },
		"WF.CreateEdge":        func() error { return wf.CreateEdge(ctx, &models.EdgeModel{}) },
		"WF.UpdateEdge":        func() error { return wf.UpdateEdge(ctx, &models.EdgeModel{}) },
		"WF.DeleteEdge":        func() error { return wf.DeleteEdge(ctx, id) },
		"WF.FindEdge":          func() error { return errOnly(wf.FindEdgeByID(ctx, id)) },
		"WF.FindEdges":         func() error { return errOnly(wf.FindEdgesByWorkflowID(ctx, id)) },
		"WF.ValidateDAG":       func() error { return wf.ValidateDAG(ctx, id) },
		"WF.AssignResource":    func() error { return wf.AssignResource(ctx, id, &models.WorkflowResourceModel{}, nil) },
		"WF.UnassignResource":  func() error { return wf.UnassignResource(ctx, id, id) },
		"WF.UnassignAll":       func() error { return errOnly(wf.UnassignResourceFromAllWorkflows(ctx, id)) },
		"WF.GetResources":      func() error { return errOnly(wf.GetWorkflowResources(ctx, id)) },
		"WF.UpdateAlias":       func() error { return wf.UpdateResourceAlias(ctx, id, id, "a") },
		"WF.ResourceExists":    func() error { return errOnly(wf.ResourceExists(ctx, id, id)) },
		"WF.ResourceByAlias":   func() error { return errOnly(wf.GetResourceByAlias(ctx, id, "a")) },
		// executions
		"Ex.Create":           func() error { return ex.Create(ctx, &models.ExecutionModel{}) },
		"Ex.Update":           func() error { return ex.Update(ctx, &models.ExecutionModel{ID: id}) },
		"Ex.Delete":           func() error { return ex.Delete(ctx, id) },
		"Ex.FindByID":         func() error { return errOnly(ex.FindByID(ctx, id)) },
		"Ex.FindWithRel":      func() error { return errOnly(ex.FindByIDWithRelations(ctx, id)) },
		"Ex.FindByWorkflow":   func() error { return errOnly(ex.FindByWorkflowID(ctx, id, 1, 0)) },
		"Ex.FindByStatus":     func() error { return errOnly(ex.FindByStatus(ctx, "s", 1, 0)) },
		"Ex.FindAll":          func() error { return errOnly(ex.FindAll(ctx, 1, 0)) },
		"Ex.FindRunning":      func() error { return errOnly(ex.FindRunning(ctx)) },
		"Ex.Count":            func() error { return errOnly(ex.Count(ctx)) },
		"Ex.CountByWorkflow":  func() error { return errOnly(ex.CountByWorkflowID(ctx, id)) },
		"Ex.CountByStatus":    func() error { return errOnly(ex.CountByStatus(ctx, "s")) },
		"Ex.CreateNodeExec":   func() error { return ex.CreateNodeExecution(ctx, &models.NodeExecutionModel{}) },
		"Ex.UpdateNodeExec":   func() error { return ex.UpdateNodeExecution(ctx, &models.NodeExecutionModel{ID: id}) },
		"Ex.DeleteNodeExec":   func() error { return ex.DeleteNodeExecution(ctx, id) },
		"Ex.FindNodeExec":     func() error { return errOnly(ex.FindNodeExecutionByID(ctx, id)) },
		"Ex.FindNodeExecs":    func() error { return errOnly(ex.FindNodeExecutionsByExecutionID(ctx, id)) },
		"Ex.FindByWave":       func() error { return errOnly(ex.FindNodeExecutionsByWave(ctx, id, 0)) },
		"Ex.FindNodeByStatus": func() error { return errOnly(ex.FindNodeExecutionsByStatus(ctx, id, "s")) },
		"Ex.Statistics":       func() error { return errOnly(ex.GetStatistics(ctx, &id, now, now)) },
		"Ex.GetEvents":        func() error { return errOnly(ex.GetEvents(ctx, id)) },
		"Ex.GetEventsSince":   func() error { return errOnly(ex.GetEventsSince(ctx, id, 0)) },
		"Ex.FindFiltered":     func() error { return errOnly(ex.FindWithFilters(ctx, repository.ExecutionFilters{}, 1, 0)) },
		// events
		"Ev.Append":         func() error { return ev.Append(ctx, &models.EventModel{}) },
		"Ev.AppendBatch":    func() error { return ev.AppendBatch(ctx, []*models.EventModel{{}}) },
		"Ev.ByExecution":    func() error { return errOnly(ev.FindByExecutionID(ctx, id)) },
		"Ev.ByExecSince":    func() error { return errOnly(ev.FindByExecutionIDSince(ctx, id, 0)) },
		"Ev.ByType":         func() error { return errOnly(ev.FindByType(ctx, "t", 1, 0)) },
		"Ev.ByTimeRange":    func() error { return errOnly(ev.FindByTimeRange(ctx, now, now, 1, 0)) },
		"Ev.Latest":         func() error { return errOnly(ev.FindLatestByExecutionID(ctx, id)) },
		"Ev.Count":          func() error { return errOnly(ev.Count(ctx)) },
		"Ev.CountByExec":    func() error { return errOnly(ev.CountByExecutionID(ctx, id)) },
		"Ev.CountByType":    func() error { return errOnly(ev.CountByType(ctx, "t")) },
		"Tr.Create":         func() error { return tr.Create(ctx, &models.TriggerModel{}) },
		"Tr.Update":         func() error { return tr.Update(ctx, &models.TriggerModel{ID: id}) },
		"Tr.Delete":         func() error { return tr.Delete(ctx, id) },
		"Tr.FindByID":       func() error { return errOnly(tr.FindByID(ctx, id)) },
		"Tr.ByWorkflow":     func() error { return errOnly(tr.FindByWorkflowID(ctx, id)) },
		"Tr.ByType":         func() error { return errOnly(tr.FindByType(ctx, "t", 1, 0)) },
		"Tr.Enabled":        func() error { return errOnly(tr.FindEnabled(ctx)) },
		"Tr.EnabledByType":  func() error { return errOnly(tr.FindEnabledByType(ctx, "t")) },
		"Tr.FindAll":        func() error { return errOnly(tr.FindAll(ctx, 1, 0)) },
		"Tr.Count":          func() error { return errOnly(tr.Count(ctx)) },
		"Tr.CountByWF":      func() error { return errOnly(tr.CountByWorkflowID(ctx, id)) },
		"Tr.CountByType":    func() error { return errOnly(tr.CountByType(ctx, "t")) },
		"Tr.Enable":         func() error { return tr.Enable(ctx, id) },
		"Tr.Disable":        func() error { return tr.Disable(ctx, id) },
		"Tr.MarkTriggered":  func() error { return tr.MarkTriggered(ctx, id) },
		"Tr.FindFiltered":   func() error { return errOnly(tr.FindWithFilters(ctx, repository.TriggerFilters{}, 1, 0)) },
		"Tr.CountFiltered":  func() error { return errOnly(tr.CountWithFilters(ctx, repository.TriggerFilters{})) },
		"File.Create":       func() error { return files.Create(ctx, &models.FileModel{}) },
		"File.Update":       func() error { return files.Update(ctx, &models.FileModel{ID: id}) },
		"File.Delete":       func() error { return files.Delete(ctx, id) },
		"File.FindByID":     func() error { return errOnly(files.FindByID(ctx, id)) },
		"File.FindByPath":   func() error { return errOnly(files.FindByStorageAndPath(ctx, "s", "p")) },
		"File.FindByQuery":  func() error { return errOnly(files.FindByQuery(ctx, &FileQuery{})) },
		"File.CountByQuery": func() error { return errOnly(files.CountByQuery(ctx, &FileQuery{})) },
		"File.FindExpired":  func() error { return errOnly(files.FindExpired(ctx, 1)) },
		"File.Usage": func() error {
			_, _, err := files.GetStorageUsage(ctx, "s")
			return err
		},
		"File.DeleteByStorage": func() error { return errOnly(files.DeleteByStorageID(ctx, "s")) },
		"File.DeleteExpired":   func() error { return errOnly(files.DeleteExpired(ctx)) },
		"Cfg.Create":           func() error { return cfgs.Create(ctx, &models.StorageConfigModel{}) },
		"Cfg.FindByStorageID":  func() error { return errOnly(cfgs.FindByStorageID(ctx, "s")) },
		"Cfg.Delete":           func() error { return cfgs.Delete(ctx, "s") },
		"Cfg.FindAll":          func() error { return errOnly(cfgs.FindAll(ctx)) },
		"API.Create":           func() error { return api.Create(ctx, &pkgmodels.APIClient{OwnerID: sid}) },
		"API.Update":           func() error { return api.Update(ctx, &pkgmodels.APIClient{ID: sid, OwnerID: sid}) },
		"API.Delete":           func() error { return api.Delete(ctx, id) },
		"API.FindByID":         func() error { return errOnly(api.FindByID(ctx, id)) },
		"API.FindSpec":         func() error { return errOnly(api.FindSpec(ctx, id)) },
		"API.FindUpdatedAt":    func() error { return errOnly(api.FindUpdatedAt(ctx, id)) },
		"API.FindByOwner":      func() error { return errOnly3(api.FindByOwner(ctx, id, 1, 0)) },
	}

	for name, call := range calls {
		err := call()
		require.Error(t, err, name)
		assert.Contains(t, err.Error(), "closed", "%s should surface the driver error, got %v", name, err)
	}

	// The event stream reports the failure on its error channel.
	events, errs := ev.Stream(ctx, id, 0)
	select {
	case err := <-errs:
		assert.Error(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("stream did not report the DB error")
	}
	_, open := <-events
	assert.False(t, open)
}

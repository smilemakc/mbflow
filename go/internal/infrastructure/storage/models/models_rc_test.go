package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
)

// ---------------------------------------------------------------------------
// Table names and hooks
// ---------------------------------------------------------------------------

func TestRC_TableNames(t *testing.T) {
	cases := map[string]string{
		BillingAccountModel{}.TableName():   "mbflow_billing_accounts",
		TransactionModel{}.TableName():      "mbflow_transactions",
		EdgeModel{}.TableName():             "mbflow_edges",
		EventModel{}.TableName():            "mbflow_events",
		ExecutionModel{}.TableName():        "mbflow_executions",
		FileModel{}.TableName():             "mbflow_files",
		StorageConfigModel{}.TableName():    "mbflow_storage_configs",
		NodeExecutionModel{}.TableName():    "mbflow_node_executions",
		NodeModel{}.TableName():             "mbflow_nodes",
		ResourceModel{}.TableName():         "mbflow_resources",
		FileStorageModel{}.TableName():      "mbflow_resource_file_storage",
		PricingPlanModel{}.TableName():      "mbflow_pricing_plans",
		CredentialsModel{}.TableName():      "mbflow_resource_credentials",
		RentalKeyModel{}.TableName():        "mbflow_resource_rental_key",
		RentalKeyUsageModel{}.TableName():   "mbflow_rental_key_usage",
		ServiceAuditLogModel{}.TableName():  "mbflow_service_audit_log",
		ServiceKeyModel{}.TableName():       "mbflow_service_keys",
		SystemKeyModel{}.TableName():        "mbflow_system_keys",
		TriggerModel{}.TableName():          "mbflow_triggers",
		UserModel{}.TableName():             "mbflow_users",
		SessionModel{}.TableName():          "mbflow_sessions",
		RoleModel{}.TableName():             "mbflow_roles",
		UserRoleModel{}.TableName():         "mbflow_user_roles",
		AuditLogModel{}.TableName():         "mbflow_audit_logs",
		WorkflowModel{}.TableName():         "mbflow_workflows",
		WorkflowResourceModel{}.TableName(): "mbflow_workflow_resources",
	}
	for got, want := range cases {
		assert.Equal(t, want, got)
	}
}

func rcAssertRecent(t *testing.T, ts time.Time) {
	t.Helper()
	assert.WithinDuration(t, time.Now(), ts, 5*time.Second)
}

func TestRC_BeforeInsertHooks_SetDefaults(t *testing.T) {
	t.Run("billing account", func(t *testing.T) {
		m := &BillingAccountModel{}
		require.NoError(t, m.BeforeInsert(nil))
		assert.NotEqual(t, uuid.Nil, m.ID)
		assert.Equal(t, "USD", m.Currency)
		assert.Equal(t, string(pkgmodels.AccountStatusActive), m.Status)
		rcAssertRecent(t, m.CreatedAt)
		assert.Equal(t, m.CreatedAt, m.UpdatedAt)

		id := uuid.New()
		m2 := &BillingAccountModel{ID: id, Currency: "EUR", Status: "closed"}
		require.NoError(t, m2.BeforeInsert(nil))
		assert.Equal(t, id, m2.ID)
		assert.Equal(t, "EUR", m2.Currency)
		assert.Equal(t, "closed", m2.Status)

		old := time.Now().Add(-time.Hour)
		m2.UpdatedAt = old
		require.NoError(t, m2.BeforeUpdate(nil))
		assert.True(t, m2.UpdatedAt.After(old))
	})

	t.Run("transaction", func(t *testing.T) {
		m := &TransactionModel{}
		require.NoError(t, m.BeforeInsert(nil))
		assert.NotEqual(t, uuid.Nil, m.ID)
		assert.NotNil(t, m.Metadata)
		assert.Equal(t, string(pkgmodels.TransactionStatusCompleted), m.Status)
		m2 := &TransactionModel{Status: "pending", Metadata: JSONBMap{"a": 1}}
		require.NoError(t, m2.BeforeInsert(nil))
		assert.Equal(t, "pending", m2.Status)
		assert.Equal(t, 1, m2.Metadata["a"])
	})

	t.Run("edge rejects self reference", func(t *testing.T) {
		e := &EdgeModel{FromNodeID: "a", ToNodeID: "a"}
		assert.ErrorIs(t, e.BeforeInsert(nil), ErrSelfReferenceEdge)
		assert.NotEqual(t, uuid.Nil, e.ID)
		assert.ErrorIs(t, e.BeforeUpdate(nil), ErrSelfReferenceEdge)

		ok := &EdgeModel{FromNodeID: "a", ToNodeID: "b"}
		require.NoError(t, ok.BeforeInsert(nil))
		require.NoError(t, ok.BeforeUpdate(nil))
		rcAssertRecent(t, ok.UpdatedAt)
	})

	t.Run("event", func(t *testing.T) {
		e := &EventModel{}
		require.NoError(t, e.BeforeInsert(nil))
		assert.NotEqual(t, uuid.Nil, e.ID)
		assert.NotNil(t, e.Payload)
		rcAssertRecent(t, e.CreatedAt)
	})

	t.Run("execution", func(t *testing.T) {
		e := &ExecutionModel{}
		require.NoError(t, e.BeforeInsert(nil))
		assert.NotEqual(t, uuid.Nil, e.ID)
		assert.NotNil(t, e.InputData)
		assert.NotNil(t, e.Variables)
		assert.NotNil(t, e.Metadata)
		require.NoError(t, e.BeforeUpdate(nil))
		rcAssertRecent(t, e.UpdatedAt)
	})

	t.Run("file and storage config", func(t *testing.T) {
		f := &FileModel{}
		require.NoError(t, f.BeforeInsert(nil))
		assert.NotEqual(t, uuid.Nil, f.ID)
		assert.NotNil(t, f.Metadata)
		require.NoError(t, f.BeforeUpdate(nil))

		s := &StorageConfigModel{}
		require.NoError(t, s.BeforeInsert(nil))
		assert.NotEqual(t, uuid.Nil, s.ID)
		assert.NotNil(t, s.Config)
		require.NoError(t, s.BeforeUpdate(nil))
		rcAssertRecent(t, s.UpdatedAt)
	})

	t.Run("node execution and node", func(t *testing.T) {
		ne := &NodeExecutionModel{}
		require.NoError(t, ne.BeforeInsert(nil))
		assert.NotEqual(t, uuid.Nil, ne.ID)
		assert.NotNil(t, ne.InputData)
		require.NoError(t, ne.BeforeUpdate(nil))

		n := &NodeModel{}
		require.NoError(t, n.BeforeInsert(nil))
		assert.NotEqual(t, uuid.Nil, n.ID)
		assert.NotNil(t, n.Config)
		assert.Equal(t, JSONBMap{"x": 0, "y": 0}, n.Position)
		require.NoError(t, n.BeforeUpdate(nil))
	})

	t.Run("resources", func(t *testing.T) {
		r := &ResourceModel{}
		require.NoError(t, r.BeforeInsert(nil))
		assert.NotEqual(t, uuid.Nil, r.ID)
		assert.NotNil(t, r.Metadata)
		assert.Equal(t, string(pkgmodels.ResourceStatusActive), r.Status)
		require.NoError(t, r.BeforeUpdate(nil))

		fs := &FileStorageModel{}
		require.NoError(t, fs.BeforeInsert(nil))
		assert.Equal(t, int64(5*1024*1024), fs.StorageLimitBytes)
		fs2 := &FileStorageModel{StorageLimitBytes: 10}
		require.NoError(t, fs2.BeforeInsert(nil))
		assert.Equal(t, int64(10), fs2.StorageLimitBytes)

		p := &PricingPlanModel{}
		require.NoError(t, p.BeforeInsert(nil))
		assert.NotEqual(t, uuid.Nil, p.ID)
		assert.Equal(t, string(pkgmodels.BillingPeriodMonthly), p.BillingPeriod)
		assert.Equal(t, string(pkgmodels.PricingModelFixed), p.PricingModel)
	})

	t.Run("keys and audit", func(t *testing.T) {
		sal := &ServiceAuditLogModel{}
		require.NoError(t, sal.BeforeInsert(nil))
		assert.NotEqual(t, uuid.Nil, sal.ID)

		sk := &ServiceKeyModel{}
		require.NoError(t, sk.BeforeInsert(nil))
		assert.NotEqual(t, uuid.Nil, sk.ID)
		assert.Equal(t, pkgmodels.ServiceKeyStatusActive, sk.Status)
		require.NoError(t, sk.BeforeUpdate(nil))

		syk := &SystemKeyModel{}
		require.NoError(t, syk.BeforeInsert(nil))
		assert.NotEqual(t, uuid.Nil, syk.ID)
		assert.Equal(t, pkgmodels.SystemKeyStatusActive, syk.Status)
		require.NoError(t, syk.BeforeUpdate(nil))
	})

	t.Run("trigger and workflow", func(t *testing.T) {
		tr := &TriggerModel{}
		require.NoError(t, tr.BeforeInsert(nil))
		assert.NotEqual(t, uuid.Nil, tr.ID)
		assert.NotNil(t, tr.Config)
		require.NoError(t, tr.BeforeUpdate(nil))

		w := &WorkflowModel{}
		require.NoError(t, w.BeforeInsert(nil))
		assert.NotEqual(t, uuid.Nil, w.ID)
		assert.NotNil(t, w.Metadata)
		assert.NotNil(t, w.Variables)
		require.NoError(t, w.BeforeUpdate(nil))

		wr := &WorkflowResourceModel{}
		require.NoError(t, wr.BeforeInsert(nil))
		assert.Equal(t, "read", wr.AccessType)
		rcAssertRecent(t, wr.AssignedAt)
		wr2 := &WorkflowResourceModel{AccessType: "admin"}
		require.NoError(t, wr2.BeforeInsert(nil))
		assert.Equal(t, "admin", wr2.AccessType)
	})

	t.Run("users", func(t *testing.T) {
		u := &UserModel{}
		require.NoError(t, u.BeforeInsert(nil))
		assert.NotEqual(t, uuid.Nil, u.ID)
		assert.NotNil(t, u.Metadata)
		require.NoError(t, u.BeforeUpdate(nil))

		s := &SessionModel{}
		require.NoError(t, s.BeforeInsert(nil))
		assert.NotEqual(t, uuid.Nil, s.ID)
		assert.NotNil(t, s.Metadata)
		rcAssertRecent(t, s.LastActivityAt)

		r := &RoleModel{}
		require.NoError(t, r.BeforeInsert(nil))
		assert.NotEqual(t, uuid.Nil, r.ID)
		assert.NotNil(t, r.Metadata)
		assert.NotNil(t, r.Permissions)
		assert.Empty(t, r.Permissions)
		require.NoError(t, r.BeforeUpdate(nil))

		ur := &UserRoleModel{}
		require.NoError(t, ur.BeforeInsert(nil))
		rcAssertRecent(t, ur.AssignedAt)

		al := &AuditLogModel{}
		require.NoError(t, al.BeforeInsert(nil))
		assert.NotEqual(t, uuid.Nil, al.ID)
		assert.NotNil(t, al.Metadata)
	})
}

// ---------------------------------------------------------------------------
// Behaviour helpers
// ---------------------------------------------------------------------------

func TestRC_BillingAccount_Predicates(t *testing.T) {
	a := &BillingAccountModel{Status: string(pkgmodels.AccountStatusActive), Balance: 10}
	assert.True(t, a.IsActive())
	assert.True(t, a.HasSufficientBalance(10))
	assert.False(t, a.HasSufficientBalance(10.01))
	assert.True(t, a.CanCharge(5))
	assert.False(t, a.CanCharge(11))
	a.Status = "suspended"
	assert.False(t, a.IsActive())
	assert.False(t, a.CanCharge(1))
}

func TestRC_Transaction_StatusPredicates(t *testing.T) {
	cases := []struct {
		status                                         string
		completed, pending, failed, reversed, terminal bool
	}{
		{"completed", true, false, false, false, true},
		{"pending", false, true, false, false, false},
		{"failed", false, false, true, false, true},
		{"reversed", false, false, false, true, true},
	}
	for _, c := range cases {
		tx := &TransactionModel{Status: c.status}
		assert.Equal(t, c.completed, tx.IsCompleted(), c.status)
		assert.Equal(t, c.pending, tx.IsPending(), c.status)
		assert.Equal(t, c.failed, tx.IsFailed(), c.status)
		assert.Equal(t, c.reversed, tx.IsReversed(), c.status)
		assert.Equal(t, c.terminal, tx.IsTerminal(), c.status)
	}
}

func TestRC_Resource_Predicates(t *testing.T) {
	r := &ResourceModel{Status: "active"}
	assert.True(t, r.IsActive())
	assert.False(t, r.IsDeleted())
	now := time.Now()
	r.Status = "deleted"
	r.DeletedAt = &now
	assert.False(t, r.IsActive())
	assert.True(t, r.IsDeleted())
}

func TestRC_FileStorage_Usage(t *testing.T) {
	fs := &FileStorageModel{}
	assert.Equal(t, 0.0, fs.GetUsagePercent())

	fs = &FileStorageModel{StorageLimitBytes: 200, UsedStorageBytes: 50}
	assert.InDelta(t, 25.0, fs.GetUsagePercent(), 1e-9)
	assert.True(t, fs.CanAddFile(150))
	assert.False(t, fs.CanAddFile(151))
	assert.Equal(t, int64(150), fs.GetAvailableSpace())

	fs.UsedStorageBytes = 300
	assert.Equal(t, int64(0), fs.GetAvailableSpace())
}

func TestRC_PricingPlan_Prices(t *testing.T) {
	free := &PricingPlanModel{IsFree: true, PricePerUnit: 100, IsActive: true}
	assert.Equal(t, 0.0, free.GetMonthlyPrice())
	assert.Equal(t, 0.0, free.GetAnnualPrice())
	assert.True(t, free.IsAvailable())

	monthly := &PricingPlanModel{PricePerUnit: 10, BillingPeriod: "monthly"}
	assert.Equal(t, 10.0, monthly.GetMonthlyPrice())
	assert.Equal(t, 120.0, monthly.GetAnnualPrice())
	assert.False(t, monthly.IsAvailable())

	annual := &PricingPlanModel{PricePerUnit: 120, BillingPeriod: "annual"}
	assert.Equal(t, 10.0, annual.GetMonthlyPrice())
	assert.Equal(t, 120.0, annual.GetAnnualPrice())
}

func TestRC_User_Predicates(t *testing.T) {
	u := &UserModel{IsActive: true}
	assert.False(t, u.IsLocked())
	assert.False(t, u.IsDeleted())
	assert.True(t, u.CanLogin())

	future := time.Now().Add(time.Hour)
	u.LockedUntil = &future
	assert.True(t, u.IsLocked())
	assert.False(t, u.CanLogin())

	past := time.Now().Add(-time.Hour)
	u.LockedUntil = &past
	assert.False(t, u.IsLocked())
	assert.True(t, u.CanLogin())

	u.DeletedAt = &past
	assert.True(t, u.IsDeleted())
	assert.False(t, u.CanLogin())

	u2 := &UserModel{IsActive: false}
	assert.False(t, u2.CanLogin())
}

func TestRC_Session_Expiry(t *testing.T) {
	s := &SessionModel{ExpiresAt: time.Now().Add(time.Hour)}
	assert.False(t, s.IsExpired())
	assert.True(t, s.IsValid())
	s.ExpiresAt = time.Now().Add(-time.Second)
	assert.True(t, s.IsExpired())
	assert.False(t, s.IsValid())
}

func TestRC_Role_HasPermission(t *testing.T) {
	r := &RoleModel{Permissions: StringArray{"read", "write"}}
	assert.True(t, r.HasPermission("write"))
	assert.False(t, r.HasPermission("admin"))
	assert.False(t, (&RoleModel{}).HasPermission("read"))
}

func TestRC_WorkflowResource_Access(t *testing.T) {
	cases := []struct {
		access                 string
		readOnly, write, admin bool
	}{
		{"read", true, false, false},
		{"write", false, true, false},
		{"admin", false, true, true},
	}
	for _, c := range cases {
		wr := &WorkflowResourceModel{AccessType: c.access}
		assert.Equal(t, c.readOnly, wr.IsReadOnly(), c.access)
		assert.Equal(t, c.write, wr.IsWritable(), c.access)
		assert.Equal(t, c.admin, wr.IsAdmin(), c.access)
	}
}

func TestRC_RentalKeyUsage_GetTotalTokens(t *testing.T) {
	u := &RentalKeyUsageModel{
		PromptTokens: 1, CompletionTokens: 2,
		ImageInputTokens: 3, ImageOutputTokens: 4,
		AudioInputTokens: 5, AudioOutputTokens: 6,
		VideoInputTokens: 7, VideoOutputTokens: 8,
	}
	assert.Equal(t, 36, u.GetTotalTokens())
}

// ---------------------------------------------------------------------------
// Account / Transaction conversions
// ---------------------------------------------------------------------------

func TestRC_AccountConversions(t *testing.T) {
	assert.Nil(t, ToAccountDomain(nil))
	assert.Nil(t, FromAccountDomain(nil))
	assert.Nil(t, ToTransactionDomain(nil))
	assert.Nil(t, FromTransactionDomain(nil))

	now := time.Now().UTC().Truncate(time.Second)
	acc := &pkgmodels.Account{
		ID:        uuid.NewString(),
		UserID:    uuid.NewString(),
		Balance:   12.5,
		Currency:  "EUR",
		Status:    pkgmodels.AccountStatusSuspended,
		CreatedAt: now,
		UpdatedAt: now,
	}
	m := FromAccountDomain(acc)
	assert.Equal(t, acc.ID, m.ID.String())
	assert.Equal(t, acc.UserID, m.UserID.String())
	assert.Equal(t, *acc, *ToAccountDomain(m))

	empty := FromAccountDomain(&pkgmodels.Account{})
	assert.Equal(t, uuid.Nil, empty.ID)
	assert.Equal(t, uuid.Nil, empty.UserID)

	tx := &pkgmodels.Transaction{
		ID:             uuid.NewString(),
		AccountID:      acc.ID,
		Type:           pkgmodels.TransactionTypeDeposit,
		Amount:         5,
		Currency:       "EUR",
		Status:         pkgmodels.TransactionStatusPending,
		Description:    "top up",
		ReferenceType:  "invoice",
		ReferenceID:    uuid.NewString(),
		IdempotencyKey: "idem-1",
		BalanceBefore:  1,
		BalanceAfter:   6,
		Metadata:       map[string]any{"k": "v"},
		CreatedAt:      now,
	}
	tm := FromTransactionDomain(tx)
	require.NotNil(t, tm.ReferenceID)
	assert.Equal(t, tx.ReferenceID, tm.ReferenceID.String())
	back := ToTransactionDomain(tm)
	assert.Equal(t, *tx, *back)

	bare := FromTransactionDomain(&pkgmodels.Transaction{})
	assert.Nil(t, bare.ReferenceID)
	assert.Nil(t, bare.Metadata)
	assert.Equal(t, uuid.Nil, bare.ID)
	bareDomain := ToTransactionDomain(bare)
	assert.Empty(t, bareDomain.ReferenceID)
	assert.Nil(t, bareDomain.Metadata)
}

// ---------------------------------------------------------------------------
// API client conversions
// ---------------------------------------------------------------------------

func TestRC_APIClientConversions(t *testing.T) {
	assert.Nil(t, (*APIClientModel)(nil).ToAPIClientDomain())
	m, err := FromAPIClientDomain(nil)
	assert.NoError(t, err)
	assert.Nil(t, m)

	_, err = FromAPIClientDomain(&pkgmodels.APIClient{OwnerID: "bad"})
	assert.ErrorIs(t, err, pkgmodels.ErrInvalidID)
	_, err = FromAPIClientDomain(&pkgmodels.APIClient{OwnerID: uuid.NewString(), ID: "bad"})
	assert.ErrorIs(t, err, pkgmodels.ErrInvalidID)

	// nil collections become empty ones (JSONB NOT NULL columns)
	c := &pkgmodels.APIClient{OwnerID: uuid.NewString(), Name: "petstore"}
	m, err = FromAPIClientDomain(c)
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, m.ID)
	assert.NotNil(t, m.Servers)
	assert.NotNil(t, m.SecuritySchemes)
	assert.NotNil(t, m.DefaultHeaders)
	assert.NotNil(t, m.Operations)
	assert.NotNil(t, m.Schemas)
	assert.NotNil(t, m.Warnings)
	assert.Equal(t, 0, m.OperationCount)

	id := uuid.NewString()
	c = &pkgmodels.APIClient{
		ID:             id,
		OwnerID:        uuid.NewString(),
		Name:           "n",
		Title:          "T",
		SpecVersion:    "3.0.0",
		Operations:     []*pkgmodels.APIOperation{{}, {}},
		DefaultHeaders: map[string]string{"X": "1"},
		SpecRaw:        "{}",
		SpecSize:       2,
	}
	m, err = FromAPIClientDomain(c)
	require.NoError(t, err)
	assert.Equal(t, id, m.ID.String())
	assert.Equal(t, 2, m.OperationCount)

	d := m.ToAPIClientDomain()
	assert.Equal(t, id, d.ID)
	assert.Equal(t, c.OwnerID, d.OwnerID)
	assert.Equal(t, 2, d.OperationCount)
	assert.Equal(t, "1", d.DefaultHeaders["X"])
	assert.Len(t, d.Operations, 2)

	d = (&APIClientModel{}).ToAPIClientDomain()
	assert.NotNil(t, d.Servers)
	assert.NotNil(t, d.Schemas)
}

// ---------------------------------------------------------------------------
// Event / Trigger / AuditLog mappers
// ---------------------------------------------------------------------------

func TestRC_EventMappers(t *testing.T) {
	assert.Nil(t, EventModelToDomain(nil))
	assert.Nil(t, EventDomainToModel(nil))
	assert.Nil(t, EventModelsToDomain(nil))

	now := time.Now()
	e := &pkgmodels.Event{
		ID:          uuid.NewString(),
		ExecutionID: uuid.NewString(),
		EventType:   "node.started",
		Sequence:    7,
		Payload:     map[string]any{"a": "b"},
		CreatedAt:   now,
	}
	m := EventDomainToModel(e)
	assert.Equal(t, e.ID, m.ID.String())
	assert.Equal(t, e.ExecutionID, m.ExecutionID.String())
	assert.Equal(t, *e, *EventModelToDomain(m))

	bad := EventDomainToModel(&pkgmodels.Event{ID: "x", ExecutionID: "y"})
	assert.Equal(t, uuid.Nil, bad.ID)
	assert.Equal(t, uuid.Nil, bad.ExecutionID)
	assert.Nil(t, EventModelToDomain(&EventModel{}).Payload)

	list := EventModelsToDomain([]*EventModel{m, nil})
	require.Len(t, list, 2)
	assert.Equal(t, e.ID, list[0].ID)
	assert.Nil(t, list[1])
}

func TestRC_TriggerMappers(t *testing.T) {
	assert.Nil(t, TriggerModelToDomain(nil))
	assert.Nil(t, TriggerDomainToModel(nil))
	assert.Nil(t, TriggerModelsToDomain(nil))

	now := time.Now()
	tr := &pkgmodels.Trigger{
		ID:         uuid.NewString(),
		WorkflowID: uuid.NewString(),
		Type:       pkgmodels.TriggerTypeCron,
		Config:     map[string]any{"schedule": "* * * * *"},
		Enabled:    true,
		CreatedAt:  now,
		UpdatedAt:  now,
		LastRun:    &now,
	}
	m := TriggerDomainToModel(tr)
	assert.Equal(t, tr.ID, m.ID.String())
	assert.Equal(t, tr.WorkflowID, m.WorkflowID.String())
	assert.Equal(t, &now, m.LastTriggeredAt)

	back := TriggerModelToDomain(m)
	assert.Equal(t, tr.ID, back.ID)
	assert.Equal(t, tr.WorkflowID, back.WorkflowID)
	assert.Equal(t, tr.Type, back.Type)
	assert.Equal(t, tr.Config, back.Config)
	assert.True(t, back.Enabled)
	assert.Equal(t, &now, back.LastRun)

	bad := TriggerDomainToModel(&pkgmodels.Trigger{ID: "x", WorkflowID: "y"})
	assert.Equal(t, uuid.Nil, bad.ID)
	assert.Equal(t, uuid.Nil, bad.WorkflowID)
	assert.Nil(t, TriggerModelToDomain(&TriggerModel{}).Config)

	list := TriggerModelsToDomain([]*TriggerModel{m})
	require.Len(t, list, 1)
	assert.Equal(t, tr.ID, list[0].ID)
}

func TestRC_AuditLogMappers(t *testing.T) {
	assert.Nil(t, AuditLogModelToDomain(nil))
	assert.Nil(t, AuditLogDomainToModel(nil))
	assert.Nil(t, AuditLogModelsToDomain(nil))

	uid := uuid.NewString()
	rid := uuid.NewString()
	a := &pkgmodels.AuditLog{
		ID:           uuid.NewString(),
		UserID:       &uid,
		Action:       "login",
		ResourceType: "user",
		ResourceID:   &rid,
		IPAddress:    "127.0.0.1",
		UserAgent:    "ua",
		Metadata:     map[string]any{"ok": true},
		CreatedAt:    time.Now(),
	}
	m := AuditLogDomainToModel(a)
	require.NotNil(t, m.UserID)
	require.NotNil(t, m.ResourceID)
	assert.Equal(t, uid, m.UserID.String())
	back := AuditLogModelToDomain(m)
	assert.Equal(t, a.ID, back.ID)
	assert.Equal(t, uid, *back.UserID)
	assert.Equal(t, rid, *back.ResourceID)
	assert.Equal(t, a.Metadata, back.Metadata)

	empty := ""
	bad := "nope"
	m2 := AuditLogDomainToModel(&pkgmodels.AuditLog{ID: "bad", UserID: &empty, ResourceID: &bad})
	assert.Equal(t, uuid.Nil, m2.ID)
	assert.Nil(t, m2.UserID)
	assert.Nil(t, m2.ResourceID)
	m3 := AuditLogDomainToModel(&pkgmodels.AuditLog{UserID: &bad, ResourceID: &empty})
	assert.Nil(t, m3.UserID)
	assert.Nil(t, m3.ResourceID)

	d := AuditLogModelToDomain(&AuditLogModel{})
	assert.Nil(t, d.UserID)
	assert.Nil(t, d.ResourceID)
	assert.Nil(t, d.Metadata)

	list := AuditLogModelsToDomain([]*AuditLogModel{m})
	require.Len(t, list, 1)
	assert.Equal(t, a.ID, list[0].ID)
}

func TestRC_ParseJSONInt(t *testing.T) {
	assert.Equal(t, 3, parseJSONInt(float64(3.9)))
	assert.Equal(t, 4, parseJSONInt(4))
	assert.Equal(t, 5, parseJSONInt(int64(5)))
	assert.Equal(t, 0, parseJSONInt("6"))
	assert.Equal(t, 0, parseJSONInt(nil))
}

// ---------------------------------------------------------------------------
// Workflow / Node / Edge / WorkflowResource mappers
// ---------------------------------------------------------------------------

func TestRC_WorkflowResourceMappers(t *testing.T) {
	wfID := uuid.New()
	resID := uuid.NewString()
	domains := []pkgmodels.WorkflowResource{
		{ResourceID: resID, Alias: "files", AccessType: "write"},
		{ResourceID: "not-a-uuid", Alias: "bad", AccessType: "read"},
	}
	models := WorkflowResourcesToStorage(domains, wfID)
	require.Len(t, models, 2)
	assert.Equal(t, wfID, models[0].WorkflowID)
	assert.Equal(t, resID, models[0].ResourceID.String())
	assert.Equal(t, "files", models[0].Alias)
	assert.Equal(t, "write", models[0].AccessType)
	rcAssertRecent(t, models[0].AssignedAt)
	assert.Equal(t, uuid.Nil, models[1].ResourceID)

	back := WorkflowResourcesFromStorage(models[:1])
	require.Len(t, back, 1)
	assert.Equal(t, domains[0], back[0])
	assert.Empty(t, WorkflowResourcesFromStorage(nil))
}

func TestRC_WorkflowModelToDomain(t *testing.T) {
	assert.Nil(t, WorkflowModelToDomain(nil))
	assert.Nil(t, NodeModelToDomain(nil))
	assert.Nil(t, EdgeModelToDomain(nil))

	creator := uuid.New()
	resID := uuid.New()
	wm := &WorkflowModel{
		ID:          uuid.New(),
		Name:        "wf",
		Description: "d",
		Status:      "active",
		Variables:   JSONBMap{"v": 1},
		Metadata:    JSONBMap{"m": 2},
		CreatedBy:   &creator,
		Nodes: []*NodeModel{
			{NodeID: "n1", Name: "N1", Type: "http", Config: JSONBMap{"url": "x"}, Position: JSONBMap{"x": 1.0, "y": 2.0}},
			{NodeID: "n2", Name: "N2", Type: "llm", Position: JSONBMap{"x": 1.0}},
		},
		Edges: []*EdgeModel{
			{EdgeID: "e1", FromNodeID: "n1", ToNodeID: "n2", Condition: JSONBMap{"expression": "a > 1"}, Loop: JSONBMap{"max_iterations": 3.0}},
		},
		Resources: []*WorkflowResourceModel{
			{ResourceID: resID, Alias: "r", AccessType: "read", Resource: &ResourceModel{Name: "store", Type: "file_storage"}},
			{ResourceID: uuid.New(), Alias: "r2", AccessType: "write"},
		},
	}
	w := WorkflowModelToDomain(wm)
	assert.Equal(t, wm.ID.String(), w.ID)
	assert.Equal(t, creator.String(), w.CreatedBy)
	assert.Equal(t, pkgmodels.WorkflowStatusActive, w.Status)
	assert.Equal(t, 1, w.Variables["v"])
	assert.Equal(t, 2, w.Metadata["m"])
	require.Len(t, w.Nodes, 2)
	assert.Equal(t, &pkgmodels.Position{X: 1, Y: 2}, w.Nodes[0].Position)
	assert.Equal(t, "x", w.Nodes[0].Config["url"])
	assert.Nil(t, w.Nodes[1].Position, "position without y must be ignored")
	assert.NotNil(t, w.Nodes[1].Config)
	require.Len(t, w.Edges, 1)
	assert.Equal(t, "a > 1", w.Edges[0].Condition)
	require.NotNil(t, w.Edges[0].Loop)
	assert.Equal(t, 3, w.Edges[0].Loop.MaxIterations)
	require.Len(t, w.Resources, 2)
	assert.Equal(t, "store", w.Resources[0].ResourceName)
	assert.Equal(t, "file_storage", w.Resources[0].ResourceType)
	assert.Empty(t, w.Resources[1].ResourceName)

	minimal := WorkflowModelToDomain(&WorkflowModel{})
	assert.Empty(t, minimal.CreatedBy)
	assert.NotNil(t, minimal.Variables)
	assert.NotNil(t, minimal.Metadata)
	assert.Empty(t, minimal.Nodes)

	e := EdgeModelToDomain(&EdgeModel{Condition: JSONBMap{"other": 1}, Loop: JSONBMap{"max_iterations": 0}})
	assert.Empty(t, e.Condition)
	assert.Nil(t, e.Loop)
}

// ---------------------------------------------------------------------------
// Execution mappers
// ---------------------------------------------------------------------------

func TestRC_ExecutionMappers_RoundTrip(t *testing.T) {
	assert.Nil(t, ExecutionModelToDomain(nil))
	assert.Nil(t, ExecutionDomainToModel(nil))
	assert.Nil(t, NodeExecutionModelToDomain(nil))
	assert.Nil(t, NodeExecutionDomainToModel(nil))

	started := time.Now().Add(-time.Minute)
	completed := time.Now()
	nodeUUID := uuid.NewString()
	exec := &pkgmodels.Execution{
		ID:             uuid.NewString(),
		WorkflowID:     uuid.NewString(),
		WorkflowSource: "stored",
		Status:         pkgmodels.ExecutionStatusFailed,
		Input:          map[string]any{"in": 1},
		Output:         map[string]any{"out": 2},
		Variables:      map[string]any{"var": 3},
		Error:          "boom",
		StartedAt:      started,
		CompletedAt:    &completed,
		NodeExecutions: []*pkgmodels.NodeExecution{
			{
				ID:             uuid.NewString(),
				NodeID:         nodeUUID,
				NodeName:       "N",
				NodeType:       "http",
				Status:         pkgmodels.NodeExecutionStatusCompleted,
				Input:          map[string]any{"a": 1},
				Output:         map[string]any{"b": 2},
				Config:         map[string]any{"c": 3},
				ResolvedConfig: map[string]any{"d": 4},
				StartedAt:      started,
				CompletedAt:    &completed,
				RetryCount:     2,
				Error:          "e",
			},
			{NodeID: "logical-key", Status: pkgmodels.NodeExecutionStatusPending},
			nil,
		},
	}
	exec.NodeExecutions[0].ExecutionID = exec.ID

	m := ExecutionDomainToModel(exec)
	assert.Equal(t, exec.ID, m.ID.String())
	require.NotNil(t, m.WorkflowID)
	assert.Equal(t, exec.WorkflowID, m.WorkflowID.String())
	assert.Equal(t, "stored", m.WorkflowSource)
	require.Len(t, m.NodeExecutions, 2, "nil node executions must be skipped")
	ne0 := m.NodeExecutions[0]
	require.NotNil(t, ne0.NodeID)
	assert.Equal(t, nodeUUID, ne0.NodeID.String())
	assert.Nil(t, ne0.NodeKey)
	ne1 := m.NodeExecutions[1]
	assert.Nil(t, ne1.NodeID)
	require.NotNil(t, ne1.NodeKey)
	assert.Equal(t, "logical-key", *ne1.NodeKey)
	assert.NotEqual(t, uuid.Nil, ne1.ID, "missing ID is generated")
	assert.Nil(t, ne1.StartedAt)
	assert.Nil(t, ne1.CompletedAt)
	assert.Nil(t, ne1.NodeName)

	back := ExecutionModelToDomain(m)
	assert.Equal(t, exec.ID, back.ID)
	assert.Equal(t, exec.WorkflowID, back.WorkflowID)
	assert.Equal(t, exec.Status, back.Status)
	assert.Equal(t, exec.Input, back.Input)
	assert.Equal(t, exec.Output, back.Output)
	assert.Equal(t, exec.Variables, back.Variables)
	assert.Equal(t, "boom", back.Error)
	assert.Equal(t, started, back.StartedAt)
	assert.Equal(t, &completed, back.CompletedAt)
	require.Len(t, back.NodeExecutions, 2)
	n0 := back.NodeExecutions[0]
	assert.Equal(t, exec.NodeExecutions[0].ID, n0.ID)
	assert.Equal(t, exec.ID, n0.ExecutionID)
	assert.Equal(t, nodeUUID, n0.NodeID)
	assert.Equal(t, "N", n0.NodeName)
	assert.Equal(t, "http", n0.NodeType)
	assert.Equal(t, 2, n0.RetryCount)
	assert.Equal(t, "e", n0.Error)
	assert.Equal(t, map[string]any{"c": 3}, n0.Config)
	assert.Equal(t, map[string]any{"d": 4}, n0.ResolvedConfig)
	assert.Equal(t, &completed, n0.CompletedAt)
	assert.Equal(t, "logical-key", back.NodeExecutions[1].NodeID)
}

func TestRC_ExecutionMappers_Defaults(t *testing.T) {
	d := ExecutionModelToDomain(&ExecutionModel{Status: "pending"})
	assert.Empty(t, d.WorkflowID)
	assert.NotNil(t, d.Input)
	assert.NotNil(t, d.Output)
	assert.NotNil(t, d.Variables)
	assert.Nil(t, d.CompletedAt)
	assert.Nil(t, d.NodeExecutions)
	assert.True(t, d.StartedAt.IsZero())

	m := ExecutionDomainToModel(&pkgmodels.Execution{ID: "bad", WorkflowID: "bad"})
	assert.Equal(t, uuid.Nil, m.ID)
	assert.Nil(t, m.WorkflowID)
	assert.Nil(t, m.CompletedAt)
	assert.Nil(t, m.NodeExecutions)

	ne := NodeExecutionModelToDomain(&NodeExecutionModel{})
	assert.Empty(t, ne.NodeID)
	assert.NotNil(t, ne.Input)
	assert.NotNil(t, ne.Config)

	nem := NodeExecutionDomainToModel(&pkgmodels.NodeExecution{ID: "bad", ExecutionID: "bad"})
	assert.NotEqual(t, uuid.Nil, nem.ID)
	assert.Equal(t, uuid.Nil, nem.ExecutionID)
	assert.Nil(t, nem.NodeID)
	assert.Nil(t, nem.NodeKey)

	zero := time.Time{}
	nem = NodeExecutionDomainToModel(&pkgmodels.NodeExecution{CompletedAt: &zero})
	assert.Nil(t, nem.CompletedAt)
}

// ---------------------------------------------------------------------------
// Service audit log / service key / system key conversions
// ---------------------------------------------------------------------------

func TestRC_ServiceAuditLogConversions(t *testing.T) {
	assert.Nil(t, (*ServiceAuditLogModel)(nil).ToServiceAuditLogDomain())
	assert.Nil(t, FromServiceAuditLogDomain(nil))

	uid := uuid.NewString()
	rid := uuid.NewString()
	body := `{"x":1}`
	log := &pkgmodels.ServiceAuditLog{
		ID:                 uuid.NewString(),
		SystemKeyID:        uuid.NewString(),
		ServiceName:        "svc",
		ImpersonatedUserID: &uid,
		Action:             "create",
		ResourceType:       "workflow",
		ResourceID:         &rid,
		RequestMethod:      "POST",
		RequestPath:        "/api",
		RequestBody:        &body,
		ResponseStatus:     201,
		IPAddress:          "10.0.0.1",
		CreatedAt:          time.Now(),
	}
	m := FromServiceAuditLogDomain(log)
	assert.Equal(t, *log, *m.ToServiceAuditLogDomain())

	bare := FromServiceAuditLogDomain(&pkgmodels.ServiceAuditLog{SystemKeyID: uuid.NewString()})
	assert.Equal(t, uuid.Nil, bare.ID)
	assert.Nil(t, bare.ImpersonatedUserID)
	assert.Nil(t, bare.ResourceID)
	d := bare.ToServiceAuditLogDomain()
	assert.Nil(t, d.ImpersonatedUserID)
	assert.Nil(t, d.ResourceID)
}

func TestRC_ServiceKeyConversions(t *testing.T) {
	assert.Nil(t, (*ServiceKeyModel)(nil).ToServiceKeyDomain())
	assert.Nil(t, FromServiceKeyDomain(nil))

	now := time.Now()
	key := &pkgmodels.ServiceKey{
		ID:          uuid.NewString(),
		UserID:      uuid.NewString(),
		Name:        "k",
		Description: "d",
		KeyPrefix:   "sk_abc",
		KeyHash:     "hash",
		Status:      "active",
		LastUsedAt:  &now,
		UsageCount:  3,
		ExpiresAt:   &now,
		CreatedBy:   uuid.NewString(),
		CreatedAt:   now,
		UpdatedAt:   now,
		RevokedAt:   &now,
	}
	assert.Equal(t, *key, *FromServiceKeyDomain(key).ToServiceKeyDomain())

	key.ID = ""
	assert.Equal(t, uuid.Nil, FromServiceKeyDomain(key).ID)
}

func TestRC_SystemKeyConversions(t *testing.T) {
	assert.Nil(t, (*SystemKeyModel)(nil).ToSystemKeyDomain())
	assert.Nil(t, FromSystemKeyDomain(nil))

	now := time.Now()
	key := &pkgmodels.SystemKey{
		ID:          uuid.NewString(),
		Name:        "k",
		Description: "d",
		KeyPrefix:   "sys_abc",
		KeyHash:     "hash",
		ServiceName: "svc",
		Status:      "active",
		LastUsedAt:  &now,
		UsageCount:  1,
		ExpiresAt:   &now,
		CreatedBy:   uuid.NewString(),
		CreatedAt:   now,
		UpdatedAt:   now,
		RevokedAt:   &now,
	}
	assert.Equal(t, *key, *FromSystemKeyDomain(key).ToSystemKeyDomain())

	key.ID = ""
	assert.Equal(t, uuid.Nil, FromSystemKeyDomain(key).ID)
}

// ---------------------------------------------------------------------------
// User conversions
// ---------------------------------------------------------------------------

func TestRC_UserConversions(t *testing.T) {
	assert.Nil(t, ToUserDomain(nil, nil))
	assert.Nil(t, ToRoleDomain(nil))
	assert.Nil(t, ToSessionDomain(nil))

	now := time.Now()
	u := &UserModel{
		ID:           uuid.New(),
		Email:        "a@b.c",
		Username:     "abc",
		PasswordHash: "h",
		FullName:     "A B",
		IsActive:     true,
		IsAdmin:      true,
		Metadata:     JSONBMap{"k": "v"},
		CreatedAt:    now,
		UpdatedAt:    now,
		LastLoginAt:  &now,
	}
	d := ToUserDomain(u, []string{"admin"})
	assert.Equal(t, u.ID.String(), d.ID)
	assert.Equal(t, "a@b.c", d.Email)
	assert.Equal(t, "abc", d.Username)
	assert.Equal(t, "h", d.PasswordHash)
	assert.True(t, d.IsActive)
	assert.True(t, d.IsAdmin)
	assert.Equal(t, []string{"admin"}, d.Roles)
	assert.Equal(t, "v", d.Metadata["k"])
	assert.Equal(t, &now, d.LastLoginAt)
	assert.Nil(t, ToUserDomain(&UserModel{}, nil).Metadata)

	r := &RoleModel{ID: uuid.New(), Name: "admin", Description: "d", IsSystem: true, Permissions: StringArray{"a", "b"}, CreatedAt: now, UpdatedAt: now}
	rd := ToRoleDomain(r)
	assert.Equal(t, r.ID.String(), rd.ID)
	assert.Equal(t, []string{"a", "b"}, rd.Permissions)
	assert.True(t, rd.IsSystem)
	// permissions must be copied, not aliased
	rd.Permissions[0] = "changed"
	assert.Equal(t, "a", r.Permissions[0])

	s := &SessionModel{ID: uuid.New(), UserID: uuid.New(), Token: "t", RefreshToken: "rt", ExpiresAt: now, IPAddress: "ip", UserAgent: "ua", CreatedAt: now}
	sd := ToSessionDomain(s)
	assert.Equal(t, s.ID.String(), sd.ID)
	assert.Equal(t, s.UserID.String(), sd.UserID)
	assert.Equal(t, "t", sd.Token)
	assert.Equal(t, "rt", sd.RefreshToken)
	assert.Equal(t, "ip", sd.IPAddress)
	assert.Equal(t, "ua", sd.UserAgent)
}

// ---------------------------------------------------------------------------
// Resource conversions
// ---------------------------------------------------------------------------

func TestRC_FileStorageResourceConversions(t *testing.T) {
	assert.Nil(t, ToResourceDomain(nil, &FileStorageModel{}))
	assert.Nil(t, ToResourceDomain(&ResourceModel{}, nil))
	r, fs := FromResourceDomain(nil)
	assert.Nil(t, r)
	assert.Nil(t, fs)

	now := time.Now()
	res := &pkgmodels.FileStorageResource{
		BaseResource: pkgmodels.BaseResource{
			ID:          uuid.NewString(),
			Type:        pkgmodels.ResourceTypeFileStorage,
			OwnerID:     uuid.NewString(),
			Name:        "store",
			Description: "d",
			Status:      pkgmodels.ResourceStatusActive,
			Metadata:    map[string]any{"k": "v"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		StorageLimitBytes: 100,
		UsedStorageBytes:  10,
		FileCount:         2,
		PricingPlanID:     uuid.NewString(),
	}
	r, fs = FromResourceDomain(res)
	assert.Equal(t, r.ID, fs.ResourceID)
	require.NotNil(t, fs.PricingPlanID)
	assert.Equal(t, *res, *ToResourceDomain(r, fs))
	generic := ToFileStorageResourceDomain(r, fs)
	assert.Equal(t, res.ID, generic.GetID())

	r, fs = FromResourceDomain(&pkgmodels.FileStorageResource{})
	assert.Equal(t, uuid.Nil, r.ID)
	assert.Equal(t, uuid.Nil, r.OwnerID)
	assert.Nil(t, r.Metadata)
	assert.Nil(t, fs.PricingPlanID)
	d := ToResourceDomain(r, fs)
	assert.Empty(t, d.PricingPlanID)
	assert.Nil(t, d.Metadata)
}

func TestRC_PricingPlanConversions(t *testing.T) {
	assert.Nil(t, ToPricingPlanDomain(nil))
	assert.Nil(t, FromPricingPlanDomain(nil))

	limit := int64(1024)
	plan := &pkgmodels.PricingPlan{
		ID:                uuid.NewString(),
		ResourceType:      pkgmodels.ResourceTypeFileStorage,
		Name:              "Free",
		Description:       "d",
		PricePerUnit:      0,
		Unit:              "month",
		StorageLimitBytes: &limit,
		BillingPeriod:     pkgmodels.BillingPeriodMonthly,
		PricingModel:      pkgmodels.PricingModelFixed,
		IsFree:            true,
		IsActive:          true,
		CreatedAt:         time.Now(),
	}
	assert.Equal(t, *plan, *ToPricingPlanDomain(FromPricingPlanDomain(plan)))
	assert.Equal(t, uuid.Nil, FromPricingPlanDomain(&pkgmodels.PricingPlan{}).ID)
}

func TestRC_CredentialsResourceConversions(t *testing.T) {
	assert.Nil(t, ToCredentialsResourceDomain(nil, &CredentialsModel{}))
	assert.Nil(t, ToCredentialsResourceDomain(&ResourceModel{}, nil))
	r, c := FromCredentialsResourceDomain(nil)
	assert.Nil(t, r)
	assert.Nil(t, c)

	now := time.Now()
	cred := &pkgmodels.CredentialsResource{
		BaseResource: pkgmodels.BaseResource{
			ID:        uuid.NewString(),
			Type:      pkgmodels.ResourceTypeCredentials,
			OwnerID:   uuid.NewString(),
			Name:      "openai",
			Status:    pkgmodels.ResourceStatusActive,
			Metadata:  map[string]any{"k": "v"},
			CreatedAt: now,
			UpdatedAt: now,
		},
		CredentialType: pkgmodels.CredentialTypeAPIKey,
		EncryptedData:  map[string]string{"api_key": "enc"},
		Provider:       "openai",
		ExpiresAt:      &now,
		LastUsedAt:     &now,
		UsageCount:     4,
		PricingPlanID:  uuid.NewString(),
	}
	r, c = FromCredentialsResourceDomain(cred)
	require.NotNil(t, c.Provider)
	assert.Equal(t, "openai", *c.Provider)
	assert.Equal(t, "enc", c.EncryptedData["api_key"])
	assert.Equal(t, *cred, *ToCredentialsResourceDomain(r, c))

	// non-string encrypted values are dropped on the way back
	c.EncryptedData["num"] = 1.0
	assert.NotContains(t, ToCredentialsResourceDomain(r, c).EncryptedData, "num")

	r, c = FromCredentialsResourceDomain(&pkgmodels.CredentialsResource{})
	assert.Equal(t, uuid.Nil, r.ID)
	assert.Nil(t, c.Provider)
	assert.Nil(t, c.PricingPlanID)
	assert.NotNil(t, c.EncryptedData)
	d := ToCredentialsResourceDomain(r, c)
	assert.Empty(t, d.Provider)
	assert.Empty(t, d.PricingPlanID)
	assert.Nil(t, d.Metadata)
}

func TestRC_RentalKeyResourceConversions(t *testing.T) {
	assert.Nil(t, ToRentalKeyResourceDomain(nil, &RentalKeyModel{}))
	assert.Nil(t, ToRentalKeyResourceDomain(&ResourceModel{}, nil))
	r, rk := FromRentalKeyResourceDomain(nil)
	assert.Nil(t, r)
	assert.Nil(t, rk)

	now := time.Now()
	daily := 100
	monthly := int64(1000)
	rental := &pkgmodels.RentalKeyResource{
		BaseResource: pkgmodels.BaseResource{
			ID:        uuid.NewString(),
			Type:      pkgmodels.ResourceTypeRentalKey,
			OwnerID:   uuid.NewString(),
			Name:      "rk",
			Status:    pkgmodels.ResourceStatusActive,
			Metadata:  map[string]any{"k": "v"},
			CreatedAt: now,
			UpdatedAt: now,
		},
		Provider:          pkgmodels.LLMProviderType("openai"),
		ProviderConfig:    map[string]any{"base_url": "x"},
		EncryptedAPIKey:   "enc",
		DailyRequestLimit: &daily,
		MonthlyTokenLimit: &monthly,
		RequestsToday:     3,
		TokensThisMonth:   30,
		LastUsageResetAt:  now,
		TotalRequests:     9,
		TotalUsage: pkgmodels.MultimodalUsage{
			PromptTokens: 1, CompletionTokens: 2,
			ImageInputTokens: 3, ImageOutputTokens: 4,
			AudioInputTokens: 5, AudioOutputTokens: 6,
			VideoInputTokens: 7, VideoOutputTokens: 8,
		},
		TotalCost:       1.5,
		LastUsedAt:      &now,
		PricingPlanID:   uuid.NewString(),
		CreatedBy:       uuid.NewString(),
		ProvisionerType: pkgmodels.ProvisionerType("manual"),
	}
	r, rk = FromRentalKeyResourceDomain(rental)
	assert.Equal(t, int64(8), rk.TotalVideoOutputTokens)
	assert.Equal(t, *rental, *ToRentalKeyResourceDomain(r, rk))

	r, rk = FromRentalKeyResourceDomain(&pkgmodels.RentalKeyResource{})
	assert.Nil(t, rk.ProviderConfig)
	assert.Nil(t, rk.PricingPlanID)
	assert.Nil(t, rk.CreatedBy)
	d := ToRentalKeyResourceDomain(r, rk)
	assert.Nil(t, d.ProviderConfig)
	assert.Empty(t, d.PricingPlanID)
	assert.Empty(t, d.CreatedBy)
	assert.Nil(t, d.Metadata)
}

func TestRC_RentalKeyUsageRecordConversions(t *testing.T) {
	assert.Nil(t, ToRentalKeyUsageRecordDomain(nil))
	assert.Nil(t, FromRentalKeyUsageRecordDomain(nil))

	rec := &pkgmodels.RentalKeyUsageRecord{
		ID:          uuid.NewString(),
		RentalKeyID: uuid.NewString(),
		Model:       "gpt",
		Usage: pkgmodels.MultimodalUsage{
			PromptTokens: 1, CompletionTokens: 2,
			ImageInputTokens: 3, ImageOutputTokens: 4,
			AudioInputTokens: 5, AudioOutputTokens: 6,
			VideoInputTokens: 7, VideoOutputTokens: 8,
		},
		EstimatedCost:  0.25,
		ExecutionID:    uuid.NewString(),
		WorkflowID:     uuid.NewString(),
		NodeID:         "node-1",
		Status:         "failed",
		ErrorMessage:   "rate limited",
		ResponseTimeMs: 120,
		CreatedAt:      time.Now(),
	}
	m := FromRentalKeyUsageRecordDomain(rec)
	assert.Equal(t, 36, m.GetTotalTokens())
	assert.Equal(t, *rec, *ToRentalKeyUsageRecordDomain(m))

	m = FromRentalKeyUsageRecordDomain(&pkgmodels.RentalKeyUsageRecord{})
	assert.Equal(t, uuid.Nil, m.ID)
	assert.Equal(t, uuid.Nil, m.RentalKeyID)
	assert.Nil(t, m.ExecutionID)
	assert.Nil(t, m.WorkflowID)
	assert.Nil(t, m.NodeID)
	assert.Nil(t, m.ErrorMessage)
	assert.Nil(t, m.ResponseTimeMs)
	d := ToRentalKeyUsageRecordDomain(m)
	assert.Empty(t, d.ExecutionID)
	assert.Empty(t, d.NodeID)
	assert.Zero(t, d.ResponseTimeMs)
}

// ---------------------------------------------------------------------------
// Custom SQL types: error branches
// ---------------------------------------------------------------------------

func TestRC_JSONBMap_ErrorBranches(t *testing.T) {
	_, err := JSONBMap{"bad": make(chan int)}.Value()
	assert.Error(t, err)

	var j JSONBMap
	assert.Error(t, j.Scan("not-bytes"))
	assert.Error(t, j.Scan([]byte("{not json")))
}

func TestRC_StringArray_ErrorBranches(t *testing.T) {
	var a StringArray
	assert.Error(t, a.Scan(42))
	assert.Error(t, a.Scan([]byte("not-an-array")))

	require.NoError(t, a.Scan([]byte("")))
	assert.Empty(t, a)
	require.NoError(t, a.Scan([]byte(`{"x","y"}`)))
	assert.Equal(t, StringArray{"x", "y"}, a)

	v, err := StringArray{`with "quote"`}.Value()
	require.NoError(t, err)
	assert.Equal(t, `{"with \"quote\""}`, v)
}

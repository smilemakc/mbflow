package models

import (
	"context"

	"github.com/uptrace/bun"
)

// insertHook and updateHook are implemented by models that fill defaults,
// timestamps and invariants before they are written.
type insertHook interface{ BeforeInsert(ctx any) error }
type updateHook interface{ BeforeUpdate(ctx any) error }

// runWriteHooks dispatches bun's BeforeAppendModel hook to the model's
// BeforeInsert/BeforeUpdate methods. bun does not call methods with those
// names itself, so without this they never ran.
func runWriteHooks(ctx context.Context, model any, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		if h, ok := model.(insertHook); ok {
			return h.BeforeInsert(ctx)
		}
	case *bun.UpdateQuery:
		if h, ok := model.(updateHook); ok {
			return h.BeforeUpdate(ctx)
		}
	}
	return nil
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (m *BillingAccountModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return runWriteHooks(ctx, m, query)
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (m *TransactionModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return runWriteHooks(ctx, m, query)
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (m *EdgeModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return runWriteHooks(ctx, m, query)
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (m *EventModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return runWriteHooks(ctx, m, query)
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (m *ExecutionModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return runWriteHooks(ctx, m, query)
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (m *FileModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return runWriteHooks(ctx, m, query)
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (m *StorageConfigModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return runWriteHooks(ctx, m, query)
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (m *NodeExecutionModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return runWriteHooks(ctx, m, query)
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (m *NodeModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return runWriteHooks(ctx, m, query)
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (m *ResourceModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return runWriteHooks(ctx, m, query)
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (m *FileStorageModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return runWriteHooks(ctx, m, query)
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (m *PricingPlanModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return runWriteHooks(ctx, m, query)
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (m *ServiceAuditLogModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return runWriteHooks(ctx, m, query)
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (m *ServiceKeyModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return runWriteHooks(ctx, m, query)
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (m *SystemKeyModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return runWriteHooks(ctx, m, query)
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (m *TriggerModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return runWriteHooks(ctx, m, query)
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (m *UserModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return runWriteHooks(ctx, m, query)
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (m *SessionModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return runWriteHooks(ctx, m, query)
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (m *RoleModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return runWriteHooks(ctx, m, query)
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (m *UserRoleModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return runWriteHooks(ctx, m, query)
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (m *AuditLogModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return runWriteHooks(ctx, m, query)
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (m *WorkflowModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return runWriteHooks(ctx, m, query)
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (m *WorkflowResourceModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return runWriteHooks(ctx, m, query)
}

var (
	_ bun.BeforeAppendModelHook = (*BillingAccountModel)(nil)
	_ bun.BeforeAppendModelHook = (*TransactionModel)(nil)
	_ bun.BeforeAppendModelHook = (*EdgeModel)(nil)
	_ bun.BeforeAppendModelHook = (*EventModel)(nil)
	_ bun.BeforeAppendModelHook = (*ExecutionModel)(nil)
	_ bun.BeforeAppendModelHook = (*FileModel)(nil)
	_ bun.BeforeAppendModelHook = (*StorageConfigModel)(nil)
	_ bun.BeforeAppendModelHook = (*NodeExecutionModel)(nil)
	_ bun.BeforeAppendModelHook = (*NodeModel)(nil)
	_ bun.BeforeAppendModelHook = (*ResourceModel)(nil)
	_ bun.BeforeAppendModelHook = (*FileStorageModel)(nil)
	_ bun.BeforeAppendModelHook = (*PricingPlanModel)(nil)
	_ bun.BeforeAppendModelHook = (*ServiceAuditLogModel)(nil)
	_ bun.BeforeAppendModelHook = (*ServiceKeyModel)(nil)
	_ bun.BeforeAppendModelHook = (*SystemKeyModel)(nil)
	_ bun.BeforeAppendModelHook = (*TriggerModel)(nil)
	_ bun.BeforeAppendModelHook = (*UserModel)(nil)
	_ bun.BeforeAppendModelHook = (*SessionModel)(nil)
	_ bun.BeforeAppendModelHook = (*RoleModel)(nil)
	_ bun.BeforeAppendModelHook = (*UserRoleModel)(nil)
	_ bun.BeforeAppendModelHook = (*AuditLogModel)(nil)
	_ bun.BeforeAppendModelHook = (*WorkflowModel)(nil)
	_ bun.BeforeAppendModelHook = (*WorkflowResourceModel)(nil)
)

package db

import (
	"context"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"time"
)

type TimeStamped struct {
	CreatedAt *time.Time `bun:",nullzero,notnull,default:current_timestamp"  pg:"default:now()" json:"created_at,omitempty" bson:"created_at"`
	UpdatedAt *time.Time `bun:",nullzero,notnull,default:current_timestamp"  pg:"updated_at" json:"updated_at,omitempty" bson:"updated_at"`
}

var _ bun.BeforeAppendModelHook = (*TimeStamped)(nil)

func (m *TimeStamped) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	now := time.Now()
	switch query.(type) {
	case *bun.InsertQuery:
		m.CreatedAt = &now
	case *bun.UpdateQuery:
		m.UpdatedAt = &now
	}
	return nil
}

type UUIDPk struct {
	ID *uuid.UUID `bun:",pk,type:uuid,default:uuid_generate_v4()" pg:",pk,type:uuid,default:uuid_generate_v4()" json:"id,omitempty"`
}

type Base struct {
	UUIDPk
	TimeStamped
}

type SoftDelete struct {
	Base
	DeletedAt *time.Time `bun:",soft_delete,nullzero" pg:",soft_delete" json:"deleted_at,omitempty" bson:"deleted_at"`
}

var _ bun.BeforeAppendModelHook = (*Base)(nil)

func (b *Base) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	now := time.Now()
	switch query.(type) {
	case *bun.InsertQuery:
		b.CreatedAt = &now
		b.UpdatedAt = &now
	case *bun.UpdateQuery:
		b.UpdatedAt = &now
	}
	return nil
}

// Delete sets deleted_at time to current_time
func (b *SoftDelete) Delete() {
	t := time.Now()
	b.DeletedAt = &t
}

type WithSort struct {
	Sort int `bun:",pk,notnull,default:0" pg:"sort,notnull,use_zero,default:0" json:"sort"`
}

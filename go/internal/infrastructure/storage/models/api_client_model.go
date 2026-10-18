package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
)

// APIClientModel represents the mbflow_api_clients table.
type APIClientModel struct {
	bun.BaseModel `bun:"table:mbflow_api_clients,alias:ac"`

	ID              uuid.UUID                      `bun:"id,pk,type:uuid,default:uuid_generate_v4()"`
	OwnerID         uuid.UUID                      `bun:"owner_id,notnull,type:uuid"`
	Name            string                         `bun:"name,notnull"`
	Description     string                         `bun:"description,nullzero"`
	Title           string                         `bun:"title,notnull"`
	APIVersion      string                         `bun:"api_version,nullzero"`
	SpecVersion     string                         `bun:"spec_version,notnull"`
	BaseURL         string                         `bun:"base_url,nullzero"`
	Servers         []pkgmodels.APIServer          `bun:"servers,type:jsonb,notnull"`
	SecuritySchemes []*pkgmodels.APISecurityScheme `bun:"security_schemes,type:jsonb,notnull"`
	Auth            pkgmodels.APIClientAuth        `bun:"auth,type:jsonb,notnull"`
	DefaultHeaders  map[string]string              `bun:"default_headers,type:jsonb,notnull"`
	Operations      []*pkgmodels.APIOperation      `bun:"operations,type:jsonb,notnull"`
	Schemas         map[string]map[string]any      `bun:"schemas,type:jsonb,notnull"`
	Warnings        []pkgmodels.APISpecWarning     `bun:"warnings,type:jsonb,notnull"`
	OperationCount  int                            `bun:"operation_count,notnull"`
	SpecRaw         string                         `bun:"spec_raw,notnull"`
	SpecSize        int                            `bun:"spec_size,notnull"`
	CreatedAt       time.Time                      `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt       time.Time                      `bun:"updated_at,notnull,default:current_timestamp"`
	DeletedAt       *time.Time                     `bun:"deleted_at"`
}

// ToAPIClientDomain converts the DB model to the domain model.
func (m *APIClientModel) ToAPIClientDomain() *pkgmodels.APIClient {
	if m == nil {
		return nil
	}
	return &pkgmodels.APIClient{
		ID:              m.ID.String(),
		OwnerID:         m.OwnerID.String(),
		Name:            m.Name,
		Description:     m.Description,
		Title:           m.Title,
		APIVersion:      m.APIVersion,
		SpecVersion:     m.SpecVersion,
		BaseURL:         m.BaseURL,
		Servers:         nonNilSlice(m.Servers),
		SecuritySchemes: nonNilSlice(m.SecuritySchemes),
		Auth:            m.Auth,
		DefaultHeaders:  nonNilMap(m.DefaultHeaders),
		Operations:      nonNilSlice(m.Operations),
		Schemas:         nonNilMap(m.Schemas),
		Warnings:        nonNilSlice(m.Warnings),
		OperationCount:  m.OperationCount,
		SpecRaw:         m.SpecRaw,
		SpecSize:        m.SpecSize,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

// FromAPIClientDomain creates the DB model from the domain model.
// JSONB columns are NOT NULL, so nil collections are stored as empty ones.
func FromAPIClientDomain(c *pkgmodels.APIClient) (*APIClientModel, error) {
	if c == nil {
		return nil, nil
	}
	ownerID, err := uuid.Parse(c.OwnerID)
	if err != nil {
		return nil, pkgmodels.ErrInvalidID
	}
	var id uuid.UUID
	if c.ID != "" {
		if id, err = uuid.Parse(c.ID); err != nil {
			return nil, pkgmodels.ErrInvalidID
		}
	}
	return &APIClientModel{
		ID:              id,
		OwnerID:         ownerID,
		Name:            c.Name,
		Description:     c.Description,
		Title:           c.Title,
		APIVersion:      c.APIVersion,
		SpecVersion:     c.SpecVersion,
		BaseURL:         c.BaseURL,
		Servers:         nonNilSlice(c.Servers),
		SecuritySchemes: nonNilSlice(c.SecuritySchemes),
		Auth:            c.Auth,
		DefaultHeaders:  nonNilMap(c.DefaultHeaders),
		Operations:      nonNilSlice(c.Operations),
		Schemas:         nonNilMap(c.Schemas),
		Warnings:        nonNilSlice(c.Warnings),
		OperationCount:  len(c.Operations),
		SpecRaw:         c.SpecRaw,
		SpecSize:        c.SpecSize,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}, nil
}

func nonNilSlice[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func nonNilMap[K comparable, V any](m map[K]V) map[K]V {
	if m == nil {
		return map[K]V{}
	}
	return m
}

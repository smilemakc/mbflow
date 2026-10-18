package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

// APIClientRepository persists API clients imported from OpenAPI specifications.
// Deleted clients are hidden from every method and reported as models.ErrAPIClientNotFound.
type APIClientRepository interface {
	// Create stores a new client including its original document.
	Create(ctx context.Context, client *models.APIClient) error

	// Update stores the client settings: name, description, base URL, auth
	// and default headers. The specification columns (document, operations,
	// schemas and metadata) are replaced only when client.SpecRaw is not empty.
	Update(ctx context.Context, client *models.APIClient) error

	// Delete marks the client as deleted.
	Delete(ctx context.Context, id uuid.UUID) error

	// FindByID returns the client with operations and schemas, without the original document.
	FindByID(ctx context.Context, id uuid.UUID) (*models.APIClient, error)

	// FindSpec returns the original document of the client.
	FindSpec(ctx context.Context, id uuid.UUID) (string, error)

	// FindUpdatedAt returns the last modification time of the client.
	FindUpdatedAt(ctx context.Context, id uuid.UUID) (time.Time, error)

	// FindByOwner lists the clients of a user, newest first, without
	// operations, schemas and the original document.
	FindByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*models.APIClient, int64, error)
}

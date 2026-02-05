package server

import (
	"net"

	"github.com/uptrace/bun"
	grpclib "google.golang.org/grpc"

	"github.com/smilemakc/mbflow/internal/application/auth"
	"github.com/smilemakc/mbflow/internal/application/engine"
	"github.com/smilemakc/mbflow/internal/application/filestorage"
	"github.com/smilemakc/mbflow/internal/application/observer"
	"github.com/smilemakc/mbflow/internal/application/rentalkey"
	"github.com/smilemakc/mbflow/internal/application/serviceapi"
	"github.com/smilemakc/mbflow/internal/application/servicekey"
	"github.com/smilemakc/mbflow/internal/application/systemkey"
	"github.com/smilemakc/mbflow/internal/application/trigger"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	serviceapigrpc "github.com/smilemakc/mbflow/internal/infrastructure/api/grpc"
	"github.com/smilemakc/mbflow/internal/infrastructure/api/rest"
	"github.com/smilemakc/mbflow/internal/infrastructure/cache"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage"
	"github.com/smilemakc/mbflow/pkg/crypto"
	"github.com/smilemakc/mbflow/pkg/executor"
)

// DataLayer holds database connections and all repositories.
type DataLayer struct {
	DB         *bun.DB
	RedisCache *cache.RedisCache

	// Repositories
	WorkflowRepo    *storage.WorkflowRepository
	ExecutionRepo   *storage.ExecutionRepository
	EventRepo       *storage.EventRepository
	TriggerRepo     repository.TriggerRepository
	UserRepo        *storage.UserRepository
	FileRepo        *storage.FileRepository
	AccountRepo     *storage.AccountRepositoryImpl
	TransactionRepo *storage.TransactionRepositoryImpl
	ResourceRepo    *storage.ResourceRepositoryImpl
	PricingPlanRepo *storage.PricingPlanRepositoryImpl
	CredentialsRepo *storage.CredentialsRepositoryImpl
	ServiceKeyRepo  *storage.ServiceKeyRepositoryImpl
	SystemKeyRepo   *storage.SystemKeyRepoImpl
	AuditLogRepo    *storage.ServiceAuditLogRepoImpl
	RentalKeyRepo   *storage.RentalKeyRepositoryImpl
}

// AuthLayer holds authentication and authorization components.
type AuthLayer struct {
	AuthService       *auth.Service
	ProviderManager   *auth.ProviderManager
	ServiceKeyService *servicekey.Service
	AuthMiddleware    *rest.AuthMiddleware
	LoginRateLimiter  *rest.LoginRateLimiter
	EncryptionService *crypto.EncryptionService
	RentalKeyProvider *rentalkey.Provider
}

// ExecutionLayer holds workflow execution components.
type ExecutionLayer struct {
	ExecutorManager  executor.Manager
	ExecutionManager *engine.ExecutionManager
	ObserverManager  *observer.ObserverManager
	WSHub            *observer.WebSocketHub
}

// ServiceAPILayer holds Service API and gRPC components.
type ServiceAPILayer struct {
	SystemKeyService     *systemkey.Service
	AuditService         *systemkey.AuditService
	SystemAuthMiddleware *rest.SystemAuthMiddleware
	AuditMiddleware      *rest.AuditMiddleware
	Operations           *serviceapi.Operations
	GRPCServer           *serviceapigrpc.ServiceAPIServer
	GRPCServerInstance   *grpclib.Server
	GRPCListener         net.Listener
}

// TriggerLayer holds trigger management components.
type TriggerLayer struct {
	TriggerManager *trigger.Manager
}

// FileStorageLayer holds file storage components.
type FileStorageLayer struct {
	FileStorageManager *filestorage.StorageManager
}

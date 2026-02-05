package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/smilemakc/mbflow/internal/application/filestorage"
	"github.com/smilemakc/mbflow/internal/application/observer"
	"github.com/smilemakc/mbflow/internal/application/rentalkey"
	"github.com/smilemakc/mbflow/internal/application/serviceapi"
	"github.com/smilemakc/mbflow/internal/infrastructure/api/rest"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage"
)

func (s *Server) setupRoutes() error {
	if s.config.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	s.router = gin.New()

	s.router.MaxMultipartMemory = s.config.Server.MaxMultipartMemory

	loggingMiddleware := rest.NewLoggingMiddleware(s.logger)
	recoveryMiddleware := rest.NewRecoveryMiddleware(s.logger)
	bodySizeMiddleware := rest.NewBodySizeMiddleware(s.logger, s.config.Server.MaxBodySize)

	s.router.Use(recoveryMiddleware.Recovery())
	s.router.Use(loggingMiddleware.RequestLogger())
	s.router.Use(bodySizeMiddleware.LimitBodySize())
	s.router.Use(gzip.Gzip(gzip.DefaultCompression))

	if s.config.Server.CORS {
		allowedOrigins := s.config.Server.CORSAllowedOrigins
		allowAll := len(allowedOrigins) == 0 && s.config.Logging.Level == "debug"

		if !allowAll && len(allowedOrigins) == 0 {
			s.logger.Warn("CORS enabled but no allowed origins configured (MBFLOW_CORS_ALLOWED_ORIGINS). Set origins or use debug log level for wildcard.")
		}

		originSet := make(map[string]struct{}, len(allowedOrigins))
		for _, o := range allowedOrigins {
			originSet[o] = struct{}{}
		}

		s.router.Use(func(c *gin.Context) {
			origin := c.GetHeader("Origin")

			if allowAll {
				c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" {
				if _, ok := originSet[origin]; ok {
					c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
					c.Writer.Header().Set("Vary", "Origin")
				}
			}

			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
			c.Writer.Header().Set("Access-Control-Max-Age", "86400")

			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}

			c.Next()
		})

		if allowAll {
			s.logger.Info("CORS enabled with wildcard origin (debug mode)")
		} else {
			s.logger.Info("CORS enabled", "allowed_origins", allowedOrigins)
		}
	}

	s.setupHealthEndpoints()
	s.setupSwaggerEndpoint()
	s.setupWebSocketEndpoints()
	s.setupAPIv1Routes()

	s.logger.Info("REST API routes registered")
	return nil
}

func (s *Server) setupHealthEndpoints() {
	s.router.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		if err := storage.Ping(ctx, s.data.DB); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  fmt.Sprintf("database: %s", err.Error()),
			})
			return
		}

		if s.data.RedisCache != nil {
			if err := s.data.RedisCache.Health(ctx); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"status": "unhealthy",
					"error":  fmt.Sprintf("redis: %s", err.Error()),
				})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	s.router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	s.router.GET("/metrics", func(c *gin.Context) {
		dbStats := storage.Stats(s.data.DB)

		metrics := gin.H{
			"database": gin.H{
				"open_connections": dbStats.OpenConnections,
				"in_use":           dbStats.InUse,
				"idle":             dbStats.Idle,
				"max_open_conns":   dbStats.MaxOpenConnections,
			},
		}

		if s.data.RedisCache != nil {
			cacheStats := s.data.RedisCache.Stats()
			metrics["redis"] = gin.H{
				"hits":        cacheStats.Hits,
				"misses":      cacheStats.Misses,
				"total_conns": cacheStats.TotalConns,
				"idle_conns":  cacheStats.IdleConns,
			}
		}

		c.JSON(http.StatusOK, gin.H{"metrics": metrics})
	})
}

func (s *Server) setupSwaggerEndpoint() {
	// Swagger UI endpoint - serves OpenAPI documentation
	// Access at /swagger/index.html
	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	s.logger.Info("Swagger documentation endpoint registered", "endpoint", "/swagger/index.html")
}

func (s *Server) setupWebSocketEndpoints() {
	if s.config.Observer.EnableWebSocket && s.execution.WSHub != nil {
		wsHandler := observer.NewWebSocketHandler(s.execution.WSHub, s.logger)
		s.router.GET("/ws/executions", func(c *gin.Context) {
			wsHandler.ServeHTTP(c.Writer, c.Request)
		})
		s.router.GET("/ws/health", func(c *gin.Context) {
			wsHandler.HandleHealthCheck(c.Writer, c.Request)
		})
		s.logger.Info("WebSocket endpoints registered",
			"endpoints", []string{"/ws/executions", "/ws/health"},
		)
	}
}

func (s *Server) setupAPIv1Routes() {
	apiV1 := s.router.Group("/api/v1")
	{
		s.setupAuthRoutes(apiV1)
		s.setupAdminRoutes(apiV1)
		s.setupWorkflowRoutes(apiV1)
		s.setupExecutionRoutes(apiV1)
		s.setupTriggerRoutes(apiV1)
		s.setupFileRoutes(apiV1)
		s.setupResourceRoutes(apiV1)
		s.setupAccountRoutes(apiV1)
		s.setupCredentialsRoutes(apiV1)
		s.setupRentalKeyRoutes(apiV1)
		s.setupServiceKeyRoutes(apiV1)
		s.setupWebhookRoutes(apiV1)
		s.setupServiceAPIRoutes(apiV1)
	}
}

func (s *Server) setupAuthRoutes(apiV1 *gin.RouterGroup) {
	authHandlers := rest.NewAuthHandlers(s.auth.AuthService, s.auth.ProviderManager, s.auth.LoginRateLimiter)

	authGroup := apiV1.Group("/auth")
	{
		authGroup.POST("/register", authHandlers.HandleRegister)
		authGroup.POST("/login", s.auth.LoginRateLimiter.Middleware(), authHandlers.HandleLogin)
		authGroup.POST("/refresh", authHandlers.HandleRefresh)
		authGroup.GET("/info", authHandlers.HandleGetAuthInfo)

		authGroup.GET("/oauth/authorize", authHandlers.HandleOAuthAuthorize)
		authGroup.GET("/oauth/callback", authHandlers.HandleOAuthCallback)

		authGroup.POST("/logout", s.auth.AuthMiddleware.RequireAuth(), authHandlers.HandleLogout)
		authGroup.GET("/me", s.auth.AuthMiddleware.RequireAuth(), authHandlers.HandleGetMe)
		authGroup.POST("/password", s.auth.AuthMiddleware.RequireAuth(), authHandlers.HandleChangePassword)
	}

	s.logger.Info("Auth endpoints registered")
}

func (s *Server) setupAdminRoutes(apiV1 *gin.RouterGroup) {
	authHandlers := rest.NewAuthHandlers(s.auth.AuthService, s.auth.ProviderManager, s.auth.LoginRateLimiter)

	adminGroup := apiV1.Group("/admin")
	adminGroup.Use(s.auth.AuthMiddleware.RequireAdmin())
	{
		adminGroup.GET("/users", authHandlers.HandleAdminListUsers)
		adminGroup.POST("/users", authHandlers.HandleAdminCreateUser)
		adminGroup.GET("/users/:id", authHandlers.HandleAdminGetUser)
		adminGroup.PUT("/users/:id", authHandlers.HandleAdminUpdateUser)
		adminGroup.DELETE("/users/:id", authHandlers.HandleAdminDeleteUser)
		adminGroup.POST("/users/:id/reset-password", authHandlers.HandleAdminResetPassword)

		adminGroup.GET("/roles", authHandlers.HandleListRoles)
		adminGroup.GET("/users/:id/roles", authHandlers.HandleGetUserRoles)
		adminGroup.POST("/users/:id/roles", authHandlers.HandleAssignRole)
		adminGroup.DELETE("/users/:id/roles/:role_id", authHandlers.HandleRemoveRole)
	}
}

func (s *Server) setupWorkflowRoutes(apiV1 *gin.RouterGroup) {
	ops := &serviceapi.Operations{
		WorkflowRepo:    s.data.WorkflowRepo,
		ExecutionRepo:   s.data.ExecutionRepo,
		TriggerRepo:     s.data.TriggerRepo,
		CredentialsRepo: s.data.CredentialsRepo,
		ExecutionMgr:    s.execution.ExecutionManager,
		ExecutorManager: s.execution.ExecutorManager,
		EncryptionSvc:   s.auth.EncryptionService,
		AuditService:    s.serviceAPI.AuditService,
		Logger:          s.logger,
	}

	workflowHandlers := rest.NewWorkflowHandlers(ops, s.logger)
	nodeHandlers := rest.NewNodeHandlers(s.data.WorkflowRepo, s.logger)
	edgeHandlers := rest.NewEdgeHandlers(s.data.WorkflowRepo, s.logger)
	executionHandlers := rest.NewExecutionHandlers(ops, s.logger)
	importHandlers := rest.NewImportHandlers(s.data.WorkflowRepo, s.data.TriggerRepo, s.logger, s.execution.ExecutorManager)

	workflows := apiV1.Group("/workflows")
	workflows.Use(s.auth.AuthMiddleware.OptionalAuth())
	{
		workflows.POST("", workflowHandlers.HandleCreateWorkflow)
		workflows.GET("", workflowHandlers.HandleListWorkflows)
		workflows.GET("/:workflow_id", workflowHandlers.HandleGetWorkflow)
		workflows.PUT("/:workflow_id", workflowHandlers.HandleUpdateWorkflow)
		workflows.POST("/:workflow_id/execute", executionHandlers.HandleRunExecution)
		workflows.DELETE("/:workflow_id", workflowHandlers.HandleDeleteWorkflow)
		workflows.POST("/:workflow_id/publish", workflowHandlers.HandlePublishWorkflow)
		workflows.POST("/:workflow_id/unpublish", workflowHandlers.HandleUnpublishWorkflow)
		workflows.GET("/:workflow_id/diagram", workflowHandlers.HandleGetWorkflowDiagram)

		workflows.POST("/:workflow_id/resources", workflowHandlers.AttachWorkflowResource)
		workflows.GET("/:workflow_id/resources", workflowHandlers.GetWorkflowResources)
		workflows.PUT("/:workflow_id/resources/:resource_id", workflowHandlers.UpdateWorkflowResourceAlias)
		workflows.DELETE("/:workflow_id/resources/:resource_id", workflowHandlers.DetachWorkflowResource)

		workflows.POST("/:workflow_id/nodes", nodeHandlers.HandleAddNode)
		workflows.GET("/:workflow_id/nodes", nodeHandlers.HandleListNodes)
		workflows.GET("/:workflow_id/nodes/:node_id", nodeHandlers.HandleGetNode)
		workflows.PUT("/:workflow_id/nodes/:node_id", nodeHandlers.HandleUpdateNode)
		workflows.DELETE("/:workflow_id/nodes/:node_id", nodeHandlers.HandleDeleteNode)

		workflows.POST("/:workflow_id/edges", edgeHandlers.HandleAddEdge)
		workflows.GET("/:workflow_id/edges", edgeHandlers.HandleListEdges)
		workflows.GET("/:workflow_id/edges/:edge_id", edgeHandlers.HandleGetEdge)
		workflows.PUT("/:workflow_id/edges/:edge_id", edgeHandlers.HandleUpdateEdge)
		workflows.DELETE("/:workflow_id/edges/:edge_id", edgeHandlers.HandleDeleteEdge)

		workflows.POST("/import", importHandlers.HandleImportWorkflow)
		workflows.GET("/import/types", importHandlers.HandleGetSupportedTypes)
		workflows.GET("/:workflow_id/export", importHandlers.HandleExportWorkflow)
	}
}

func (s *Server) setupExecutionRoutes(apiV1 *gin.RouterGroup) {
	ops := &serviceapi.Operations{
		WorkflowRepo:    s.data.WorkflowRepo,
		ExecutionRepo:   s.data.ExecutionRepo,
		TriggerRepo:     s.data.TriggerRepo,
		CredentialsRepo: s.data.CredentialsRepo,
		ExecutionMgr:    s.execution.ExecutionManager,
		ExecutorManager: s.execution.ExecutorManager,
		EncryptionSvc:   s.auth.EncryptionService,
		AuditService:    s.serviceAPI.AuditService,
		Logger:          s.logger,
	}

	executionHandlers := rest.NewExecutionHandlers(ops, s.logger)

	executions := apiV1.Group("/executions")
	{
		executions.POST("/run/:workflow_id", executionHandlers.HandleRunExecution)
		executions.GET("", executionHandlers.HandleListExecutions)
		executions.GET("/:id", executionHandlers.HandleGetExecution)
		executions.GET("/:id/logs", executionHandlers.HandleGetLogs)
		executions.GET("/:id/nodes/:node_id/result", executionHandlers.HandleGetNodeResult)
		executions.POST("/:id/cancel", executionHandlers.HandleCancelExecution)
		executions.POST("/:id/retry", executionHandlers.HandleRetryExecution)
		executions.GET("/:id/watch", executionHandlers.HandleWatchExecution)
		executions.GET("/:id/stream", executionHandlers.HandleStreamLogs)
	}
}

func (s *Server) setupTriggerRoutes(apiV1 *gin.RouterGroup) {
	ops := &serviceapi.Operations{
		WorkflowRepo:    s.data.WorkflowRepo,
		ExecutionRepo:   s.data.ExecutionRepo,
		TriggerRepo:     s.data.TriggerRepo,
		CredentialsRepo: s.data.CredentialsRepo,
		ExecutionMgr:    s.execution.ExecutionManager,
		ExecutorManager: s.execution.ExecutorManager,
		EncryptionSvc:   s.auth.EncryptionService,
		AuditService:    s.serviceAPI.AuditService,
		Logger:          s.logger,
	}

	triggerHandlers := rest.NewTriggerHandlers(ops, s.logger)

	triggers := apiV1.Group("/triggers")
	{
		triggers.POST("", triggerHandlers.HandleCreateTrigger)
		triggers.GET("", triggerHandlers.HandleListTriggers)
		triggers.GET("/:id", triggerHandlers.HandleGetTrigger)
		triggers.PUT("/:id", triggerHandlers.HandleUpdateTrigger)
		triggers.DELETE("/:id", triggerHandlers.HandleDeleteTrigger)
		triggers.POST("/:id/enable", triggerHandlers.HandleEnableTrigger)
		triggers.POST("/:id/disable", triggerHandlers.HandleDisableTrigger)
		triggers.POST("/:id/execute", triggerHandlers.HandleTriggerManual)
	}
}

func (s *Server) setupFileRoutes(apiV1 *gin.RouterGroup) {
	fileHandlers := rest.NewFileHandlers(s.data.FileRepo, s.fileStorage.FileStorageManager, s.logger)

	files := apiV1.Group("/files")
	files.Use(s.auth.AuthMiddleware.OptionalAuth())
	{
		files.POST("", fileHandlers.HandleUploadFile)
		files.GET("", fileHandlers.HandleListFiles)
		files.GET("/:id", fileHandlers.HandleGetFile)
		files.GET("/:id/metadata", fileHandlers.HandleGetFileMetadata)
		files.DELETE("/:id", fileHandlers.HandleDeleteFile)
		files.GET("/storage/:storage_id/usage", fileHandlers.HandleGetStorageUsage)
	}
}

func (s *Server) setupResourceRoutes(apiV1 *gin.RouterGroup) {
	resourceHandlers := rest.NewResourceHandlers(s.data.ResourceRepo, s.data.PricingPlanRepo, s.data.WorkflowRepo, s.logger)

	resourceFileService := filestorage.NewResourceFileService(
		s.data.DB,
		s.data.ResourceRepo,
		s.data.FileRepo,
		s.fileStorage.FileStorageManager,
		s.config.FileStorage.MaxFileSize,
	)
	fileStorageHandlers := rest.NewFileStorageHandlers(s.data.ResourceRepo, resourceFileService, s.logger)

	resources := apiV1.Group("/resources")
	resources.Use(s.auth.AuthMiddleware.RequireAuth())
	{
		resources.POST("/file-storage", resourceHandlers.CreateFileStorage)
		resources.GET("", resourceHandlers.ListResources)
		resources.GET("/:id", resourceHandlers.GetResource)
		resources.PUT("/:id", resourceHandlers.UpdateResource)
		resources.DELETE("/:id", resourceHandlers.DeleteResource)
		resources.GET("/pricing-plans", resourceHandlers.ListPricingPlans)

		resources.POST("/:id/files", fileStorageHandlers.UploadFile)
		resources.GET("/:id/files", fileStorageHandlers.ListFiles)
		resources.GET("/:id/files/:file_id", fileStorageHandlers.GetFileMetadata)
		resources.GET("/:id/files/:file_id/download", fileStorageHandlers.DownloadFile)
		resources.DELETE("/:id/files/:file_id", fileStorageHandlers.DeleteFile)
	}
}

func (s *Server) setupAccountRoutes(apiV1 *gin.RouterGroup) {
	accountHandlers := rest.NewAccountHandlers(s.data.AccountRepo, s.data.TransactionRepo, s.logger)

	account := apiV1.Group("/account")
	account.Use(s.auth.AuthMiddleware.RequireAuth())
	{
		account.GET("", accountHandlers.GetAccount)
		account.POST("/deposit", accountHandlers.Deposit)
		account.GET("/transactions", accountHandlers.ListTransactions)
		account.GET("/transactions/:id", accountHandlers.GetTransaction)
	}
}

func (s *Server) setupCredentialsRoutes(apiV1 *gin.RouterGroup) {
	if s.auth.EncryptionService == nil {
		s.logger.Warn("Credentials endpoints disabled - encryption key not configured")
		return
	}

	credentialsHandlers := rest.NewCredentialsHandlers(s.data.CredentialsRepo, s.data.WorkflowRepo, s.auth.EncryptionService, s.logger)

	credentials := apiV1.Group("/credentials")
	credentials.Use(s.auth.AuthMiddleware.RequireAuth())
	{
		credentials.POST("/api-key", credentialsHandlers.CreateAPIKey)
		credentials.POST("/basic-auth", credentialsHandlers.CreateBasicAuth)
		credentials.POST("/oauth2", credentialsHandlers.CreateOAuth2)
		credentials.POST("/service-account", credentialsHandlers.CreateServiceAccount)
		credentials.POST("/custom", credentialsHandlers.CreateCustom)

		credentials.GET("", credentialsHandlers.ListCredentials)
		credentials.GET("/:id", credentialsHandlers.GetCredential)
		credentials.GET("/:id/secrets", credentialsHandlers.GetCredentialSecrets)
		credentials.PUT("/:id", credentialsHandlers.UpdateCredential)
		credentials.DELETE("/:id", credentialsHandlers.DeleteCredential)
	}

	s.logger.Info("Credentials endpoints registered")
}

func (s *Server) setupRentalKeyRoutes(apiV1 *gin.RouterGroup) {
	if s.auth.RentalKeyProvider == nil {
		s.logger.Warn("Rental Keys endpoints disabled - encryption key not configured")
		return
	}

	rentalKeyHandlers := rest.NewRentalKeyHandlers(s.auth.RentalKeyProvider, s.logger)
	rentalKeyAdminHandlers := rest.NewRentalKeyAdminHandlers(rentalkey.NewAdminService(s.data.RentalKeyRepo, s.auth.EncryptionService), s.logger)

	rentalKeys := apiV1.Group("/rental-keys")
	rentalKeys.Use(s.auth.AuthMiddleware.RequireAuth())
	{
		rentalKeys.GET("", rentalKeyHandlers.ListRentalKeys)
		rentalKeys.GET("/:id", rentalKeyHandlers.GetRentalKey)
		rentalKeys.GET("/:id/usage", rentalKeyHandlers.GetRentalKeyUsage)
		rentalKeys.GET("/:id/summary", rentalKeyHandlers.GetRentalKeyUsageSummary)
	}

	adminRentalKeys := apiV1.Group("/admin/rental-keys")
	adminRentalKeys.Use(s.auth.AuthMiddleware.RequireAuth())
	adminRentalKeys.Use(s.auth.AuthMiddleware.RequireAdmin())
	{
		adminRentalKeys.POST("", rentalKeyAdminHandlers.CreateRentalKey)
		adminRentalKeys.GET("", rentalKeyAdminHandlers.ListAllRentalKeys)
		adminRentalKeys.GET("/:id", rentalKeyAdminHandlers.GetRentalKey)
		adminRentalKeys.PUT("/:id", rentalKeyAdminHandlers.UpdateRentalKey)
		adminRentalKeys.DELETE("/:id", rentalKeyAdminHandlers.DeleteRentalKey)
		adminRentalKeys.POST("/:id/rotate-key", rentalKeyAdminHandlers.RotateRentalKeyAPIKey)
		adminRentalKeys.POST("/reset-daily", rentalKeyAdminHandlers.ResetDailyUsage)
		adminRentalKeys.POST("/reset-monthly", rentalKeyAdminHandlers.ResetMonthlyUsage)
	}

	s.logger.Info("Rental Keys endpoints registered")
}

func (s *Server) setupServiceKeyRoutes(apiV1 *gin.RouterGroup) {
	serviceKeyHandlers := rest.NewServiceKeyHandlers(s.auth.ServiceKeyService, s.logger)
	serviceKeyAdminHandlers := rest.NewServiceKeyAdminHandlers(s.auth.ServiceKeyService, s.logger)

	serviceKeys := apiV1.Group("/service-keys")
	serviceKeys.Use(s.auth.AuthMiddleware.RequireAuth())
	{
		serviceKeys.GET("", serviceKeyHandlers.ListMyServiceKeys)
		serviceKeys.GET("/:id", serviceKeyHandlers.GetMyServiceKey)
	}

	adminServiceKeys := apiV1.Group("/admin/service-keys")
	adminServiceKeys.Use(s.auth.AuthMiddleware.RequireAdmin())
	{
		adminServiceKeys.POST("", serviceKeyAdminHandlers.CreateServiceKey)
		adminServiceKeys.GET("", serviceKeyAdminHandlers.ListServiceKeys)
		adminServiceKeys.GET("/:id", serviceKeyAdminHandlers.GetServiceKey)
		adminServiceKeys.DELETE("/:id", serviceKeyAdminHandlers.DeleteServiceKey)
		adminServiceKeys.POST("/:id/revoke", serviceKeyAdminHandlers.RevokeServiceKey)
	}

	s.logger.Info("Service Keys endpoints registered")
}

func (s *Server) setupWebhookRoutes(apiV1 *gin.RouterGroup) {
	if s.triggers.TriggerManager == nil {
		return
	}

	webhookHandlers := rest.NewWebhookHandlers(s.triggers.TriggerManager.WebhookRegistry(), s.logger)
	apiV1.POST("/webhooks/:path", webhookHandlers.HandleWebhook)
	apiV1.GET("/webhooks/:path", webhookHandlers.HandleWebhookGet)

	telegramWebhookHandlers := rest.NewTelegramWebhookHandlers(s.triggers.TriggerManager.WebhookRegistry(), s.logger)
	apiV1.POST("/webhooks/telegram/:trigger_id", telegramWebhookHandlers.HandleTelegramWebhook)

	s.logger.Info("Webhook endpoints registered",
		"endpoints", []string{"/api/v1/webhooks/:path", "/api/v1/webhooks/telegram/:trigger_id"},
	)
}

func (s *Server) setupServiceAPIRoutes(apiV1 *gin.RouterGroup) {
	systemKeyHandlers := rest.NewServiceAPISystemKeyHandlers(s.serviceAPI.SystemKeyService, s.logger)
	adminSystemKeys := apiV1.Group("/service/system-keys")
	adminSystemKeys.Use(s.auth.AuthMiddleware.RequireAdmin())
	{
		adminSystemKeys.POST("", systemKeyHandlers.CreateSystemKey)
		adminSystemKeys.GET("", systemKeyHandlers.ListSystemKeys)
		adminSystemKeys.GET("/:id", systemKeyHandlers.GetSystemKey)
		adminSystemKeys.DELETE("/:id", systemKeyHandlers.DeleteSystemKey)
		adminSystemKeys.POST("/:id/revoke", systemKeyHandlers.RevokeSystemKey)
	}

	serviceAPI := apiV1.Group("/service")
	serviceAPI.Use(s.serviceAPI.SystemAuthMiddleware.RequireSystemAccess())
	serviceAPI.Use(s.serviceAPI.SystemAuthMiddleware.HandleImpersonation())
	serviceAPI.Use(s.serviceAPI.AuditMiddleware.RecordAction())
	{
		ops := &serviceapi.Operations{
			WorkflowRepo:    s.data.WorkflowRepo,
			ExecutionRepo:   s.data.ExecutionRepo,
			TriggerRepo:     s.data.TriggerRepo,
			CredentialsRepo: s.data.CredentialsRepo,
			ExecutionMgr:    s.execution.ExecutionManager,
			ExecutorManager: s.execution.ExecutorManager,
			EncryptionSvc:   s.auth.EncryptionService,
			AuditService:    s.serviceAPI.AuditService,
			Logger:          s.logger,
		}

		wfh := rest.NewServiceAPIWorkflowHandlers(ops)
		serviceAPI.GET("/workflows", wfh.ListWorkflows)
		serviceAPI.GET("/workflows/:id", wfh.GetWorkflow)
		serviceAPI.POST("/workflows", wfh.CreateWorkflow)
		serviceAPI.PUT("/workflows/:id", wfh.UpdateWorkflow)
		serviceAPI.DELETE("/workflows/:id", wfh.DeleteWorkflow)

		exh := rest.NewServiceAPIExecutionHandlers(ops)
		serviceAPI.GET("/executions", exh.ListExecutions)
		serviceAPI.GET("/executions/:id", exh.GetExecution)
		serviceAPI.POST("/workflows/:id/execute", exh.StartExecution)
		serviceAPI.POST("/executions/:id/cancel", exh.CancelExecution)
		serviceAPI.POST("/executions/:id/retry", exh.RetryExecution)

		trh := rest.NewServiceAPITriggerHandlers(ops)
		serviceAPI.GET("/triggers", trh.ListTriggers)
		serviceAPI.POST("/triggers", trh.CreateTrigger)
		serviceAPI.PUT("/triggers/:id", trh.UpdateTrigger)
		serviceAPI.DELETE("/triggers/:id", trh.DeleteTrigger)

		crh := rest.NewServiceAPICredentialHandlers(ops)
		serviceAPI.GET("/credentials", crh.ListCredentials)
		serviceAPI.POST("/credentials", crh.CreateCredential)
		serviceAPI.PUT("/credentials/:id", crh.UpdateCredential)
		serviceAPI.DELETE("/credentials/:id", crh.DeleteCredential)

		auh := rest.NewServiceAPIAuditHandlers(ops)
		serviceAPI.GET("/audit-log", auh.ListAuditLog)
	}

	s.logger.Info("Service API endpoints registered")
}

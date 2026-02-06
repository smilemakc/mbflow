// Package swagger provides OpenAPI documentation for the MBFlow API.
//
//	@title						MBFlow API
//	@version					1.0
//	@description				MBFlow is a workflow orchestration engine for building and running automated workflows.
//	@termsOfService				https://github.com/smilemakc/mbflow
//
//	@contact.name				MBFlow Support
//	@contact.url				https://github.com/smilemakc/mbflow/issues
//	@contact.email				support@mbflow.io
//
//	@license.name				MIT
//	@license.url				https://opensource.org/licenses/MIT
//
//	@host						localhost:8080
//	@BasePath					/api/v1
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				JWT Bearer token authentication. Format: "Bearer {token}"
//
//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						X-API-Key
//	@description				API Key authentication for service-to-service calls
//
//	@tag.name					workflows
//	@tag.description			Workflow management operations
//
//	@tag.name					executions
//	@tag.description			Workflow execution operations
//
//	@tag.name					triggers
//	@tag.description			Trigger management operations
//
//	@tag.name					nodes
//	@tag.description			Node management within workflows
//
//	@tag.name					edges
//	@tag.description			Edge (connection) management within workflows
//
//	@tag.name					auth
//	@tag.description			Authentication operations
//
//	@tag.name					resources
//	@tag.description			Resource management (credentials, files, etc.)
//
//	@tag.name					service-api
//	@tag.description			Service API operations for programmatic access
package swagger

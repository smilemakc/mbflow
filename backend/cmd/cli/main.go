// MBFlow CLI - Command-line tool for workflow management
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/smilemakc/mbflow/internal/application/auth"
	"github.com/smilemakc/mbflow/internal/config"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/pkg/sdk"
	"github.com/smilemakc/mbflow/pkg/visualization"
	"golang.org/x/term"
)

const (
	version = "1.0.0"
	usage   = `MBFlow CLI - Workflow management tool

USAGE:
    mbflow-cli <command> [options]

COMMANDS:
    workflow show <id>    Show workflow diagram
    workflow list         List all workflows
    user create           Create user (local or via auth-gateway)
    admin create          Create admin user (requires DATABASE_URL)
    version               Show version information
    help                  Show this help message

WORKFLOW SHOW OPTIONS:
    -format <format>      Output format: mermaid, ascii (default: mermaid)
    -direction <dir>      Diagram direction: TB, LR, RL, BT, elk (default: TB)
    -config               Show node configuration details (default: true)
    -conditions           Show edge conditions (default: true)
    -compact              Compact mode for ASCII (default: false)
    -color                Use colors in ASCII (default: true)
    -output <file>        Save to file instead of stdout

USER CREATE OPTIONS:
    -email <email>        User email address (required)
    -username <name>      Username (required)
    -password <pass>      Password (will prompt if not provided)
    -full-name <name>     Full name (optional)
    -phone <phone>        Phone number (optional)
    -admin                Create as admin user (default: false)
    -gateway              Create user via auth-gateway gRPC (requires MBFLOW_AUTH_GRPC_ADDRESS)
    -local                Create user in local database (requires DATABASE_URL)

ADMIN CREATE OPTIONS:
    -email <email>        Admin email address (required)
    -username <name>      Admin username (required)
    -password <pass>      Admin password (will prompt if not provided)
    -full-name <name>     Admin full name (optional)

CONNECTION OPTIONS:
    -endpoint <url>       MBFlow server endpoint (default: http://localhost:8585)
    -api-key <key>        API key for authentication
    -timeout <duration>   Request timeout (default: 30s)

EXAMPLES:
    # Show workflow as Mermaid diagram
    mbflow-cli workflow show wf-123 -format mermaid

    # Show workflow as ASCII tree with ELK layout
    mbflow-cli workflow show wf-123 -format ascii -compact

    # Save Mermaid diagram to file with ELK layout
    mbflow-cli workflow show wf-123 -format mermaid -direction elk -output diagram.mmd

    # List all workflows
    mbflow-cli workflow list

    # Create user in local database
    mbflow-cli user create -email user@example.com -username user -local

    # Create user via auth-gateway
    mbflow-cli user create -email user@example.com -username user -gateway

    # Create admin user via auth-gateway
    mbflow-cli user create -email admin@example.com -username admin -admin -gateway

    # Create admin user (interactive password prompt)
    mbflow-cli admin create -email admin@example.com -username admin

    # Create admin user with password
    mbflow-cli admin create -email admin@example.com -username admin -password SecurePass123!

ENVIRONMENT VARIABLES:
    MBFLOW_ENDPOINT       Server endpoint (overridden by -endpoint)
    MBFLOW_API_KEY        API key (overridden by -api-key)
    DATABASE_URL          Database connection string for local user creation
    MBFLOW_AUTH_GRPC_ADDRESS     Auth-gateway gRPC address (e.g., localhost:50051)
`
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	godotenv.Load()

	command := os.Args[1]

	switch command {
	case "workflow":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Error: workflow command requires a subcommand (show, list)")
			fmt.Fprint(os.Stderr, usage)
			os.Exit(1)
		}
		subcommand := os.Args[2]
		switch subcommand {
		case "show":
			handleWorkflowShow(os.Args[3:])
		case "list":
			handleWorkflowList(os.Args[3:])
		default:
			fmt.Fprintf(os.Stderr, "Error: unknown workflow subcommand: %s\n", subcommand)
			os.Exit(1)
		}

	case "user":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Error: user command requires a subcommand (create)")
			fmt.Fprint(os.Stderr, usage)
			os.Exit(1)
		}
		subcommand := os.Args[2]
		switch subcommand {
		case "create":
			handleUserCreate(os.Args[3:])
		default:
			fmt.Fprintf(os.Stderr, "Error: unknown user subcommand: %s\n", subcommand)
			os.Exit(1)
		}

	case "admin":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Error: admin command requires a subcommand (create)")
			fmt.Fprint(os.Stderr, usage)
			os.Exit(1)
		}
		subcommand := os.Args[2]
		switch subcommand {
		case "create":
			handleAdminCreate(os.Args[3:])
		default:
			fmt.Fprintf(os.Stderr, "Error: unknown admin subcommand: %s\n", subcommand)
			os.Exit(1)
		}

	case "version":
		fmt.Printf("MBFlow CLI v%s\n", version)

	case "help", "-h", "--help":
		fmt.Print(usage)

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command: %s\n", command)
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}
}

func handleWorkflowShow(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: workflow show requires a workflow ID")
		os.Exit(1)
	}

	workflowID := args[0]

	// Parse flags
	fs := flag.NewFlagSet("workflow show", flag.ExitOnError)
	format := fs.String("format", "mermaid", "Output format: mermaid, ascii")
	direction := fs.String("direction", "TB", "Diagram direction: TB, LR, RL, BT, elk")
	showConfig := fs.Bool("config", true, "Show node configuration details")
	showConditions := fs.Bool("conditions", true, "Show edge conditions")
	compact := fs.Bool("compact", false, "Compact mode for ASCII")
	useColor := fs.Bool("color", true, "Use colors in ASCII")
	output := fs.String("output", "", "Save to file instead of stdout")
	endpoint := fs.String("endpoint", getEnv("MBFLOW_ENDPOINT", "http://localhost:8585"), "MBFlow server endpoint")
	apiKey := fs.String("api-key", getEnv("MBFLOW_API_KEY", ""), "API key for authentication")
	timeout := fs.Duration("timeout", 30*time.Second, "Request timeout")

	if err := fs.Parse(args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Validate format
	*format = strings.ToLower(*format)
	if *format != "mermaid" && *format != "ascii" {
		fmt.Fprintf(os.Stderr, "Error: invalid format '%s' (must be mermaid or ascii)\n", *format)
		os.Exit(1)
	}

	// Create SDK client
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	clientOpts := []sdk.ClientOption{
		sdk.WithHTTPEndpoint(*endpoint),
	}
	if *apiKey != "" {
		clientOpts = append(clientOpts, sdk.WithAPIKey(*apiKey))
	}

	client, err := sdk.NewClient(clientOpts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create client: %v\n", err)
		os.Exit(1)
	}

	// Get workflow from server
	workflow, err := client.Workflows().Get(ctx, workflowID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get workflow '%s': %v\n", workflowID, err)
		os.Exit(1)
	}

	// Prepare render options
	opts := &visualization.RenderOptions{
		ShowConfig:     *showConfig,
		ShowConditions: *showConditions,
		Direction:      *direction,
		CompactMode:    *compact,
		UseColor:       *useColor && *output == "", // Only use color for stdout
	}

	// Render diagram
	diagram, err := visualization.RenderWorkflow(workflow, *format, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to render workflow: %v\n", err)
		os.Exit(1)
	}

	// Output diagram
	if *output != "" {
		if err := os.WriteFile(*output, []byte(diagram), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to write to file '%s': %v\n", *output, err)
			os.Exit(1)
		}
		fmt.Printf("Diagram saved to %s\n", *output)
	} else {
		fmt.Println(diagram)
	}
}

func handleWorkflowList(args []string) {
	// Parse flags
	fs := flag.NewFlagSet("workflow list", flag.ExitOnError)
	endpoint := fs.String("endpoint", getEnv("MBFLOW_ENDPOINT", "http://localhost:8585"), "MBFlow server endpoint")
	apiKey := fs.String("api-key", getEnv("MBFLOW_API_KEY", ""), "API key for authentication")
	timeout := fs.Duration("timeout", 30*time.Second, "Request timeout")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Create SDK client
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	clientOpts := []sdk.ClientOption{
		sdk.WithHTTPEndpoint(*endpoint),
	}
	if *apiKey != "" {
		clientOpts = append(clientOpts, sdk.WithAPIKey(*apiKey))
	}

	client, err := sdk.NewClient(clientOpts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create client: %v\n", err)
		os.Exit(1)
	}

	// List workflows
	workflows, err := client.Workflows().List(ctx, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to list workflows: %v\n", err)
		os.Exit(1)
	}

	if len(workflows) == 0 {
		fmt.Println("No workflows found")
		return
	}

	// Print workflows
	fmt.Printf("Found %d workflow(s):\n\n", len(workflows))
	for _, wf := range workflows {
		fmt.Printf("ID:          %s\n", wf.ID)
		fmt.Printf("Name:        %s\n", wf.Name)
		if wf.Description != "" {
			fmt.Printf("Description: %s\n", wf.Description)
		}
		fmt.Printf("Status:      %s\n", wf.Status)
		fmt.Printf("Nodes:       %d\n", len(wf.Nodes))
		fmt.Printf("Edges:       %d\n", len(wf.Edges))
		if len(wf.Tags) > 0 {
			fmt.Printf("Tags:        %s\n", strings.Join(wf.Tags, ", "))
		}
		fmt.Println("---")
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func handleAdminCreate(args []string) {
	// Parse flags
	fs := flag.NewFlagSet("admin create", flag.ExitOnError)
	email := fs.String("email", "", "Admin email address (required)")
	username := fs.String("username", "", "Admin username (required)")
	password := fs.String("password", "", "Admin password (will prompt if not provided)")
	fullName := fs.String("full-name", "", "Admin full name (optional)")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Validate required fields
	if *email == "" {
		fmt.Fprintln(os.Stderr, "Error: -email is required")
		os.Exit(1)
	}
	if *username == "" {
		fmt.Fprintln(os.Stderr, "Error: -username is required")
		os.Exit(1)
	}

	// Get password if not provided
	if *password == "" {
		*password = promptPassword("Enter admin password: ")
		if *password == "" {
			fmt.Fprintln(os.Stderr, "Error: password cannot be empty")
			os.Exit(1)
		}

		// Confirm password
		confirm := promptPassword("Confirm password: ")
		if *password != confirm {
			fmt.Fprintln(os.Stderr, "Error: passwords do not match")
			os.Exit(1)
		}
	}

	// Get database URL from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		fmt.Fprintln(os.Stderr, "Error: DATABASE_URL environment variable is required")
		fmt.Fprintln(os.Stderr, "Example: DATABASE_URL=\"postgres://user:pass@localhost:5432/mbflow?sslmode=disable\"")
		os.Exit(1)
	}

	// Connect to database
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbConfig := storage.DefaultConfig()
	dbConfig.DSN = databaseURL

	db, err := storage.NewDB(dbConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer storage.Close(db)

	// Create user repository
	userRepo := storage.NewUserRepository(db)

	// Check if user already exists
	existingUser, _ := userRepo.FindByEmail(ctx, *email)
	if existingUser != nil {
		fmt.Fprintf(os.Stderr, "Error: user with email '%s' already exists\n", *email)
		os.Exit(1)
	}

	existingUser, _ = userRepo.FindByUsername(ctx, *username)
	if existingUser != nil {
		fmt.Fprintf(os.Stderr, "Error: user with username '%s' already exists\n", *username)
		os.Exit(1)
	}

	// Create auth config for password service
	authCfg := &config.AuthConfig{
		MinPasswordLength: 8,
	}
	passwordService := auth.NewPasswordService(authCfg.MinPasswordLength)

	// Validate password
	if err := passwordService.ValidatePassword(*password); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Hash password
	passwordHash, err := passwordService.HashPassword(*password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to hash password: %v\n", err)
		os.Exit(1)
	}

	// Create admin user
	user := &models.UserModel{
		ID:           uuid.New(),
		Email:        *email,
		Username:     *username,
		PasswordHash: passwordHash,
		FullName:     *fullName,
		IsActive:     true,
		IsAdmin:      true,
	}

	if err := userRepo.Create(ctx, user); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create admin user: %v\n", err)
		os.Exit(1)
	}

	// Assign admin role if exists
	adminRole, err := userRepo.FindRoleByName(ctx, "admin")
	if err == nil && adminRole != nil {
		if err := userRepo.AssignRole(ctx, user.ID, adminRole.ID, nil); err != nil {
			fmt.Printf("Warning: failed to assign admin role: %v\n", err)
		}
	}

	fmt.Println("Admin user created successfully!")
	fmt.Printf("  ID:       %s\n", user.ID)
	fmt.Printf("  Email:    %s\n", user.Email)
	fmt.Printf("  Username: %s\n", user.Username)
	if user.FullName != "" {
		fmt.Printf("  Name:     %s\n", user.FullName)
	}
	fmt.Printf("  Admin:    true\n")
}

func handleUserCreate(args []string) {
	// Parse flags
	fs := flag.NewFlagSet("user create", flag.ExitOnError)
	email := fs.String("email", "", "User email address (required)")
	username := fs.String("username", "", "Username (required)")
	password := fs.String("password", "", "Password (will prompt if not provided)")
	fullName := fs.String("full-name", "", "Full name (optional)")
	phone := fs.String("phone", "", "Phone number (optional)")
	isAdmin := fs.Bool("admin", false, "Create as admin user")
	useGateway := fs.Bool("gateway", false, "Create user via auth-gateway gRPC")
	useLocal := fs.Bool("local", false, "Create user in local database")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Validate required fields
	if *email == "" {
		fmt.Fprintln(os.Stderr, "Error: -email is required")
		os.Exit(1)
	}
	if *username == "" {
		fmt.Fprintln(os.Stderr, "Error: -username is required")
		os.Exit(1)
	}

	// Must specify either -gateway or -local
	if !*useGateway && !*useLocal {
		fmt.Fprintln(os.Stderr, "Error: must specify either -gateway or -local")
		fmt.Fprintln(os.Stderr, "  -gateway: create user via auth-gateway gRPC")
		fmt.Fprintln(os.Stderr, "  -local:   create user in local database")
		os.Exit(1)
	}

	// Get password if not provided
	if *password == "" {
		*password = promptPassword("Enter password: ")
		if *password == "" {
			fmt.Fprintln(os.Stderr, "Error: password cannot be empty")
			os.Exit(1)
		}

		// Confirm password
		confirm := promptPassword("Confirm password: ")
		if *password != confirm {
			fmt.Fprintln(os.Stderr, "Error: passwords do not match")
			os.Exit(1)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if *useGateway {
		createUserViaGateway(ctx, *email, *username, *password, *fullName, *phone)
	} else {
		createUserLocal(ctx, *email, *username, *password, *fullName, *isAdmin)
	}
}

func createUserViaGateway(ctx context.Context, email, username, password, fullName, phone string) {
	grpcAddress := os.Getenv("MBFLOW_AUTH_GRPC_ADDRESS")
	if grpcAddress == "" {
		fmt.Fprintln(os.Stderr, "Error: MBFLOW_AUTH_GRPC_ADDRESS environment variable is required for -gateway mode")
		fmt.Fprintln(os.Stderr, "Example: MBFLOW_AUTH_GRPC_ADDRESS=\"localhost:50051\"")
		os.Exit(1)
	}

	// Create gRPC provider
	authCfg := &config.AuthConfig{
		GRPCAddress: grpcAddress,
		GRPCTimeout: 30 * time.Second,
	}

	provider, err := auth.NewGRPCProvider(authCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create gRPC provider: %v\n", err)
		os.Exit(1)
	}
	defer provider.Close()

	if !provider.IsAvailable() {
		fmt.Fprintln(os.Stderr, "Error: gRPC auth provider is not available")
		os.Exit(1)
	}

	// Create user via gRPC
	req := &auth.CreateUserRequest{
		Email:       email,
		Phone:       phone,
		Username:    username,
		Password:    password,
		FullName:    fullName,
		AccountType: "human",
	}

	result, err := provider.CreateUser(ctx, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create user via auth-gateway: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("User created successfully via auth-gateway!")
	fmt.Printf("  ID:       %s\n", result.User.ID)
	fmt.Printf("  Email:    %s\n", result.User.Email)
	fmt.Printf("  Username: %s\n", result.User.Username)
	if result.User.FullName != "" {
		fmt.Printf("  Name:     %s\n", result.User.FullName)
	}
	fmt.Printf("  Admin:    %v\n", result.User.IsAdmin)
	if result.AccessToken != "" {
		fmt.Printf("  Token:    %s...\n", result.AccessToken[:min(20, len(result.AccessToken))])
	}
}

func createUserLocal(ctx context.Context, email, username, password, fullName string, isAdmin bool) {
	// Get database URL from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		fmt.Fprintln(os.Stderr, "Error: DATABASE_URL environment variable is required for -local mode")
		fmt.Fprintln(os.Stderr, "Example: DATABASE_URL=\"postgres://user:pass@localhost:5432/mbflow?sslmode=disable\"")
		os.Exit(1)
	}

	// Connect to database
	dbConfig := storage.DefaultConfig()
	dbConfig.DSN = databaseURL

	db, err := storage.NewDB(dbConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer storage.Close(db)

	// Create user repository
	userRepo := storage.NewUserRepository(db)

	// Check if user already exists
	existingUser, _ := userRepo.FindByEmail(ctx, email)
	if existingUser != nil {
		fmt.Fprintf(os.Stderr, "Error: user with email '%s' already exists\n", email)
		os.Exit(1)
	}

	existingUser, _ = userRepo.FindByUsername(ctx, username)
	if existingUser != nil {
		fmt.Fprintf(os.Stderr, "Error: user with username '%s' already exists\n", username)
		os.Exit(1)
	}

	// Create auth config for password service
	authCfg := &config.AuthConfig{
		MinPasswordLength: 8,
	}
	passwordService := auth.NewPasswordService(authCfg.MinPasswordLength)

	// Validate password
	if err := passwordService.ValidatePassword(password); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Hash password
	passwordHash, err := passwordService.HashPassword(password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to hash password: %v\n", err)
		os.Exit(1)
	}

	// Create user
	user := &models.UserModel{
		ID:           uuid.New(),
		Email:        email,
		Username:     username,
		PasswordHash: passwordHash,
		FullName:     fullName,
		IsActive:     true,
		IsAdmin:      isAdmin,
	}

	if err := userRepo.Create(ctx, user); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create user: %v\n", err)
		os.Exit(1)
	}

	// Assign role if admin
	if isAdmin {
		adminRole, err := userRepo.FindRoleByName(ctx, "admin")
		if err == nil && adminRole != nil {
			if err := userRepo.AssignRole(ctx, user.ID, adminRole.ID, nil); err != nil {
				fmt.Printf("Warning: failed to assign admin role: %v\n", err)
			}
		}
	}

	fmt.Println("User created successfully in local database!")
	fmt.Printf("  ID:       %s\n", user.ID)
	fmt.Printf("  Email:    %s\n", user.Email)
	fmt.Printf("  Username: %s\n", user.Username)
	if user.FullName != "" {
		fmt.Printf("  Name:     %s\n", user.FullName)
	}
	fmt.Printf("  Admin:    %v\n", user.IsAdmin)
}

func promptPassword(prompt string) string {
	fmt.Print(prompt)

	// Try to read password without echo
	if term.IsTerminal(int(syscall.Stdin)) {
		password, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println() // Print newline after password input
		if err != nil {
			// Fallback to regular input
			return promptPasswordFallback()
		}
		return string(password)
	}

	// Fallback for non-terminal input
	return promptPasswordFallback()
}

func promptPasswordFallback() string {
	reader := bufio.NewReader(os.Stdin)
	password, _ := reader.ReadString('\n')
	return strings.TrimSpace(password)
}

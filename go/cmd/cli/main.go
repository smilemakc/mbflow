// MBFlow CLI - Command-line tool for workflow management
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/uptrace/bun"
	"golang.org/x/term"

	"github.com/smilemakc/mbflow/go/internal/application/auth"
	"github.com/smilemakc/mbflow/go/internal/application/systemkey"
	"github.com/smilemakc/mbflow/go/internal/config"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/sdk"
	"github.com/smilemakc/mbflow/go/pkg/visualization"
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
    system-key create     Generate a new system key (requires DATABASE_URL)
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

SYSTEM-KEY CREATE OPTIONS:
    -name <name>          Key name (required)
    -description <desc>   Key description (optional)
    -service <name>       Service name (default: cli)
    -created-by <email>   Creator email (default: first admin user)
    -expires <days>       Expiration in days (optional, 0 = no expiration)

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

    # Generate a system key for Service API
    mbflow-cli system-key create -name "my-service-key" -service "backend"

    # Generate a system key with 90-day expiration
    mbflow-cli system-key create -name "temp-key" -expires 90

ENVIRONMENT VARIABLES:
    MBFLOW_ENDPOINT       Server endpoint (overridden by -endpoint)
    MBFLOW_API_KEY        API key (overridden by -api-key)
    DATABASE_URL          Database connection string for local user creation
    MBFLOW_AUTH_GRPC_ADDRESS     Auth-gateway gRPC address (e.g., localhost:50051)
`
)

// gatewayProvider is the part of the auth-gateway gRPC provider the CLI uses.
type gatewayProvider interface {
	IsAvailable() bool
	CreateUser(ctx context.Context, req *auth.CreateUserRequest) (*auth.ProviderAuthResult, error)
	Close() error
}

// cli holds the process dependencies of the command-line tool so commands can
// be run (and tested) without touching the real process environment.
type cli struct {
	stdout io.Writer
	stderr io.Writer
	stdin  *bufio.Reader
	getenv func(string) string

	// readSecret reads a password without echo. ok is false when the input
	// is not a terminal, in which case the password is read from stdin.
	readSecret func() (secret string, ok bool, err error)

	newGatewayProvider func(cfg *config.AuthConfig) (gatewayProvider, error)
}

func newCLI() *cli {
	return &cli{
		stdout: os.Stdout,
		stderr: os.Stderr,
		stdin:  bufio.NewReader(os.Stdin),
		getenv: os.Getenv,
		readSecret: func() (string, bool, error) {
			fd := int(syscall.Stdin)
			if !term.IsTerminal(fd) {
				return "", false, nil
			}
			secret, err := term.ReadPassword(fd)
			return string(secret), true, err
		},
		newGatewayProvider: func(cfg *config.AuthConfig) (gatewayProvider, error) {
			return auth.NewGRPCProvider(cfg)
		},
	}
}

// usageError is an error after which the usage text is printed.
type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

// exit terminates the process; replaceable in tests.
var exit = os.Exit

func main() {
	_ = godotenv.Load()
	exit(newCLI().run(os.Args[1:]))
}

// run executes the command given by args (without the program name) and
// returns the process exit code.
func (c *cli) run(args []string) int {
	err := c.dispatch(args)
	if err == nil || errors.Is(err, flag.ErrHelp) {
		return 0
	}
	_, _ = fmt.Fprintf(c.stderr, "Error: %v\n", err)
	var ue *usageError
	if errors.As(err, &ue) {
		_, _ = fmt.Fprint(c.stderr, usage)
	}
	return 1
}

func (c *cli) dispatch(args []string) error {
	if len(args) < 1 {
		return &usageError{msg: "a command is required"}
	}

	command := args[0]
	subcommands := map[string]map[string]func([]string) error{
		"workflow":   {"show": c.workflowShow, "list": c.workflowList},
		"user":       {"create": c.userCreate},
		"admin":      {"create": c.adminCreate},
		"system-key": {"create": c.systemKeyCreate},
	}

	if subs, ok := subcommands[command]; ok {
		names := make([]string, 0, len(subs))
		for name := range subs {
			names = append(names, name)
		}
		sort.Strings(names)
		if len(args) < 2 {
			return &usageError{msg: fmt.Sprintf("%s command requires a subcommand (%s)", command, strings.Join(names, ", "))}
		}
		handler, ok := subs[args[1]]
		if !ok {
			return fmt.Errorf("unknown %s subcommand: %s", command, args[1])
		}
		return handler(args[2:])
	}

	switch command {
	case "version":
		_, _ = fmt.Fprintf(c.stdout, "MBFlow CLI v%s\n", version)
		return nil
	case "help", "-h", "--help":
		_, _ = fmt.Fprint(c.stdout, usage)
		return nil
	default:
		return &usageError{msg: "unknown command: " + command}
	}
}

func (c *cli) newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(c.stderr)
	return fs
}

func (c *cli) env(key, defaultValue string) string {
	if value := c.getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// connectionFlags registers the flags shared by commands that talk to the server.
func (c *cli) connectionFlags(fs *flag.FlagSet) (endpoint, apiKey *string, timeout *time.Duration) {
	endpoint = fs.String("endpoint", c.env("MBFLOW_ENDPOINT", "http://localhost:8585"), "MBFlow server endpoint")
	apiKey = fs.String("api-key", c.env("MBFLOW_API_KEY", ""), "API key for authentication")
	timeout = fs.Duration("timeout", 30*time.Second, "Request timeout")
	return endpoint, apiKey, timeout
}

func newRemoteClient(endpoint, apiKey string, timeout time.Duration) (*sdk.Client, error) {
	opts := []sdk.ClientOption{sdk.WithHTTPEndpoint(endpoint), sdk.WithTimeout(timeout)}
	if apiKey != "" {
		opts = append(opts, sdk.WithAPIKey(apiKey))
	}
	client, err := sdk.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	return client, nil
}

func (c *cli) workflowShow(args []string) error {
	if len(args) < 1 || strings.HasPrefix(args[0], "-") {
		return errors.New("workflow show requires a workflow ID")
	}
	workflowID := args[0]

	fs := c.newFlagSet("workflow show")
	format := fs.String("format", "mermaid", "Output format: mermaid, ascii")
	direction := fs.String("direction", "TB", "Diagram direction: TB, LR, RL, BT, elk")
	showConfig := fs.Bool("config", true, "Show node configuration details")
	showConditions := fs.Bool("conditions", true, "Show edge conditions")
	compact := fs.Bool("compact", false, "Compact mode for ASCII")
	useColor := fs.Bool("color", true, "Use colors in ASCII")
	output := fs.String("output", "", "Save to file instead of stdout")
	endpoint, apiKey, timeout := c.connectionFlags(fs)
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	*format = strings.ToLower(*format)
	if *format != "mermaid" && *format != "ascii" {
		return fmt.Errorf("invalid format '%s' (must be mermaid or ascii)", *format)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	client, err := newRemoteClient(*endpoint, *apiKey, *timeout)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	workflow, err := client.Workflows().Get(ctx, workflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow '%s': %w", workflowID, err)
	}

	opts := &visualization.RenderOptions{
		ShowConfig:     *showConfig,
		ShowConditions: *showConditions,
		Direction:      *direction,
		CompactMode:    *compact,
		UseColor:       *useColor && *output == "", // Only use color for stdout
	}
	diagram, err := visualization.RenderWorkflow(workflow, *format, opts)
	if err != nil {
		return fmt.Errorf("failed to render workflow: %w", err)
	}

	if *output != "" {
		if err := os.WriteFile(*output, []byte(diagram), 0o644); err != nil {
			return fmt.Errorf("failed to write to file '%s': %w", *output, err)
		}
		_, _ = fmt.Fprintf(c.stdout, "Diagram saved to %s\n", *output)
		return nil
	}
	_, _ = fmt.Fprintln(c.stdout, diagram)
	return nil
}

func (c *cli) workflowList(args []string) error {
	fs := c.newFlagSet("workflow list")
	endpoint, apiKey, timeout := c.connectionFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	client, err := newRemoteClient(*endpoint, *apiKey, *timeout)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	workflows, err := client.Workflows().List(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	if len(workflows) == 0 {
		_, _ = fmt.Fprintln(c.stdout, "No workflows found")
		return nil
	}

	_, _ = fmt.Fprintf(c.stdout, "Found %d workflow(s):\n\n", len(workflows))
	for _, wf := range workflows {
		_, _ = fmt.Fprintf(c.stdout, "ID:          %s\n", wf.ID)
		_, _ = fmt.Fprintf(c.stdout, "Name:        %s\n", wf.Name)
		if wf.Description != "" {
			_, _ = fmt.Fprintf(c.stdout, "Description: %s\n", wf.Description)
		}
		_, _ = fmt.Fprintf(c.stdout, "Status:      %s\n", wf.Status)
		_, _ = fmt.Fprintf(c.stdout, "Nodes:       %d\n", len(wf.Nodes))
		_, _ = fmt.Fprintf(c.stdout, "Edges:       %d\n", len(wf.Edges))
		if len(wf.Tags) > 0 {
			_, _ = fmt.Fprintf(c.stdout, "Tags:        %s\n", strings.Join(wf.Tags, ", "))
		}
		_, _ = fmt.Fprintln(c.stdout, "---")
	}
	return nil
}

// requirePassword prompts for a password (twice) when none was given.
func (c *cli) requirePassword(password *string, prompt string) error {
	if *password != "" {
		return nil
	}
	*password = c.promptPassword(prompt)
	if *password == "" {
		return errors.New("password cannot be empty")
	}
	if confirm := c.promptPassword("Confirm password: "); *password != confirm {
		return errors.New("passwords do not match")
	}
	return nil
}

func (c *cli) databaseURL(mode string) (string, error) {
	databaseURL := c.getenv("DATABASE_URL")
	if databaseURL == "" {
		return "", fmt.Errorf("DATABASE_URL environment variable is required%s\nExample: DATABASE_URL=\"postgres://user:pass@localhost:5432/mbflow?sslmode=disable\"", mode)
	}
	return databaseURL, nil
}

func openDatabase(databaseURL string) (*bun.DB, error) {
	dbConfig := storage.DefaultConfig()
	dbConfig.DSN = databaseURL
	db, err := storage.NewDB(dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return db, nil
}

func (c *cli) adminCreate(args []string) error {
	fs := c.newFlagSet("admin create")
	email := fs.String("email", "", "Admin email address (required)")
	username := fs.String("username", "", "Admin username (required)")
	password := fs.String("password", "", "Admin password (will prompt if not provided)")
	fullName := fs.String("full-name", "", "Admin full name (optional)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *email == "" {
		return errors.New("-email is required")
	}
	if *username == "" {
		return errors.New("-username is required")
	}
	if err := c.requirePassword(password, "Enter admin password: "); err != nil {
		return err
	}
	databaseURL, err := c.databaseURL("")
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	user, err := createLocalUser(ctx, c.stdout, databaseURL, *email, *username, *password, *fullName, true)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(c.stdout, "Admin user created successfully!")
	c.printUser(user.ID.String(), user.Email, user.Username, user.FullName, user.IsAdmin)
	return nil
}

func (c *cli) printUser(id, email, username, fullName string, isAdmin bool) {
	_, _ = fmt.Fprintf(c.stdout, "  ID:       %s\n", id)
	_, _ = fmt.Fprintf(c.stdout, "  Email:    %s\n", email)
	_, _ = fmt.Fprintf(c.stdout, "  Username: %s\n", username)
	if fullName != "" {
		_, _ = fmt.Fprintf(c.stdout, "  Name:     %s\n", fullName)
	}
	_, _ = fmt.Fprintf(c.stdout, "  Admin:    %v\n", isAdmin)
}

// createLocalUser creates a user directly in the database. Admin users also
// get the "admin" role when it exists; failing to assign it is only a warning.
func createLocalUser(ctx context.Context, warn io.Writer, databaseURL, email, username, password, fullName string, isAdmin bool) (*models.UserModel, error) {
	db, err := openDatabase(databaseURL)
	if err != nil {
		return nil, err
	}
	defer storage.Close(db)

	userRepo := storage.NewUserRepository(db)

	if existing, _ := userRepo.FindByEmail(ctx, email); existing != nil {
		return nil, fmt.Errorf("user with email '%s' already exists", email)
	}
	if existing, _ := userRepo.FindByUsername(ctx, username); existing != nil {
		return nil, fmt.Errorf("user with username '%s' already exists", username)
	}

	authCfg := &config.AuthConfig{MinPasswordLength: 8}
	passwordService := auth.NewPasswordService(authCfg.MinPasswordLength)
	if err := passwordService.ValidatePassword(password); err != nil {
		return nil, err
	}
	passwordHash, err := passwordService.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

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
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if isAdmin {
		adminRole, err := userRepo.FindRoleByName(ctx, "admin")
		if err == nil && adminRole != nil {
			if err := userRepo.AssignRole(ctx, user.ID, adminRole.ID, nil); err != nil {
				_, _ = fmt.Fprintf(warn, "Warning: failed to assign admin role: %v\n", err)
			}
		}
	}
	return user, nil
}

func (c *cli) systemKeyCreate(args []string) error {
	fs := c.newFlagSet("system-key create")
	name := fs.String("name", "", "Key name (required)")
	description := fs.String("description", "", "Key description")
	serviceName := fs.String("service", "cli", "Service name")
	createdByEmail := fs.String("created-by", "", "Creator email (default: first admin user)")
	expires := fs.Int("expires", 0, "Expiration in days (0 = no expiration)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *name == "" {
		return errors.New("-name is required")
	}
	databaseURL, err := c.databaseURL("")
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := openDatabase(databaseURL)
	if err != nil {
		return err
	}
	defer storage.Close(db)

	creatorID, err := resolveCreatorID(ctx, storage.NewUserRepository(db), *createdByEmail)
	if err != nil {
		return err
	}

	svc := systemkey.NewService(storage.NewSystemKeyRepo(db), systemkey.Config{})

	var expiresInDays *int
	if *expires > 0 {
		expiresInDays = expires
	}

	result, err := svc.CreateKey(ctx, *name, *description, *serviceName, creatorID, expiresInDays)
	if err != nil {
		return fmt.Errorf("failed to create system key: %w", err)
	}

	_, _ = fmt.Fprintln(c.stdout, "System key created successfully!")
	_, _ = fmt.Fprintf(c.stdout, "  ID:      %s\n", result.Key.ID)
	_, _ = fmt.Fprintf(c.stdout, "  Name:    %s\n", result.Key.Name)
	_, _ = fmt.Fprintf(c.stdout, "  Service: %s\n", result.Key.ServiceName)
	if result.Key.ExpiresAt != nil {
		_, _ = fmt.Fprintf(c.stdout, "  Expires: %s\n", result.Key.ExpiresAt.Format(time.RFC3339))
	}
	_, _ = fmt.Fprintln(c.stdout)
	_, _ = fmt.Fprintln(c.stdout, "  Plain key (save it now, it will not be shown again):")
	_, _ = fmt.Fprintf(c.stdout, "  %s\n", result.PlainKey)
	return nil
}

// creatorLookup is the part of the user repository resolveCreatorID needs.
type creatorLookup interface {
	FindByEmail(ctx context.Context, email string) (*models.UserModel, error)
	FindAll(ctx context.Context, limit, offset int) ([]*models.UserModel, error)
}

func resolveCreatorID(ctx context.Context, userRepo creatorLookup, email string) (uuid.UUID, error) {
	if email != "" {
		user, err := userRepo.FindByEmail(ctx, email)
		if err != nil || user == nil {
			return uuid.Nil, fmt.Errorf("user with email '%s' not found", email)
		}
		return user.ID, nil
	}

	// Fallback: find first admin user
	users, err := userRepo.FindAll(ctx, 100, 0)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to query users: %w", err)
	}
	for _, u := range users {
		if u.IsAdmin {
			return u.ID, nil
		}
	}
	return uuid.Nil, errors.New("no admin user found; create one first with 'admin create' or specify -created-by")
}

func (c *cli) userCreate(args []string) error {
	fs := c.newFlagSet("user create")
	email := fs.String("email", "", "User email address (required)")
	username := fs.String("username", "", "Username (required)")
	password := fs.String("password", "", "Password (will prompt if not provided)")
	fullName := fs.String("full-name", "", "Full name (optional)")
	phone := fs.String("phone", "", "Phone number (optional)")
	isAdmin := fs.Bool("admin", false, "Create as admin user")
	useGateway := fs.Bool("gateway", false, "Create user via auth-gateway gRPC")
	useLocal := fs.Bool("local", false, "Create user in local database")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *email == "" {
		return errors.New("-email is required")
	}
	if *username == "" {
		return errors.New("-username is required")
	}
	if !*useGateway && !*useLocal {
		return errors.New("must specify either -gateway or -local\n  -gateway: create user via auth-gateway gRPC\n  -local:   create user in local database")
	}
	if err := c.requirePassword(password, "Enter password: "); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if *useGateway {
		return c.createUserViaGateway(ctx, *email, *username, *password, *fullName, *phone)
	}

	databaseURL, err := c.databaseURL(" for -local mode")
	if err != nil {
		return err
	}
	user, err := createLocalUser(ctx, c.stdout, databaseURL, *email, *username, *password, *fullName, *isAdmin)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(c.stdout, "User created successfully in local database!")
	c.printUser(user.ID.String(), user.Email, user.Username, user.FullName, user.IsAdmin)
	return nil
}

func (c *cli) createUserViaGateway(ctx context.Context, email, username, password, fullName, phone string) error {
	grpcAddress := c.getenv("MBFLOW_AUTH_GRPC_ADDRESS")
	if grpcAddress == "" {
		return errors.New("MBFLOW_AUTH_GRPC_ADDRESS environment variable is required for -gateway mode\nExample: MBFLOW_AUTH_GRPC_ADDRESS=\"localhost:50051\"")
	}

	provider, err := c.newGatewayProvider(&config.AuthConfig{
		GRPCAddress: grpcAddress,
		GRPCTimeout: 30 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to create gRPC provider: %w", err)
	}
	defer provider.Close()

	if !provider.IsAvailable() {
		return errors.New("gRPC auth provider is not available")
	}

	result, err := provider.CreateUser(ctx, &auth.CreateUserRequest{
		Email:       email,
		Phone:       phone,
		Username:    username,
		Password:    password,
		FullName:    fullName,
		AccountType: "human",
	})
	if err != nil {
		return fmt.Errorf("failed to create user via auth-gateway: %w", err)
	}
	if result == nil || result.User == nil {
		return errors.New("auth-gateway returned no user")
	}

	_, _ = fmt.Fprintln(c.stdout, "User created successfully via auth-gateway!")
	c.printUser(result.User.ID, result.User.Email, result.User.Username, result.User.FullName, result.User.IsAdmin)
	if result.AccessToken != "" {
		_, _ = fmt.Fprintf(c.stdout, "  Token:    %s...\n", result.AccessToken[:min(20, len(result.AccessToken))])
	}
	return nil
}

// promptPassword reads a password without echo from a terminal, or a line
// from stdin otherwise. stdin is read through one shared buffered reader so
// that consecutive prompts (password and confirmation) both see their line.
func (c *cli) promptPassword(prompt string) string {
	_, _ = fmt.Fprint(c.stdout, prompt)

	if c.readSecret != nil {
		secret, ok, err := c.readSecret()
		if ok {
			_, _ = fmt.Fprintln(c.stdout) // newline after hidden input
			if err == nil {
				return secret
			}
		}
	}

	line, _ := c.stdin.ReadString('\n')
	return strings.TrimSpace(line)
}

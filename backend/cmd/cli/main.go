// MBFlow CLI - Command-line tool for workflow management
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/smilemakc/mbflow/pkg/sdk"
	"github.com/smilemakc/mbflow/pkg/visualization"
)

const (
	version = "1.0.0"
	usage   = `MBFlow CLI - Workflow management tool

USAGE:
    mbflow-cli <command> [options]

COMMANDS:
    workflow show <id>    Show workflow diagram
    workflow list         List all workflows
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

CONNECTION OPTIONS:
    -endpoint <url>       MBFlow server endpoint (default: http://localhost:8181)
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

ENVIRONMENT VARIABLES:
    MBFLOW_ENDPOINT       Server endpoint (overridden by -endpoint)
    MBFLOW_API_KEY        API key (overridden by -api-key)
`
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "workflow":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Error: workflow command requires a subcommand (show, list)")
			fmt.Fprintln(os.Stderr, usage)
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

	case "version":
		fmt.Printf("MBFlow CLI v%s\n", version)

	case "help", "-h", "--help":
		fmt.Println(usage)

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command: %s\n", command)
		fmt.Fprintln(os.Stderr, usage)
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
	endpoint := fs.String("endpoint", getEnv("MBFLOW_ENDPOINT", "http://localhost:8181"), "MBFlow server endpoint")
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
	endpoint := fs.String("endpoint", getEnv("MBFLOW_ENDPOINT", "http://localhost:8181"), "MBFlow server endpoint")
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

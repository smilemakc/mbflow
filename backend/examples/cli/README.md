# CLI Visualization Example

This example demonstrates how to use the `mbflow-cli` tool to visualize workflows.

## Prerequisites

1. Build the CLI tool:
```bash
go build -o bin/mbflow-cli ./cmd/cli
```

2. Have a running MBFlow server (optional for this demo):
```bash
go run ./cmd/server
```

## Using CLI with Local Workflow File

Since we don't have the workflow in the server yet, we can use the visualization library directly:

```bash
# View the example workflow
cat examples/cli/test_workflow.json
```

## Demo Script

Run the demo script to see all visualization formats:

```bash
cd examples/cli
go run demo.go
```

This will demonstrate:
1. Mermaid diagram with top-bottom layout
2. Mermaid diagram with ELK adaptive layout
3. ASCII tree visualization (compact mode)
4. ASCII tree visualization (detailed mode)
5. Saving to file

## Using CLI with Server

Once you have workflows in the server:

```bash
# List all workflows
./bin/mbflow-cli workflow list -endpoint http://localhost:8181

# Show workflow as Mermaid diagram
./bin/mbflow-cli workflow show <workflow-id> -format mermaid

# Show workflow as ASCII tree
./bin/mbflow-cli workflow show <workflow-id> -format ascii -compact

# Save diagram to file with ELK layout
./bin/mbflow-cli workflow show <workflow-id> -format mermaid -direction elk -output diagram.mmd
```

## Environment Variables

You can set default connection parameters:

```bash
export MBFLOW_ENDPOINT=http://localhost:8181
export MBFLOW_API_KEY=your-api-key

# Now you can use CLI without specifying endpoint
./bin/mbflow-cli workflow list
```

## Visualization Options

### Mermaid Format
- `-direction TB` - Top to Bottom (default)
- `-direction LR` - Left to Right
- `-direction elk` - Adaptive ELK layout (best for complex workflows)
- `-config` - Show node configuration (default: true)
- `-conditions` - Show edge conditions (default: true)

### ASCII Format
- `-compact` - Compact tree view
- `-color` - Use ANSI colors (default: true)
- `-config` - Show node configuration details
- `-conditions` - Show edge conditions on arrows

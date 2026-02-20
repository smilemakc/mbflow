package main

// FileRule describes what to extract from a single source file and where to write it.
type FileRule struct {
	SourceFile    string              // Source filename in pkg/models/ (e.g., "workflow.go")
	OutputFile    string              // Output filename in sdk/go/models/ (e.g., "workflow.go")
	ExcludeTypes  map[string]bool     // Type names to skip entirely
	ExcludeFields map[string][]string // Struct name -> field names to skip
}

var fileRules = []FileRule{
	{
		SourceFile:    "workflow.go",
		OutputFile:    "workflow.go",
		ExcludeTypes:  map[string]bool{"WorkflowResource": true},
		ExcludeFields: map[string][]string{"Workflow": {"Resources"}},
	},
	{
		SourceFile: "execution.go",
		OutputFile: "execution.go",
	},
	{
		SourceFile: "trigger.go",
		OutputFile: "trigger.go",
		ExcludeTypes: map[string]bool{
			"CronConfig":     true,
			"WebhookConfig":  true,
			"EventConfig":    true,
			"IntervalConfig": true,
		},
	},
	{
		SourceFile: "event.go",
		OutputFile: "event.go",
	},
}

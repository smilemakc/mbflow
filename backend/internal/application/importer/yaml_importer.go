package importer

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/models"
)

// YAMLWorkflow represents the top-level YAML workflow configuration.
type YAMLWorkflow struct {
	Metadata  YAMLMetadata   `yaml:"metadata"`
	Variables map[string]any `yaml:"variables,omitempty"`
	Nodes     []YAMLNode     `yaml:"nodes"`
	Edges     []YAMLEdge     `yaml:"edges,omitempty"`
	Trigger   *YAMLTrigger   `yaml:"trigger,omitempty"`
}

// YAMLMetadata represents workflow metadata in YAML.
type YAMLMetadata struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Version     int      `yaml:"version,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
}

// YAMLNode represents a node in YAML format.
type YAMLNode struct {
	ID          string         `yaml:"id"`
	Name        string         `yaml:"name"`
	Type        string         `yaml:"type"`
	Description string         `yaml:"description,omitempty"`
	Config      map[string]any `yaml:"config,omitempty"`
	Position    *YAMLPosition  `yaml:"position,omitempty"`
	Metadata    map[string]any `yaml:"metadata,omitempty"`
}

// YAMLPosition represents node position in YAML.
type YAMLPosition struct {
	X float64 `yaml:"x"`
	Y float64 `yaml:"y"`
}

// YAMLEdge represents an edge in YAML format.
type YAMLEdge struct {
	ID           string         `yaml:"id"`
	From         string         `yaml:"from"`
	To           string         `yaml:"to"`
	SourceHandle string         `yaml:"source_handle,omitempty"`
	Condition    string         `yaml:"condition,omitempty"`
	Metadata     map[string]any `yaml:"metadata,omitempty"`
}

// YAMLTrigger represents a trigger in YAML format.
type YAMLTrigger struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description,omitempty"`
	Type        string         `yaml:"type"`
	Enabled     *bool          `yaml:"enabled,omitempty"`
	Config      map[string]any `yaml:"config,omitempty"`
	Metadata    map[string]any `yaml:"metadata,omitempty"`
}

// ImportResult contains the result of importing a YAML workflow.
type ImportResult struct {
	Workflow   *models.Workflow
	Trigger    *models.Trigger
	NodesCount int
	EdgesCount int
}

// ValidationError represents a validation error with context.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// YAMLImporter handles importing and exporting YAML workflow configurations.
type YAMLImporter struct {
	executorManager executor.Manager
}

// NewYAMLImporter creates a new YAML importer with the given executor manager.
func NewYAMLImporter(executorManager executor.Manager) *YAMLImporter {
	return &YAMLImporter{
		executorManager: executorManager,
	}
}

// ImportFromYAML parses YAML data and converts it to domain models.
func (i *YAMLImporter) ImportFromYAML(data []byte) (*ImportResult, error) {
	var yamlWorkflow YAMLWorkflow
	if err := yaml.Unmarshal(data, &yamlWorkflow); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate YAML structure
	if err := i.validateYAML(&yamlWorkflow); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Convert to domain models
	workflow := i.convertToWorkflow(&yamlWorkflow)
	var trigger *models.Trigger
	if yamlWorkflow.Trigger != nil {
		trigger = i.convertToTrigger(&yamlWorkflow, workflow.ID)
	}

	// Validate domain models
	if err := workflow.Validate(); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	if trigger != nil {
		if err := trigger.Validate(); err != nil {
			return nil, fmt.Errorf("trigger validation failed: %w", err)
		}
	}

	return &ImportResult{
		Workflow:   workflow,
		Trigger:    trigger,
		NodesCount: len(workflow.Nodes),
		EdgesCount: len(workflow.Edges),
	}, nil
}

// validateYAML validates the YAML structure before conversion.
func (i *YAMLImporter) validateYAML(y *YAMLWorkflow) error {
	// Validate metadata
	if y.Metadata.Name == "" {
		return &ValidationError{Field: "metadata.name", Message: "workflow name is required"}
	}

	// Validate nodes
	if len(y.Nodes) == 0 {
		return &ValidationError{Field: "nodes", Message: "at least one node is required"}
	}

	nodeIDs := make(map[string]bool)
	for idx, node := range y.Nodes {
		if node.ID == "" {
			return &ValidationError{
				Field:   fmt.Sprintf("nodes[%d].id", idx),
				Message: "node ID is required",
			}
		}
		if nodeIDs[node.ID] {
			return &ValidationError{
				Field:   fmt.Sprintf("nodes[%d].id", idx),
				Message: fmt.Sprintf("duplicate node ID: %s", node.ID),
			}
		}
		nodeIDs[node.ID] = true

		if node.Name == "" {
			return &ValidationError{
				Field:   fmt.Sprintf("nodes[%d].name", idx),
				Message: "node name is required",
			}
		}

		if node.Type == "" {
			return &ValidationError{
				Field:   fmt.Sprintf("nodes[%d].type", idx),
				Message: "node type is required",
			}
		}

		// Validate node type exists in executor registry
		if i.executorManager != nil && !i.executorManager.Has(node.Type) {
			return &ValidationError{
				Field:   fmt.Sprintf("nodes[%d].type", idx),
				Message: fmt.Sprintf("unknown executor type: %s", node.Type),
			}
		}
	}

	// Validate edges
	edgeIDs := make(map[string]bool)
	for idx, edge := range y.Edges {
		if edge.ID == "" {
			return &ValidationError{
				Field:   fmt.Sprintf("edges[%d].id", idx),
				Message: "edge ID is required",
			}
		}
		if edgeIDs[edge.ID] {
			return &ValidationError{
				Field:   fmt.Sprintf("edges[%d].id", idx),
				Message: fmt.Sprintf("duplicate edge ID: %s", edge.ID),
			}
		}
		edgeIDs[edge.ID] = true

		if edge.From == "" {
			return &ValidationError{
				Field:   fmt.Sprintf("edges[%d].from", idx),
				Message: "edge source is required",
			}
		}

		if edge.To == "" {
			return &ValidationError{
				Field:   fmt.Sprintf("edges[%d].to", idx),
				Message: "edge target is required",
			}
		}

		if !nodeIDs[edge.From] {
			return &ValidationError{
				Field:   fmt.Sprintf("edges[%d].from", idx),
				Message: fmt.Sprintf("edge references non-existent source node: %s", edge.From),
			}
		}

		if !nodeIDs[edge.To] {
			return &ValidationError{
				Field:   fmt.Sprintf("edges[%d].to", idx),
				Message: fmt.Sprintf("edge references non-existent target node: %s", edge.To),
			}
		}

		if edge.From == edge.To {
			return &ValidationError{
				Field:   fmt.Sprintf("edges[%d]", idx),
				Message: "self-loop edges are not allowed",
			}
		}
	}

	// Validate trigger if present
	if y.Trigger != nil {
		if y.Trigger.Name == "" {
			return &ValidationError{Field: "trigger.name", Message: "trigger name is required"}
		}

		if y.Trigger.Type == "" {
			return &ValidationError{Field: "trigger.type", Message: "trigger type is required"}
		}

		validTriggerTypes := map[string]bool{
			"manual":   true,
			"cron":     true,
			"webhook":  true,
			"event":    true,
			"interval": true,
		}
		if !validTriggerTypes[y.Trigger.Type] {
			return &ValidationError{
				Field:   "trigger.type",
				Message: fmt.Sprintf("invalid trigger type: %s", y.Trigger.Type),
			}
		}
	}

	return nil
}

// convertToWorkflow converts YAML structure to domain Workflow.
func (i *YAMLImporter) convertToWorkflow(y *YAMLWorkflow) *models.Workflow {
	now := time.Now()
	workflowID := uuid.New().String()

	version := y.Metadata.Version
	if version == 0 {
		version = 1
	}

	workflow := &models.Workflow{
		ID:          workflowID,
		Name:        y.Metadata.Name,
		Description: y.Metadata.Description,
		Version:     version,
		Status:      models.WorkflowStatusDraft,
		Tags:        y.Metadata.Tags,
		Variables:   y.Variables,
		Nodes:       make([]*models.Node, 0, len(y.Nodes)),
		Edges:       make([]*models.Edge, 0, len(y.Edges)),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Convert nodes
	for _, yamlNode := range y.Nodes {
		node := &models.Node{
			ID:          yamlNode.ID,
			Name:        yamlNode.Name,
			Type:        yamlNode.Type,
			Description: yamlNode.Description,
			Config:      yamlNode.Config,
			Metadata:    yamlNode.Metadata,
		}
		if yamlNode.Position != nil {
			node.Position = &models.Position{
				X: yamlNode.Position.X,
				Y: yamlNode.Position.Y,
			}
		}
		if node.Config == nil {
			node.Config = make(map[string]any)
		}
		workflow.Nodes = append(workflow.Nodes, node)
	}

	// Convert edges
	for _, yamlEdge := range y.Edges {
		edge := &models.Edge{
			ID:           yamlEdge.ID,
			From:         yamlEdge.From,
			To:           yamlEdge.To,
			SourceHandle: yamlEdge.SourceHandle,
			Condition:    yamlEdge.Condition,
			Metadata:     yamlEdge.Metadata,
		}
		workflow.Edges = append(workflow.Edges, edge)
	}

	return workflow
}

// convertToTrigger converts YAML trigger to domain Trigger.
func (i *YAMLImporter) convertToTrigger(y *YAMLWorkflow, workflowID string) *models.Trigger {
	if y.Trigger == nil {
		return nil
	}

	now := time.Now()
	enabled := true
	if y.Trigger.Enabled != nil {
		enabled = *y.Trigger.Enabled
	}

	config := y.Trigger.Config
	if config == nil {
		config = make(map[string]any)
	}

	return &models.Trigger{
		ID:          uuid.New().String(),
		WorkflowID:  workflowID,
		Name:        y.Trigger.Name,
		Description: y.Trigger.Description,
		Type:        models.TriggerType(y.Trigger.Type),
		Config:      config,
		Enabled:     enabled,
		Metadata:    y.Trigger.Metadata,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// ExportToYAML exports a workflow and optional trigger to YAML format.
func (i *YAMLImporter) ExportToYAML(workflow *models.Workflow, trigger *models.Trigger) ([]byte, error) {
	yamlWorkflow := i.convertFromWorkflow(workflow, trigger)
	return yaml.Marshal(yamlWorkflow)
}

// convertFromWorkflow converts domain models back to YAML structure.
func (i *YAMLImporter) convertFromWorkflow(workflow *models.Workflow, trigger *models.Trigger) *YAMLWorkflow {
	y := &YAMLWorkflow{
		Metadata: YAMLMetadata{
			Name:        workflow.Name,
			Description: workflow.Description,
			Version:     workflow.Version,
			Tags:        workflow.Tags,
		},
		Variables: workflow.Variables,
		Nodes:     make([]YAMLNode, 0, len(workflow.Nodes)),
		Edges:     make([]YAMLEdge, 0, len(workflow.Edges)),
	}

	// Convert nodes
	for _, node := range workflow.Nodes {
		yamlNode := YAMLNode{
			ID:          node.ID,
			Name:        node.Name,
			Type:        node.Type,
			Description: node.Description,
			Config:      node.Config,
			Metadata:    node.Metadata,
		}
		if node.Position != nil {
			yamlNode.Position = &YAMLPosition{
				X: node.Position.X,
				Y: node.Position.Y,
			}
		}
		y.Nodes = append(y.Nodes, yamlNode)
	}

	// Convert edges
	for _, edge := range workflow.Edges {
		yamlEdge := YAMLEdge{
			ID:           edge.ID,
			From:         edge.From,
			To:           edge.To,
			SourceHandle: edge.SourceHandle,
			Condition:    edge.Condition,
			Metadata:     edge.Metadata,
		}
		y.Edges = append(y.Edges, yamlEdge)
	}

	// Convert trigger
	if trigger != nil {
		enabled := trigger.Enabled
		y.Trigger = &YAMLTrigger{
			Name:        trigger.Name,
			Description: trigger.Description,
			Type:        string(trigger.Type),
			Enabled:     &enabled,
			Config:      trigger.Config,
			Metadata:    trigger.Metadata,
		}
	}

	return y
}

// GetSupportedNodeTypes returns a list of supported node types.
func (i *YAMLImporter) GetSupportedNodeTypes() []string {
	if i.executorManager == nil {
		return nil
	}
	return i.executorManager.List()
}

// ValidateNodeType checks if a node type is supported.
func (i *YAMLImporter) ValidateNodeType(nodeType string) bool {
	if i.executorManager == nil {
		return true // Allow all types if no executor manager
	}
	return i.executorManager.Has(nodeType)
}

// ParseYAMLContent handles different YAML content formats (with or without comments/frontmatter).
func ParseYAMLContent(data []byte) ([]byte, error) {
	content := string(data)

	// Remove BOM if present
	content = strings.TrimPrefix(content, "\xef\xbb\xbf")

	// Handle potential frontmatter or leading whitespace
	content = strings.TrimSpace(content)

	// Check if content is empty
	if content == "" {
		return nil, fmt.Errorf("empty YAML content")
	}

	return []byte(content), nil
}

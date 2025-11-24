package domain

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// VariableScope defines the scope of a variable
type VariableScope string

const (
	ScopeWorkflow  VariableScope = "workflow"  // Workflow-level (global for all executions)
	ScopeExecution VariableScope = "execution" // Execution-level (per execution instance)
	ScopeNode      VariableScope = "node"      // Node-level (local to node execution)
)

// VariableDefinition defines the schema for a variable (optional typing)
type VariableDefinition struct {
	Name         string
	Type         VariableType
	Required     bool
	DefaultValue any
	Description  string
}

// Validate validates a value against this definition
func (vd *VariableDefinition) Validate(value any) error {
	// Check if required and nil
	if vd.Required && value == nil {
		return NewDomainError(
			ErrCodeValidationFailed,
			fmt.Sprintf("variable '%s' is required but not provided", vd.Name),
			nil,
		)
	}

	// If type is specified and not 'any', validate type
	if vd.Type != VariableTypeAny && vd.Type != VariableTypeUnknown {
		inferredType := InferType(value)
		if inferredType != vd.Type && inferredType != VariableTypeUnknown {
			return NewDomainError(
				ErrCodeInvalidType,
				fmt.Sprintf("variable '%s' expected type %s but got %s", vd.Name, vd.Type, inferredType),
				nil,
			)
		}
	}

	return nil
}

// VariableSchema is an optional schema for validating variables
type VariableSchema struct {
	definitions map[string]*VariableDefinition
}

// NewVariableSchema creates a new variable schema
func NewVariableSchema() *VariableSchema {
	return &VariableSchema{
		definitions: make(map[string]*VariableDefinition),
	}
}

// AddDefinition adds a variable definition to the schema
func (vs *VariableSchema) AddDefinition(def *VariableDefinition) {
	vs.definitions[def.Name] = def
}

// GetDefinition gets a variable definition
func (vs *VariableSchema) GetDefinition(name string) (*VariableDefinition, bool) {
	def, exists := vs.definitions[name]
	return def, exists
}

func (vs *VariableSchema) GetDefinitions() map[string]*VariableDefinition {
	return vs.definitions
}

// Validate validates a map of variables against the schema
func (vs *VariableSchema) Validate(variables map[string]any) error {
	// Check required variables
	for name, def := range vs.definitions {
		if def.Required {
			if _, exists := variables[name]; !exists {
				return NewDomainError(
					ErrCodeValidationFailed,
					fmt.Sprintf("required variable '%s' is missing", name),
					nil,
				)
			}
		}
	}

	// Validate types of provided variables
	for name, value := range variables {
		if def, exists := vs.definitions[name]; exists {
			if err := def.Validate(value); err != nil {
				return err
			}
		}
		// If no definition exists, allow the variable (permissive schema)
	}

	return nil
}

// NodeIOSchema defines input/output schema for a node
type NodeIOSchema struct {
	Inputs  *VariableSchema // What this node requires
	Outputs *VariableSchema // What this node produces
}

// ParseStructToVariableSchema builds VariableSchema from any struct.
func ParseStructToVariableSchema(v any) (*VariableSchema, error) {
	val := reflect.ValueOf(v)
	typ := reflect.TypeOf(v)

	// Pointer → value
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
		val = val.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got: %s", typ.Kind())
	}

	schema := &VariableSchema{
		definitions: make(map[string]*VariableDefinition),
	}

	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		fieldVal := val.Field(i)

		// skip unexported
		if f.PkgPath != "" {
			continue
		}

		jsonName := parseJSONName(f)
		if jsonName == "" {
			jsonName = f.Name
		}

		schema.definitions[jsonName] = &VariableDefinition{
			Name:         jsonName,
			Type:         detectVarType(f.Type),
			Required:     !strings.Contains(f.Tag.Get("json"), "omitempty"),
			DefaultValue: zeroValue(fieldVal),
			Description:  f.Tag.Get("desc"),
		}
	}

	return schema, nil
}

// ---------------- Helpers ----------------

func parseJSONName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return ""
	}
	parts := strings.Split(tag, ",")
	if parts[0] == "" || parts[0] == "-" {
		return ""
	}
	return parts[0]
}

func detectVarType(t reflect.Type) VariableType {
	switch t.Kind() {
	case reflect.String:
		return VariableTypeString
	case reflect.Slice, reflect.Array:
		return VariableTypeArray
	case reflect.Bool:
		return VariableTypeBool
	case reflect.Int, reflect.Int64, reflect.Int32:
		return VariableTypeInt
	case reflect.Float32, reflect.Float64:
		return VariableTypeFloat
	case reflect.Map:
		return VariableTypeObject
	case reflect.Interface:
		return VariableTypeAny
	default:
		return VariableTypeArray
	}
}

func zeroValue(v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}
	return reflect.Zero(v.Type()).Interface()
}

// CollisionStrategy defines how to handle collisions when merging parent outputs
type CollisionStrategy string

const (
	// CollisionStrategyNamespaceByParent namespaces each parent's output by parent name
	// Example: branch1.status, branch2.status
	CollisionStrategyNamespaceByParent CollisionStrategy = "namespace"

	// CollisionStrategyCollect collects colliding values into arrays
	// Example: status: [200, 404]
	CollisionStrategyCollect CollisionStrategy = "collect"

	// CollisionStrategyError fails execution on any collision
	CollisionStrategyError CollisionStrategy = "error"
)

// InputBindingConfig defines how to bind parent outputs to node inputs
type InputBindingConfig struct {
	// AutoBind enables automatic binding by matching variable names (default: true)
	AutoBind bool

	// Mappings defines explicit mappings for renames/conflicts
	// Key: node input name, Value: source path (e.g., "parent_node.field")
	Mappings map[string]string

	// CollisionStrategy defines how to handle collisions when multiple parents
	// produce the same output key
	CollisionStrategy CollisionStrategy
}

// VariableSet manages a set of variables with optional schema validation
type VariableSet struct {
	mu        sync.RWMutex
	variables map[string]any
	schema    *VariableSchema // Optional schema for validation
	readOnly  bool            // If true, Set operations will fail
}

// NewVariableSet creates a new variable set
func NewVariableSet(schema *VariableSchema) *VariableSet {
	return &VariableSet{
		variables: make(map[string]any),
		schema:    schema,
	}
}

// NewVariableSetFromMap creates a variable set from a map
func NewVariableSetFromMap(variables map[string]any, schema *VariableSchema) (*VariableSet, error) {
	vs := NewVariableSet(schema)

	// Validate if schema exists
	if schema != nil {
		if err := schema.Validate(variables); err != nil {
			return nil, err
		}
	}

	for k, v := range variables {
		vs.variables[k] = v
	}

	return vs, nil
}

// Set sets a variable value
func (vs *VariableSet) Set(name string, value any) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	// Check if read-only
	if vs.readOnly {
		return NewDomainError(
			ErrCodeInvalidState,
			"cannot modify read-only variable set",
			nil,
		)
	}

	// Validate against schema if exists
	if vs.schema != nil {
		if def, exists := vs.schema.GetDefinition(name); exists {
			if err := def.Validate(value); err != nil {
				return err
			}
		}
	}

	vs.variables[name] = value
	return nil
}

// Get gets a variable value
func (vs *VariableSet) Get(name string) (any, bool) {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	value, exists := vs.variables[name]
	return value, exists
}

// GetString gets a variable as string
func (vs *VariableSet) GetString(name string) (string, error) {
	value, exists := vs.Get(name)
	if !exists {
		return "", NewDomainError(ErrCodeNotFound, fmt.Sprintf("variable '%s' not found", name), nil)
	}

	str, ok := value.(string)
	if !ok {
		return "", NewDomainError(
			ErrCodeInvalidType,
			fmt.Sprintf("variable '%s' is not a string", name),
			nil,
		)
	}

	return str, nil
}

// GetInt gets a variable as int
func (vs *VariableSet) GetInt(name string) (int, error) {
	value, exists := vs.Get(name)
	if !exists {
		return 0, NewDomainError(ErrCodeNotFound, fmt.Sprintf("variable '%s' not found", name), nil)
	}

	// Try different int types
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		return 0, NewDomainError(
			ErrCodeInvalidType,
			fmt.Sprintf("variable '%s' is not an int", name),
			nil,
		)
	}
}

// GetBool gets a variable as bool
func (vs *VariableSet) GetBool(name string) (bool, error) {
	value, exists := vs.Get(name)
	if !exists {
		return false, NewDomainError(ErrCodeNotFound, fmt.Sprintf("variable '%s' not found", name), nil)
	}

	b, ok := value.(bool)
	if !ok {
		return false, NewDomainError(
			ErrCodeInvalidType,
			fmt.Sprintf("variable '%s' is not a bool", name),
			nil,
		)
	}

	return b, nil
}

// GetMap gets a variable as map
func (vs *VariableSet) GetMap(name string) (map[string]any, error) {
	value, exists := vs.Get(name)
	if !exists {
		return nil, NewDomainError(ErrCodeNotFound, fmt.Sprintf("variable '%s' not found", name), nil)
	}

	m, ok := value.(map[string]any)
	if !ok {
		return nil, NewDomainError(
			ErrCodeInvalidType,
			fmt.Sprintf("variable '%s' is not a map", name),
			nil,
		)
	}

	return m, nil
}

// Delete deletes a variable
func (vs *VariableSet) Delete(name string) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	delete(vs.variables, name)
}

// Has checks if a variable exists
func (vs *VariableSet) Has(name string) bool {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	_, exists := vs.variables[name]
	return exists
}

// All returns all variables as a map (copy)
func (vs *VariableSet) All() map[string]any {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	result := make(map[string]any, len(vs.variables))
	for k, v := range vs.variables {
		result[k] = v
	}

	return result
}

// Merge merges variables from another set
func (vs *VariableSet) Merge(other *VariableSet) error {
	if other == nil {
		return nil
	}

	otherVars := other.All()
	for k, v := range otherVars {
		if err := vs.Set(k, v); err != nil {
			return err
		}
	}

	return nil
}

// Clone creates a copy of the variable set
func (vs *VariableSet) Clone() *VariableSet {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	clone := &VariableSet{
		variables: make(map[string]any, len(vs.variables)),
		schema:    vs.schema,
		readOnly:  false, // Clone is writable by default
	}

	for k, v := range vs.variables {
		clone.variables[k] = v
	}

	return clone
}

// Count returns the number of variables
func (vs *VariableSet) Count() int {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	return len(vs.variables)
}

// Clear removes all variables
func (vs *VariableSet) Clear() {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	vs.variables = make(map[string]any)
}

// ToMap returns variables as a map (alias for All)
func (vs *VariableSet) ToMap() map[string]any {
	return vs.All()
}

// SetSchema sets the schema for the variable set
func (vs *VariableSet) SetSchema(schema *VariableSchema) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	// Validate existing variables against new schema
	if schema != nil {
		if err := schema.Validate(vs.variables); err != nil {
			return err
		}
	}

	vs.schema = schema
	return nil
}

// GetSchema returns the current schema
func (vs *VariableSet) GetSchema() *VariableSchema {
	return vs.schema
}

// SetReadOnly makes the VariableSet read-only or writable
func (vs *VariableSet) SetReadOnly(readOnly bool) {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	vs.readOnly = readOnly
}

// IsReadOnly returns whether the VariableSet is read-only
func (vs *VariableSet) IsReadOnly() bool {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	return vs.readOnly
}

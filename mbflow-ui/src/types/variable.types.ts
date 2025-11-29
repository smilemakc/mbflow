/**
 * Variable types - for variable tracking and context management
 */

import type { VariableType } from './domain.types'

// ============================================================================
// Variable Definition
// ============================================================================

export interface VariableDefinition {
    name: string
    type: VariableType
    scope: 'global' | 'node' | 'edge'
    readOnly?: boolean
    description?: string
    children?: VariableDefinition[] // For nested objects
    sourceNodeId?: string // Which node produced this variable
    sourceNodeName?: string
}

// ============================================================================
// Variable Context (for a specific node)
// ============================================================================

export interface VariableContext {
    nodeId: string
    nodeName: string
    availableVariables: VariableDefinition[]
    globalContext: Record<string, unknown>
    parentOutputs: Record<string, Record<string, unknown>> // parentNodeName -> outputs
}

// ============================================================================
// Variable Schema (output schema for node types)
// ============================================================================

export interface VariableSchema {
    name: string
    type: VariableType
    required?: boolean
    description?: string
    properties?: Record<string, VariableSchema> // For objects
    items?: VariableSchema // For arrays
}

// ============================================================================
// Variable Autocomplete
// ============================================================================

export interface VariableAutocompleteItem {
    label: string // Display text
    value: string // Actual value to insert
    type: VariableType
    description?: string
    icon?: string
    scope: 'global' | 'node' | 'edge'
}

// ============================================================================
// Template Expression
// ============================================================================

export interface TemplateExpression {
    raw: string // Original template string
    variables: string[] // Extracted variable paths (e.g., ["user.id", "GlobalContext.api_key"])
    isValid: boolean
    errors?: string[]
}

// ============================================================================
// Variable Validation
// ============================================================================

export interface VariableValidationResult {
    isValid: boolean
    errors: Array<{
        variable: string
        message: string
    }>
}

import type { Node, Edge, VariableDefinition } from '@/types'
import { NodeTypes, VariableTypes } from '@/types'

/**
 * Resolves available variables for a specific node in the workflow.
 * Traverses the graph backwards to find all predecessors and their outputs.
 */
export function resolveAvailableVariables(
    targetNodeId: string,
    nodes: Node[],
    edges: Edge[]
): VariableDefinition[] {
    const variables: VariableDefinition[] = []

    // Find direct predecessors first
    const directPredecessors = edges
        .filter((e) => e.to === targetNodeId)
        .map((e) => e.from)

    // For each predecessor, determine its output variables
    for (const predId of directPredecessors) {
        const node = nodes.find((n) => n.id === predId)
        if (!node) continue

        const nodeVars = getNodeOutputVariables(node)
        variables.push(...nodeVars)
    }

    // Add global/system variables
    variables.push(
        {
            name: 'workflow.id',
            type: VariableTypes.STRING,
            description: 'Current workflow ID',
            sourceNodeId: 'system',
            scope: 'global',
        },
        {
            name: 'execution.id',
            type: VariableTypes.STRING,
            description: 'Current execution ID',
            sourceNodeId: 'system',
            scope: 'global',
        },
        {
            name: 'execution.started_at',
            type: VariableTypes.STRING,
            description: 'Execution start timestamp',
            sourceNodeId: 'system',
            scope: 'global',
        }
    )

    return variables
}

/**
 * Determines output variables based on node type and config
 */
function getNodeOutputVariables(node: Node): VariableDefinition[] {
    const prefix = node.name || node.id
    const variables: VariableDefinition[] = []

    switch (node.type) {
        case NodeTypes.TRANSFORM:
            if (node.config?.transformations) {
                Object.keys(node.config.transformations).forEach((key) => {
                    variables.push({
                        name: `${prefix}.${key}`,
                        type: VariableTypes.ANY,
                        description: `Transformation result: ${key}`,
                        sourceNodeId: node.id,
                        scope: 'node',
                    })
                })
            }
            break

        case NodeTypes.HTTP:
            variables.push(
                {
                    name: `${prefix}.status`,
                    type: VariableTypes.INT,
                    description: 'HTTP status code',
                    sourceNodeId: node.id,
                    scope: 'node',
                },
                {
                    name: `${prefix}.body`,
                    type: VariableTypes.ANY,
                    description: 'Response body',
                    sourceNodeId: node.id,
                    scope: 'node',
                },
                {
                    name: `${prefix}.headers`,
                    type: VariableTypes.OBJECT,
                    description: 'Response headers',
                    sourceNodeId: node.id,
                    scope: 'node',
                }
            )
            break

        case NodeTypes.JSON_PARSER:
            variables.push({
                name: `${prefix}.data`,
                type: VariableTypes.OBJECT,
                description: 'Parsed JSON data',
                sourceNodeId: node.id,
                scope: 'node',
            })
            break

        case NodeTypes.OPENAI_COMPLETION:
            variables.push(
                {
                    name: `${prefix}.text`,
                    type: VariableTypes.STRING,
                    description: 'Completion text',
                    sourceNodeId: node.id,
                    scope: 'node',
                },
                {
                    name: `${prefix}.usage`,
                    type: VariableTypes.OBJECT,
                    description: 'Token usage',
                    sourceNodeId: node.id,
                    scope: 'node',
                }
            )
            break

        default:
            // Generic output for other nodes
            variables.push({
                name: `${prefix}.output`,
                type: VariableTypes.ANY,
                description: 'Node output',
                sourceNodeId: node.id,
                scope: 'node',
            })
    }

    return variables
}

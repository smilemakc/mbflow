import { useWorkflowStore } from "@/stores/workflow";
import type { VariableContext } from "@/types/template";
import { NODE_OUTPUT_SCHEMAS } from "@/data/nodeSchemas";

export function useVariableContext() {
  const workflowStore = useWorkflowStore();

  function getContext(selectedNodeId?: string): VariableContext {
    return {
      workflowVars: workflowStore.workflowVariables || {},
      executionVars: {}, // Not available at design time
      inputVars: getParentOutputs(selectedNodeId),
    };
  }

  function getParentOutputs(_nodeId?: string): Record<string, any> {
    // At design time, we don't have actual parent outputs
    // Return empty object with placeholder info
    // In the future, this could return sample/mock data
    return {};
  }

  function getAvailableVariables(targetNodeId?: string): {
    workflow: Array<{ key: string; value: any }>;
    input: Array<{ key: string; description: string; type?: string }>;
  } {
    const workflowVars = workflowStore.workflowVariables || {};

    // If no target node is specified, try to use the selected one
    const nodeId = targetNodeId || workflowStore.selectedNodeId;
    const inputVars: Array<{ key: string; description: string; type?: string }> =
      [];

    if (nodeId) {
      // Find incoming edges to this node
      const incomingEdges = workflowStore.edges.filter(
        (edge) => edge.target === nodeId,
      );

      if (incomingEdges.length === 1 && incomingEdges[0]) {
        // Single parent case - direct access
        const parentId = incomingEdges[0].source;
        const parentNode = workflowStore.nodes.find((n) => n.id === parentId);

        if (parentNode) {
          addNodeOutputsToVars(parentNode, inputVars, "");
        }
      } else if (incomingEdges.length > 1) {
        // Multiple parents case - access via parentN
        // Sort edges by ID for deterministic ordering
        const sortedEdges = [...incomingEdges].sort((a, b) =>
          a.id.localeCompare(b.id),
        );

        sortedEdges.forEach((edge, index) => {
          const parentId = edge.source;
          const parentNode = workflowStore.nodes.find((n) => n.id === parentId);
          // 1-based index for user friendliness: parent1, parent2...
          const prefix = `parent${index + 1}`;

          if (parentNode) {
            // Add the parent object itself as a hint
            inputVars.push({
              key: prefix,
              description: `Output from ${parentNode.data.label || parentId}`,
              type: "object",
            });

            // Add its fields
            addNodeOutputsToVars(parentNode, inputVars, `${prefix}.`);
          }
        });
      }
    }

    return {
      workflow: Object.entries(workflowVars).map(([key, value]) => ({
        key,
        value,
      })),
      input: inputVars,
    };
  }

  function addNodeOutputsToVars(
    node: any,
    vars: Array<{ key: string; description: string; type?: string }>,
    prefix: string,
  ) {
    // Get schema for node type
    const type = node.type;
    const schema = NODE_OUTPUT_SCHEMAS[type];

    if (schema) {
      Object.entries(schema).forEach(([key, field]) => {
        if (typeof field === "string") {
          vars.push({
            key: `${prefix}${key}`,
            description: field,
            type: "string",
          });
        } else {
          vars.push({
            key: `${prefix}${key}`,
            description: field.description || "",
            type: field.type,
          });
        }
      });
    } else {
      // Fallback for unknown schemas
      vars.push({
        key: `${prefix}*`,
        description: `Any output from ${node.type}`,
        type: "any",
      });
    }
  }

  return {
    getContext,
    getAvailableVariables,
  };
}

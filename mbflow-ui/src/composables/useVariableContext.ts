import { useWorkflowStore } from "@/stores/workflow";
import type { VariableContext } from "@/types/template";

export function useVariableContext() {
  const workflowStore = useWorkflowStore();

  function getContext(selectedNodeId?: string): VariableContext {
    return {
      workflowVars: workflowStore.workflowVariables || {},
      executionVars: {}, // Not available at design time
      inputVars: getParentOutputs(selectedNodeId),
    };
  }

  function getParentOutputs(nodeId?: string): Record<string, any> {
    // At design time, we don't have actual parent outputs
    // Return empty object with placeholder info
    // In the future, this could return sample/mock data
    return {};
  }

  function getAvailableVariables(): {
    workflow: Array<{ key: string; value: any }>;
    input: Array<{ key: string; description: string }>;
  } {
    const workflowVars = workflowStore.workflowVariables || {};

    return {
      workflow: Object.entries(workflowVars).map(([key, value]) => ({
        key,
        value,
      })),
      input: [
        {
          key: "*",
          description: "Parent node output (available at runtime)",
        },
      ],
    };
  }

  return {
    getContext,
    getAvailableVariables,
  };
}

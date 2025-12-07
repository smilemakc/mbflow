/**
 * Composable for animating workflow execution on canvas
 */

import { useWorkflowStore } from "@/stores/workflow";
import type { ExecutionEvent } from "./useExecutionObserver";

export function useWorkflowExecutionAnimation() {
  const workflowStore = useWorkflowStore();

  /**
   * Apply animation to a node based on its execution state
   */
  function animateNode(
    nodeId: string,
    state: "running" | "completed" | "failed",
  ) {
    const node = workflowStore.nodes.find((n) => n.id === nodeId);
    if (!node) {
      console.warn("[Animation] Node not found:", nodeId);
      return;
    }

    console.log("[Animation] Animating node:", nodeId, "state:", state);

    // Add animation class to node data
    if (!node.data) node.data = {};

    switch (state) {
      case "running":
        node.data.executionState = "running";
        node.data.animated = true;
        break;
      case "completed":
        node.data.executionState = "completed";
        node.data.animated = false;
        break;
      case "failed":
        node.data.executionState = "failed";
        node.data.animated = false;
        break;
    }

    // Trigger reactivity
    workflowStore.updateNode(nodeId, node.data);
    console.log("[Animation] Node updated:", nodeId, node.data);
  }

  /**
   * Animate edge between two nodes
   */
  function animateEdge(
    sourceId: string,
    targetId: string,
    state: "active" | "completed",
  ) {
    const edge = workflowStore.edges.find(
      (e) => e.source === sourceId && e.target === targetId,
    );

    if (!edge) return;

    // Add animation to edge
    if (state === "active") {
      edge.animated = true;
      edge.style = { stroke: "#3b82f6", strokeWidth: 2 };
    } else if (state === "completed") {
      edge.animated = false;
      edge.style = { stroke: "#10b981", strokeWidth: 2 };
    }
  }

  /**
   * Reset all animations
   */
  function resetAnimations() {
    // Reset all nodes
    workflowStore.nodes.forEach((node) => {
      if (node.data) {
        node.data.executionState = undefined;
        node.data.animated = false;
        workflowStore.updateNode(node.id, node.data);
      }
    });

    // Reset all edges
    workflowStore.edges.forEach((edge) => {
      edge.animated = false;
      edge.style = undefined;
    });
  }

  /**
   * Handle execution events and animate accordingly
   */
  function handleExecutionEvent(event: ExecutionEvent) {
    if (event.type !== "event" || !event.event) return;

    const { event: execEvent } = event;

    switch (execEvent.event_type) {
      case "execution.started":
        resetAnimations();
        break;

      case "node.started":
        if (execEvent.node_id) {
          animateNode(execEvent.node_id, "running");

          // Store input data if available
          if (execEvent.input) {
            const node = workflowStore.nodes.find(
              (n) => n.id === execEvent.node_id,
            );
            if (node && node.data) {
              node.data.executionInput = execEvent.input;
              workflowStore.updateNode(execEvent.node_id, node.data);
            }
          }
        }
        break;

      case "node.completed":
        if (execEvent.node_id) {
          animateNode(execEvent.node_id, "completed");

          // Store output data if available
          if (execEvent.output) {
            const node = workflowStore.nodes.find(
              (n) => n.id === execEvent.node_id,
            );
            if (node && node.data) {
              node.data.executionOutput = execEvent.output;
              workflowStore.updateNode(execEvent.node_id, node.data);
            }
          }

          // Mark incoming edges as completed (green)
          // The edge connecting Previous -> Current is now fully traversed
          const incomingEdges = workflowStore.edges.filter(
            (e) => e.target === execEvent.node_id,
          );
          incomingEdges.forEach((edge) => {
            animateEdge(edge.source, edge.target, "completed");
          });

          // Animate outgoing edges as active (blue/animated)
          const outgoingEdges = workflowStore.edges.filter(
            (e) => e.source === execEvent.node_id,
          );
          outgoingEdges.forEach((edge) => {
            animateEdge(edge.source, edge.target, "active");
          });
        }
        break;

      case "node.failed":
        if (execEvent.node_id) {
          animateNode(execEvent.node_id, "failed");

          // Store error data if available
          if (execEvent.error) {
            const node = workflowStore.nodes.find(
              (n) => n.id === execEvent.node_id,
            );
            if (node && node.data) {
              node.data.executionError = execEvent.error;
              workflowStore.updateNode(execEvent.node_id, node.data);
            }
          }
        }
        break;

      case "execution.completed":
      case "execution.failed":
        // Keep final state but stop animations
        workflowStore.nodes.forEach((node) => {
          if (node.data) {
            node.data.animated = false;
          }
        });
        workflowStore.edges.forEach((edge) => {
          edge.animated = false;
        });
        break;
    }
  }

  return {
    animateNode,
    animateEdge,
    resetAnimations,
    handleExecutionEvent,
  };
}

// @ts-nocheck
import { ref } from "vue";
import type { Node as VueFlowNode, Edge as VueFlowEdge } from "@vue-flow/core";
import ELK, { type ElkNode } from "elkjs/lib/elk.bundled.js";
import dagre from "dagre";

export type LayoutAlgorithm = "elk" | "dagre";

export interface LayoutOptions {
  direction?: "LR" | "TB" | "RL" | "BT";
  spacing?: {
    node?: number;
    rank?: number;
  };
}

const defaultOptions: LayoutOptions = {
  direction: "TB",
  spacing: {
    node: 50,
    rank: 100,
  },
};

export function useAutoLayout() {
  const isLayouting = ref(false);

  /**
   * Apply ELK layout algorithm
   */
  async function layoutWithELK(
    nodes: VueFlowNode[],
    edges: VueFlowEdge[],
    options: LayoutOptions = defaultOptions,
  ): Promise<{ nodes: VueFlowNode[]; edges: VueFlowEdge[] }> {
    isLayouting.value = true;

    try {
      const elk = new ELK();

      const elkNodes: ElkNode["children"] = nodes.map((node) => ({
        id: node.id,
        width: node.measured?.width || 200,
        height: node.measured?.height || 100,
      }));

      const elkEdges = edges.map((edge) => ({
        id: edge.id,
        sources: [edge.source],
        targets: [edge.target],
      }));

      const graph: ElkNode = {
        id: "root",
        layoutOptions: {
          "elk.algorithm": "layered",
          "elk.direction": options.direction || "DOWN",
          "elk.spacing.nodeNode": String(options.spacing?.node || 50),
          "elk.layered.spacing.nodeNodeBetweenLayers": String(
            options.spacing?.rank || 100,
          ),
          "elk.spacing.edgeNode": "40",
          "elk.spacing.edgeEdge": "20",
        },
        children: elkNodes,
        edges: elkEdges,
      };

      const layout = await elk.layout(graph);

      const layoutedNodes = nodes.map((node) => {
        const elkNode = layout.children?.find((n) => n.id === node.id);
        if (!elkNode) return node;

        return {
          ...node,
          position: {
            x: elkNode.x || 0,
            y: elkNode.y || 0,
          },
        };
      });

      return { nodes: layoutedNodes, edges };
    } catch (error) {
      console.error("ELK layout failed:", error);
      return { nodes, edges };
    } finally {
      isLayouting.value = false;
    }
  }

  /**
   * Apply dagre layout algorithm
   */
  function layoutWithDagre(
    nodes: VueFlowNode[],
    edges: VueFlowEdge[],
    options: LayoutOptions = defaultOptions,
  ): { nodes: VueFlowNode[]; edges: VueFlowEdge[] } {
    isLayouting.value = true;

    try {
      const dagreGraph = new dagre.graphlib.Graph();
      dagreGraph.setDefaultEdgeLabel(() => ({}));

      const direction = options.direction || "TB";
      const nodeSpacing = options.spacing?.node || 50;
      const rankSpacing = options.spacing?.rank || 100;

      dagreGraph.setGraph({
        rankdir: direction,
        nodesep: nodeSpacing,
        ranksep: rankSpacing,
        edgesep: 20,
      });

      // Add nodes
      nodes.forEach((node) => {
        dagreGraph.setNode(node.id, {
          width: node.measured?.width || 200,
          height: node.measured?.height || 100,
        });
      });

      // Add edges
      edges.forEach((edge) => {
        dagreGraph.setEdge(edge.source, edge.target);
      });

      // Calculate layout
      dagre.layout(dagreGraph);

      // Apply positions
      const layoutedNodes = nodes.map((node) => {
        const position = dagreGraph.node(node.id);
        return {
          ...node,
          position: {
            x: position.x - (node.measured?.width || 200) / 2,
            y: position.y - (node.measured?.height || 100) / 2,
          },
        };
      });

      return { nodes: layoutedNodes, edges };
    } catch (error) {
      console.error("Dagre layout failed:", error);
      return { nodes, edges };
    } finally {
      isLayouting.value = false;
    }
  }

  /**
   * Apply layout algorithm based on selected type
   */
  async function applyLayout(
    nodes: VueFlowNode[],
    edges: VueFlowEdge[],
    algorithm: LayoutAlgorithm = "elk",
    options?: LayoutOptions,
  ): Promise<{ nodes: VueFlowNode[]; edges: VueFlowEdge[] }> {
    if (algorithm === "elk") {
      return await layoutWithELK(nodes, edges, options);
    } else {
      return layoutWithDagre(nodes, edges, options);
    }
  }

  return {
    isLayouting,
    applyLayout,
    layoutWithELK,
    layoutWithDagre,
  };
}

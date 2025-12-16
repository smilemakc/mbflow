import { useState, useCallback } from 'react';
import type { Node, Edge } from 'reactflow';
import dagre from 'dagre';

export type LayoutDirection = 'TB' | 'LR' | 'BT' | 'RL';

export interface UseAutoLayoutOptions {
  nodeSpacing?: number;
  rankSpacing?: number;
}

export interface UseAutoLayout {
  applyLayout: (nodes: Node[], edges: Edge[], direction?: LayoutDirection) => { nodes: Node[]; edges: Edge[] };
  isLayouting: boolean;
}

const DEFAULT_NODE_WIDTH = 200;
const DEFAULT_NODE_HEIGHT = 80;
const DEFAULT_NODE_SPACING = 50;
const DEFAULT_RANK_SPACING = 100;

export function useAutoLayout(options: UseAutoLayoutOptions = {}): UseAutoLayout {
  const [isLayouting, setIsLayouting] = useState(false);

  const {
    nodeSpacing = DEFAULT_NODE_SPACING,
    rankSpacing = DEFAULT_RANK_SPACING,
  } = options;

  const applyLayout = useCallback(
    (nodes: Node[], edges: Edge[], direction: LayoutDirection = 'TB'): { nodes: Node[]; edges: Edge[] } => {
      setIsLayouting(true);

      try {
        const dagreGraph = new dagre.graphlib.Graph();
        dagreGraph.setDefaultEdgeLabel(() => ({}));

        dagreGraph.setGraph({
          rankdir: direction,
          nodesep: nodeSpacing,
          ranksep: rankSpacing,
          edgesep: 20,
        });

        nodes.forEach((node) => {
          const nodeAny = node as any;
          const width = (nodeAny.measured?.width || node.width || DEFAULT_NODE_WIDTH) as number;
          const height = (nodeAny.measured?.height || node.height || DEFAULT_NODE_HEIGHT) as number;

          dagreGraph.setNode(node.id, {
            width,
            height,
          });
        });

        edges.forEach((edge) => {
          dagreGraph.setEdge(edge.source, edge.target);
        });

        dagre.layout(dagreGraph);

        const layoutedNodes = nodes.map((node) => {
          const position = dagreGraph.node(node.id);
          const nodeAny = node as any;
          const width = (nodeAny.measured?.width || node.width || DEFAULT_NODE_WIDTH) as number;
          const height = (nodeAny.measured?.height || node.height || DEFAULT_NODE_HEIGHT) as number;

          return {
            ...node,
            position: {
              x: position.x - width / 2,
              y: position.y - height / 2,
            },
          };
        });

        return { nodes: layoutedNodes, edges };
      } catch (error) {
        console.error('Dagre layout failed:', error);
        return { nodes, edges };
      } finally {
        setIsLayouting(false);
      }
    },
    [nodeSpacing, rankSpacing]
  );

  return {
    applyLayout,
    isLayouting,
  };
}

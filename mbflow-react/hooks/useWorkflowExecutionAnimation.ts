import { useCallback } from 'react';
import { useDagStore } from '@/store/dagStore';
import type { ExecutionEvent } from '@/types/execution';

export function useWorkflowExecutionAnimation() {
  const nodes = useDagStore((state) => state.nodes);
  const edges = useDagStore((state) => state.edges);
  const updateNodeData = useDagStore((state) => state.updateNodeData);

  const animateNode = useCallback(
    (nodeId: string, state: 'running' | 'completed' | 'failed') => {
      const node = nodes.find((n) => n.id === nodeId);
      if (!node) {
        console.warn('[Animation] Node not found:', nodeId);
        return;
      }

      console.log('[Animation] Animating node:', nodeId, 'state:', state);

      switch (state) {
        case 'running':
          updateNodeData(nodeId, {
            executionState: 'running',
            animated: true,
          });
          break;
        case 'completed':
          updateNodeData(nodeId, {
            executionState: 'completed',
            animated: false,
          });
          break;
        case 'failed':
          updateNodeData(nodeId, {
            executionState: 'failed',
            animated: false,
          });
          break;
      }

      console.log('[Animation] Node updated:', nodeId);
    },
    [nodes, updateNodeData]
  );

  const animateEdge = useCallback(
    (sourceId: string, targetId: string, state: 'active' | 'completed') => {
      const store = useDagStore.getState();
      const edge = store.edges.find(
        (e) => e.source === sourceId && e.target === targetId
      );

      if (!edge) return;

      if (state === 'active') {
        edge.animated = true;
        edge.style = { stroke: '#3b82f6', strokeWidth: 2 };
      } else if (state === 'completed') {
        edge.animated = false;
        edge.style = { stroke: '#10b981', strokeWidth: 2 };
      }

      useDagStore.setState({ edges: [...store.edges] });
    },
    []
  );

  const resetAnimations = useCallback(() => {
    const store = useDagStore.getState();

    store.nodes.forEach((node) => {
      if (node.data) {
        updateNodeData(node.id, {
          executionState: undefined,
          animated: false,
        });
      }
    });

    const resetEdges = store.edges.map((edge) => ({
      ...edge,
      animated: false,
      style: undefined,
    }));

    useDagStore.setState({ edges: resetEdges });
  }, [updateNodeData]);

  const handleExecutionEvent = useCallback(
    (event: ExecutionEvent) => {
      if (event.type !== 'event' || !event.event) return;

      const { event: execEvent } = event;

      switch (execEvent.event_type) {
        case 'execution.started':
          resetAnimations();
          break;

        case 'node.started':
          if (execEvent.node_id) {
            animateNode(execEvent.node_id, 'running');

            if (execEvent.input) {
              const node = nodes.find((n) => n.id === execEvent.node_id);
              if (node && node.data) {
                updateNodeData(execEvent.node_id, {
                  executionInput: execEvent.input,
                });
              }
            }
          }
          break;

        case 'node.completed':
          if (execEvent.node_id) {
            animateNode(execEvent.node_id, 'completed');

            if (execEvent.output) {
              const node = nodes.find((n) => n.id === execEvent.node_id);
              if (node && node.data) {
                updateNodeData(execEvent.node_id, {
                  executionOutput: execEvent.output,
                });
              }
            }

            const incomingEdges = edges.filter(
              (e) => e.target === execEvent.node_id
            );
            incomingEdges.forEach((edge) => {
              animateEdge(edge.source, edge.target, 'completed');
            });

            const outgoingEdges = edges.filter(
              (e) => e.source === execEvent.node_id
            );
            outgoingEdges.forEach((edge) => {
              animateEdge(edge.source, edge.target, 'active');
            });
          }
          break;

        case 'node.failed':
          if (execEvent.node_id) {
            animateNode(execEvent.node_id, 'failed');

            if (execEvent.error) {
              const node = nodes.find((n) => n.id === execEvent.node_id);
              if (node && node.data) {
                updateNodeData(execEvent.node_id, {
                  executionError: execEvent.error,
                });
              }
            }
          }
          break;

        case 'execution.completed':
        case 'execution.failed':
          nodes.forEach((node) => {
            if (node.data && node.data.animated) {
              updateNodeData(node.id, {
                animated: false,
              });
            }
          });

          const store = useDagStore.getState();
          const stoppedEdges = store.edges.map((edge) => ({
            ...edge,
            animated: false,
          }));
          useDagStore.setState({ edges: stoppedEdges });
          break;
      }
    },
    [nodes, edges, animateNode, animateEdge, resetAnimations, updateNodeData]
  );

  return {
    animateNode,
    animateEdge,
    resetAnimations,
    handleExecutionEvent,
  };
}

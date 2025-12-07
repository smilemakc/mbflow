import { create } from 'zustand';
import {
  Connection,
  EdgeChange,
  NodeChange,
  addEdge,
  applyNodeChanges,
  applyEdgeChanges,
  MarkerType,
  getIncomers
} from 'reactflow';
import {
  AppNode,
  AppEdge,
  DAGHistoryState,
  Variable,
  NodeType,
  NodeStatus,
  VariableType,
  VariableSource,
  ExecutionLog,
  NodeExecutionResult
} from '@/types';
import type { ExecutionEvent } from '@/types/execution';
import { workflowService } from '@/services/workflowService';
import { executionService } from '@/services/executionService';
import { executionWS } from '@/services/executionWebSocket';
import { toast } from '../lib/toast';

interface DAGState {
  // === Graph State ===
  nodes: AppNode[];
  edges: AppEdge[];
  dagId: string; // Added to track backend ID
  dagName: string;

  // === Selection ===
  selectedNodeId: string | null;
  selectedEdgeId: string | null;

  // === History ===
  history: DAGHistoryState[];
  historyIndex: number;

  // === Meta ===
  isDirty: boolean;
  lastSavedAt: Date | null;
  isLoading: boolean;
  isSaving: boolean;
  executionId: string | null;

  // === Execution & Monitoring ===
  isRunning: boolean;
  logs: ExecutionLog[];
  executionResults: Record<string, NodeExecutionResult>; // Map nodeId -> result

  // === WebSocket ===
  wsConnected: boolean;

  // === Actions ===
  onNodesChange: (changes: NodeChange[]) => void;
  onEdgesChange: (changes: EdgeChange[]) => void;
  onConnect: (connection: Connection) => void;
  addNode: (type: NodeType, position: { x: number, y: number }) => AppNode;
  duplicateNode: (nodeId: string) => void;
  deleteNode: (nodeId: string) => void;
  setSelectedNodeId: (id: string | null) => void;
  setSelectedEdgeId: (id: string | null) => void;
  updateNodeData: (nodeId: string, data: Record<string, any>) => void;
  updateEdge: (edgeId: string, updates: Partial<AppEdge>) => void;
  deleteEdge: (edgeId: string) => void;
  loadGraph: (nodes: AppNode[], edges: AppEdge[]) => void;

  // === Variables ===
  variables: Variable[];
  workflowVariables: Record<string, string>;
  addVariable: (variable: Variable) => void;
  updateVariable: (key: string, value: string) => void;
  deleteVariable: (key: string) => void;
  updateWorkflowVariables: (variables: Record<string, string>) => void;
  getAvailableVariables: () => Variable[];

  // === DAG Name ===
  setDAGName: (name: string) => void;

  // === Persistence & Execution ===
  saveDAG: () => Promise<void>;
  createNewWorkflow: (name: string) => Promise<void>;
  fetchWorkflow: (id: string) => Promise<void>;
  fetchWorkflowList: () => Promise<{ id: string; name: string }[]>;
  runWorkflow: () => Promise<void>;
  undo: () => void;
  redo: () => void;
  clearExecution: () => void;

  // === WebSocket Actions ===
  connectToExecution: (executionId: string) => void;
  disconnectFromExecution: () => void;
  handleExecutionEvent: (event: ExecutionEvent) => void;
}

// Start with empty state - load from API
export const useDagStore = create<DAGState>((set, get) => ({
  nodes: [],
  edges: [],
  dagId: '',
  dagName: 'New Workflow',
  selectedNodeId: null,
  selectedEdgeId: null,
  history: [{ nodes: [], edges: [] }],
  historyIndex: 0,
  isDirty: false,
  lastSavedAt: null,
  isLoading: false,
  isSaving: false,
  executionId: null,
  variables: [],
  workflowVariables: {},

  // Execution State
  isRunning: false,
  logs: [],
  executionResults: {},

  // WebSocket State
  wsConnected: false,

  onNodesChange: (changes) => {
    set({
      nodes: applyNodeChanges(changes, get().nodes),
      isDirty: true,
    });
  },

  onEdgesChange: (changes) => {
    set({
      edges: applyEdgeChanges(changes, get().edges),
      isDirty: true,
    });
  },

  onConnect: (connection) => {
    set({
      edges: addEdge({
        ...connection,
        type: 'smoothstep',
        markerEnd: { type: MarkerType.ArrowClosed },
        animated: false
      }, get().edges),
      isDirty: true,
    });
    const { nodes, edges, history, historyIndex } = get();
    const newHistory = history.slice(0, historyIndex + 1);
    newHistory.push({ nodes, edges });
    set({ history: newHistory, historyIndex: newHistory.length - 1 });
  },

  addNode: (type, position) => {
    const { nodes } = get();

    // Generate node ID based on type + suffix
    const generateNodeId = (nodeType: NodeType): string => {
      // Count existing nodes of this type
      const existingCount = nodes.filter(n => n.data.type === nodeType).length;
      // If first node of this type, use just the type, otherwise add suffix
      return existingCount === 0 ? nodeType : `${nodeType}_${existingCount + 1}`;
    };

    // Generate default label based on type
    const getDefaultLabel = (nodeType: NodeType): string => {
      const labelMap: Record<string, string> = {
        [NodeType.TELEGRAM]: 'Telegram Send',
        [NodeType.TELEGRAM_DOWNLOAD]: 'TG Download',
        [NodeType.TELEGRAM_PARSE]: 'TG Parse',
        [NodeType.TELEGRAM_CALLBACK]: 'TG Callback',
        [NodeType.HTTP]: 'HTTP Request',
        [NodeType.LLM]: 'LLM',
        [NodeType.DELAY]: 'Delay',
        [NodeType.CONDITIONAL]: 'Condition',
        [NodeType.TRANSFORM]: 'Transform',
        [NodeType.FUNCTION_CALL]: 'Function Call',
        [NodeType.FILE_STORAGE]: 'File Storage',
        [NodeType.MERGE]: 'Merge',
        [NodeType.BASE64_TO_BYTES]: 'Base64 → Bytes',
        [NodeType.BYTES_TO_BASE64]: 'Bytes → Base64',
        [NodeType.STRING_TO_JSON]: 'String → JSON',
        [NodeType.JSON_TO_STRING]: 'JSON → String',
        [NodeType.BYTES_TO_JSON]: 'Bytes → JSON',
        [NodeType.FILE_TO_BYTES]: 'File → Bytes',
        [NodeType.BYTES_TO_FILE]: 'Bytes → File',
      };
      return labelMap[nodeType] || 'New Node';
    };

    const newNode: AppNode = {
      id: generateNodeId(type),
      type: 'custom',
      position,
      data: {
        label: getDefaultLabel(type),
        type,
        status: NodeStatus.IDLE,
        description: 'Configure this node in the panel.',
        config: {}
      },
    };

    const newNodes = [...get().nodes, newNode];
    set({ nodes: newNodes, isDirty: true });

    const { edges, history, historyIndex } = get();
    const newHistory = history.slice(0, historyIndex + 1);
    newHistory.push({ nodes: newNodes, edges });
    set({ history: newHistory, historyIndex: newHistory.length - 1 });

    return newNode;
  },

  duplicateNode: (nodeId) => {
    const { nodes, addNode } = get();
    const nodeToDuplicate = nodes.find(n => n.id === nodeId);

    if (nodeToDuplicate) {
      const position = {
        x: nodeToDuplicate.position.x + 50,
        y: nodeToDuplicate.position.y + 50,
      };

      const newNode = addNode(nodeToDuplicate.data.type as NodeType, position);

      set(state => ({
        nodes: state.nodes.map(n => n.id === newNode.id ? {
          ...n,
          data: {
            ...n.data,
            label: `${nodeToDuplicate.data.label} (Copy)`,
            description: nodeToDuplicate.data.description,
            config: JSON.parse(JSON.stringify(nodeToDuplicate.data.config || {})),
          }
        } : n)
      }));
    }
  },

  deleteNode: (nodeId) => {
    const newNodes = get().nodes.filter((n) => n.id !== nodeId);
    const newEdges = get().edges.filter((e) => e.source !== nodeId && e.target !== nodeId);
    set({ nodes: newNodes, edges: newEdges, selectedNodeId: null, isDirty: true });

    const { history, historyIndex } = get();
    const newHistory = history.slice(0, historyIndex + 1);
    newHistory.push({ nodes: newNodes, edges: newEdges });
    set({ history: newHistory, historyIndex: newHistory.length - 1 });
  },

  setSelectedNodeId: (id) => set({ selectedNodeId: id }),

  setSelectedEdgeId: (id) => set({ selectedEdgeId: id }),

  updateEdge: (edgeId, updates) => {
    set({
      edges: get().edges.map((edge) => {
        if (edge.id === edgeId) {
          return { ...edge, ...updates };
        }
        return edge;
      }),
      isDirty: true,
    });
  },

  deleteEdge: (edgeId) => {
    const newEdges = get().edges.filter((e) => e.id !== edgeId);
    set({ edges: newEdges, selectedEdgeId: null, isDirty: true });

    const { nodes, history, historyIndex } = get();
    const newHistory = history.slice(0, historyIndex + 1);
    newHistory.push({ nodes, edges: newEdges });
    set({ history: newHistory, historyIndex: newHistory.length - 1 });
  },

  updateNodeData: (nodeId, data) => {
    set({
      nodes: get().nodes.map((node) => {
        if (node.id === nodeId) {
          return { ...node, data: { ...node.data, ...data } };
        }
        return node;
      }),
      isDirty: true,
    });
  },

  loadGraph: (nodes, edges) => {
    set({
      nodes,
      edges,
      isDirty: false,
      history: [{ nodes, edges }],
      historyIndex: 0,
      selectedNodeId: null,
      logs: [],
      executionResults: {}
    });
  },

  addVariable: (variable) => {
    set({ variables: [...get().variables, variable] });
  },

  getAvailableVariables: () => {
    const { nodes, variables } = get();

    const systemVars: Variable[] = [
      { id: 'sys_1', key: 'execution_id', name: 'Execution ID', type: VariableType.STRING, source: VariableSource.GLOBAL },
      { id: 'sys_2', key: 'timestamp', name: 'Timestamp', type: VariableType.STRING, source: VariableSource.GLOBAL },
    ];

    const nodeVars: Variable[] = nodes.map(node => ({
      id: `node_${node.id}`,
      key: `step_${node.id}.output`,
      name: `${node.data.label} Output`,
      type: VariableType.OBJECT,
      source: VariableSource.NODE,
      nodeId: node.id
    }));

    return [...systemVars, ...variables, ...nodeVars];
  },

  updateVariable: (key, value) => {
    set(state => ({
      workflowVariables: { ...state.workflowVariables, [key]: value },
      isDirty: true
    }));
  },

  deleteVariable: (key) => {
    set(state => {
      const newVars = { ...state.workflowVariables };
      delete newVars[key];
      return { workflowVariables: newVars, isDirty: true };
    });
  },

  updateWorkflowVariables: (variables) => {
    set({ workflowVariables: variables, isDirty: true });
  },

  setDAGName: (name: string) => {
    set({ dagName: name, isDirty: true });
  },

  // === Persistence with Backend Support ===

  saveDAG: async () => {
    const { nodes, edges, dagId, dagName } = get();

    try {
      // Optimistic UI update
      set({ isDirty: false, lastSavedAt: new Date() });

      // Attempt to save to backend
      const payload = {
        id: dagId,
        name: dagName,
        nodes,
        edges,
      variables: get().workflowVariables
      };

      console.log('Saving to backend...', payload);
      await workflowService.save(payload);
      console.log('Backend save successful');

    } catch (error) {
      console.error('Failed to save to backend:', error);
      // Fallback: If network fails, we still consider it "saved" locally for the session,
      // but in a real app you might show an error toast here.
    }
  },

  fetchWorkflow: async (id: string) => {
    set({ isLoading: true });
    try {
      const dag = await workflowService.getById(id);
      if (dag) {
        // Convert variables array back to Record<string, string>
        const workflowVars: Record<string, string> = {};
        if (dag.variables && Array.isArray(dag.variables)) {
          dag.variables.forEach((v: any) => {
            workflowVars[v.key] = String(v.value ?? '');
          });
        }

        set({
          dagId: dag.id,
          dagName: dag.name,
          nodes: dag.nodes,
          edges: dag.edges,
          workflowVariables: workflowVars,
          isDirty: false,
          isLoading: false,
          history: [{ nodes: dag.nodes, edges: dag.edges }],
          historyIndex: 0
        });
      }
    } catch (error) {
      console.error('Failed to fetch workflow:', error);
      set({ isLoading: false });
    }
  },

  createNewWorkflow: async (name: string) => {
    set({ isLoading: true });
    try {
      const created = await workflowService.create(name);
      set({
        dagId: created.id,
        dagName: created.name,
        nodes: [],
        edges: [],
        isDirty: false,
        isLoading: false,
        history: [{ nodes: [], edges: [] }],
        historyIndex: 0
      });
    } catch (error) {
      console.error('Failed to create workflow:', error);
      set({ isLoading: false });
    }
  },

  fetchWorkflowList: async () => {
    try {
      const workflows = await workflowService.getAll();
      return workflows.map(w => ({ id: w.id, name: w.name }));
    } catch (error) {
      console.error('Failed to fetch workflow list:', error);
      return [];
    }
  },

  undo: () => {
    const { historyIndex, history } = get();
    if (historyIndex > 0) {
      const prevIndex = historyIndex - 1;
      const prevState = history[prevIndex];
      set({
        nodes: prevState.nodes,
        edges: prevState.edges,
        historyIndex: prevIndex,
      });
    }
  },

  redo: () => {
    const { historyIndex, history } = get();
    if (historyIndex < history.length - 1) {
      const nextIndex = historyIndex + 1;
      const nextState = history[nextIndex];
      set({
        nodes: nextState.nodes,
        edges: nextState.edges,
        historyIndex: nextIndex,
      });
    }
  },

  clearExecution: () => {
    const { nodes, edges, executionId } = get();

    // Disconnect WebSocket via service
    if (executionId) {
      executionWS.disconnect(executionId);
    }

    const resetNodes = nodes.map(n => ({ ...n, data: { ...n.data, status: NodeStatus.IDLE } }));

    const resetEdges = edges.map(e => ({
      ...e,
      animated: false,
      style: { ...e.style, stroke: undefined, strokeWidth: undefined, strokeDasharray: undefined }
    }));

    set({
      logs: [],
      executionResults: {},
      nodes: resetNodes,
      edges: resetEdges,
      isRunning: false,
      wsConnected: false
    });
  },

  // === Real API Execution with WebSocket ===
  runWorkflow: async () => {
    let { dagId, nodes, edges } = get();

    // If workflow not saved yet, save it first
    if (!dagId) {
      console.log('Workflow not saved, saving first...');
      try {
        await get().saveDAG();
        dagId = get().dagId;
        nodes = get().nodes;
        edges = get().edges;
      } catch (error) {
        console.error('Failed to save workflow before running:', error);
        toast.error('Cannot Run', 'Please save the workflow before running.');
        return;
      }
    }

    // Still no ID after save attempt
    if (!dagId) {
      console.error('Cannot run workflow: no workflow ID after save');
      toast.error('Cannot Run', 'Please save the workflow first.');
      return;
    }

    // Reset state
    set({
      isRunning: true,
      logs: [],
      executionResults: {},
      executionId: null,
      wsConnected: false
    });

    const addLog = (nodeId: string | null, level: ExecutionLog['level'], message: string) => {
      set(state => ({
        logs: [...state.logs, {
          id: Math.random().toString(36),
          nodeId,
          level,
          message,
          timestamp: new Date()
        }]
      }));
    };

    // Set all nodes to pending
    set({
      nodes: nodes.map(n => ({ ...n, data: { ...n.data, status: NodeStatus.PENDING } })),
      edges: edges.map(e => ({
        ...e,
        animated: false,
        style: { ...e.style, stroke: undefined, strokeWidth: undefined, strokeDasharray: undefined }
      }))
    });

    addLog(null, 'info', 'Starting workflow execution...');

    try {
      // Trigger execution via API
      const execution = await executionService.trigger(dagId);
      set({ executionId: execution.id });
      addLog(null, 'info', `Execution started: ${execution.id}`);

      // Connect to WebSocket for real-time updates
      get().connectToExecution(execution.id);

    } catch (err) {
      console.error('Execution error:', err);
      addLog(null, 'error', `Execution error: ${err instanceof Error ? err.message : 'Unknown error'}`);
      set({ isRunning: false });
    }
  },

  // === WebSocket Methods (using centralized service) ===
  connectToExecution: (executionId: string) => {
    set({ wsConnected: true });

    // Connect via centralized service
    executionWS.connect(executionId, (event) => {
      get().handleExecutionEvent(event);
    });

    // Poll connection status
    const checkInterval = setInterval(() => {
      const connected = executionWS.isConnected(executionId);
      if (!connected) {
        set({ wsConnected: false });
        clearInterval(checkInterval);
      }
    }, 2000);
  },

  disconnectFromExecution: () => {
    const { executionId } = get();
    if (executionId) {
      executionWS.disconnect(executionId);
    }
    set({ wsConnected: false });
  },

  handleExecutionEvent: (event: ExecutionEvent) => {
    if (event.type === 'control') {
      console.log('[dagStore] Control message:', event.control);
      return;
    }

    if (event.type !== 'event' || !event.event) return;

    const { event: execEvent } = event;
    const { edges } = get();

    const addLog = (nodeId: string | null, level: ExecutionLog['level'], message: string) => {
      set(state => ({
        logs: [...state.logs, {
          id: Math.random().toString(36),
          nodeId,
          level,
          message,
          timestamp: new Date()
        }]
      }));
    };

    switch (execEvent.event_type) {
      case 'execution.started':
        addLog(null, 'info', 'Execution started');
        break;

      case 'node.started':
        if (execEvent.node_id) {
          // Update node status to RUNNING
          set(state => ({
            nodes: state.nodes.map(n =>
              n.id === execEvent.node_id
                ? { ...n, data: { ...n.data, status: NodeStatus.RUNNING } }
                : n
            )
          }));

          // Update executionResults with input data
          set(state => ({
            executionResults: {
              ...state.executionResults,
              [execEvent.node_id!]: {
                nodeId: execEvent.node_id!,
                status: NodeStatus.RUNNING,
                inputs: execEvent.input || {},
                outputs: {},
                startTime: Date.now(),
                logs: []
              }
            }
          }));

          // Animate incoming edges
          const incomingEdges = edges.filter(e => e.target === execEvent.node_id);
          if (incomingEdges.length > 0) {
            set(state => ({
              edges: state.edges.map(e =>
                incomingEdges.some(ie => ie.id === e.id)
                  ? { ...e, animated: true, style: { stroke: '#3b82f6', strokeWidth: 2 } }
                  : e
              )
            }));
          }

          addLog(execEvent.node_id, 'info', `Node "${execEvent.node_name || execEvent.node_id}" started`);
        }
        break;

      case 'node.completed':
        if (execEvent.node_id) {
          // Update node status to SUCCESS
          set(state => ({
            nodes: state.nodes.map(n =>
              n.id === execEvent.node_id
                ? { ...n, data: { ...n.data, status: NodeStatus.SUCCESS } }
                : n
            )
          }));

          // Update executionResults with output data
          set(state => ({
            executionResults: {
              ...state.executionResults,
              [execEvent.node_id!]: {
                ...state.executionResults[execEvent.node_id!],
                status: NodeStatus.SUCCESS,
                outputs: execEvent.output || {},
                endTime: Date.now()
              }
            }
          }));

          // Update edges: incoming → completed (green), outgoing → active (blue animated)
          const incomingEdges = edges.filter(e => e.target === execEvent.node_id);
          const outgoingEdges = edges.filter(e => e.source === execEvent.node_id);

          set(state => ({
            edges: state.edges.map(e => {
              if (incomingEdges.some(ie => ie.id === e.id)) {
                return { ...e, animated: false, style: { stroke: '#10b981', strokeWidth: 2 } };
              }
              if (outgoingEdges.some(oe => oe.id === e.id)) {
                return { ...e, animated: true, style: { stroke: '#3b82f6', strokeWidth: 2 } };
              }
              return e;
            })
          }));

          const duration = execEvent.duration_ms ? ` (${execEvent.duration_ms}ms)` : '';
          addLog(execEvent.node_id, 'success', `Node "${execEvent.node_name || execEvent.node_id}" completed${duration}`);
        }
        break;

      case 'node.failed':
        if (execEvent.node_id) {
          // Update node status to ERROR
          set(state => ({
            nodes: state.nodes.map(n =>
              n.id === execEvent.node_id
                ? { ...n, data: { ...n.data, status: NodeStatus.ERROR } }
                : n
            )
          }));

          // Update executionResults
          set(state => ({
            executionResults: {
              ...state.executionResults,
              [execEvent.node_id!]: {
                ...state.executionResults[execEvent.node_id!],
                status: NodeStatus.ERROR,
                endTime: Date.now()
              }
            }
          }));

          addLog(execEvent.node_id, 'error', `Node "${execEvent.node_name || execEvent.node_id}" failed: ${execEvent.error || 'Unknown error'}`);
        }
        break;

      case 'node.skipped':
        if (execEvent.node_id) {
          set(state => ({
            nodes: state.nodes.map(n =>
              n.id === execEvent.node_id
                ? { ...n, data: { ...n.data, status: NodeStatus.SKIPPED } }
                : n
            )
          }));

          addLog(execEvent.node_id, 'warning', `Node "${execEvent.node_name || execEvent.node_id}" skipped`);
        }
        break;

      case 'wave.started':
        addLog(null, 'info', `Wave ${execEvent.wave_index} started (${execEvent.node_count} nodes)`);
        break;

      case 'wave.completed':
        addLog(null, 'info', `Wave ${execEvent.wave_index} completed`);
        break;

      case 'execution.completed':
        // Mark all edges as completed (green)
        set(state => ({
          edges: state.edges.map(e => ({
            ...e,
            animated: false,
            style: { ...e.style, stroke: '#22c55e', strokeWidth: 2 }
          }))
        }));

        addLog(null, 'success', 'Workflow completed successfully');
        set({ isRunning: false });
        get().disconnectFromExecution();
        break;

      case 'execution.failed':
        addLog(null, 'error', `Workflow failed: ${execEvent.error || 'Unknown error'}`);
        set({ isRunning: false });
        get().disconnectFromExecution();
        break;

      case 'execution.cancelled':
        addLog(null, 'warning', 'Workflow was cancelled');
        set({ isRunning: false });
        get().disconnectFromExecution();
        break;
    }
  }
}));
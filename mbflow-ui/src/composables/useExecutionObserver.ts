import { ref, computed, watch, onUnmounted, type Ref } from "vue";
import { useWebSocket } from "./useWebSocket";
import type { Execution } from "@/types/execution";

export interface ExecutionEvent {
  type: "event" | "control";
  event?: {
    event_type: string;
    execution_id: string;
    workflow_id: string;
    timestamp: string;
    status: string;
    node_id?: string;
    node_name?: string;
    node_type?: string;
    wave_index?: number;
    node_count?: number;
    duration_ms?: number;
    error?: string;
    input?: Record<string, any>;
    output?: Record<string, any>;
  };
  control?: Record<string, any>;
  timestamp: string;
}

export interface ExecutionObserverOptions {
  /**
   * Execution ID to observe. If not provided, will observe all executions.
   */
  executionId?: string | Ref<string>;

  /**
   * Auto-connect on mount
   */
  autoConnect?: boolean;

  /**
   * Callback when execution status changes
   */
  onStatusChange?: (status: string, execution: Execution) => void;

  /**
   * Callback when execution completes
   */
  onComplete?: (execution: Execution) => void;

  /**
   * Callback when execution fails
   */
  onError?: (error: string, execution: Execution) => void;

  /**
   * Callback when node execution updates
   */
  onNodeUpdate?: (nodeId: string, event: ExecutionEvent["event"]) => void;
}

/**
 * Composable for observing execution updates via WebSocket
 *
 * @example
 * ```ts
 * const { execution, isConnected, connect, disconnect } = useExecutionObserver({
 *   executionId: 'exec-123',
 *   onStatusChange: (status) => console.log('Status:', status),
 *   onComplete: () => toast.success('Execution completed!'),
 * });
 * ```
 */
export function useExecutionObserver(options: ExecutionObserverOptions = {}) {
  const {
    executionId,
    autoConnect = true,
    onStatusChange,
    onComplete,
    onError,
    onNodeUpdate,
  } = options;

  // Build WebSocket URL
  const wsUrl = computed(() => {
    const baseUrl = import.meta.env.VITE_WS_URL || "/ws";
    const execId =
      typeof executionId === "string" ? executionId : executionId?.value;

    // Ensure proper WebSocket URL format
    let wsBase = baseUrl;
    if (!wsBase.startsWith("ws://") && !wsBase.startsWith("wss://")) {
      const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      wsBase = `${protocol}//${window.location.host}${baseUrl}`;
    }

    // Add execution ID as query parameter if provided
    const url = `${wsBase}/executions`;
    return execId ? `${url}?execution_id=${execId}` : url;
  });

  // Initialize WebSocket connection
  const {
    data: wsData,
    error: wsError,
    isConnected,
    reconnectAttempts,
    send,
    connect: wsConnect,
    disconnect: wsDisconnect,
    reconnect,
  } = useWebSocket<ExecutionEvent>(wsUrl);

  // Current execution state
  const execution = ref<Execution | null>(null);
  const lastEvent = ref<ExecutionEvent | null>(null);
  const events = ref<ExecutionEvent[]>([]);
  const previousStatus = ref<string | null>(null);

  // Watch for WebSocket data updates
  watch(wsData, (event) => {
    if (!event) return;

    lastEvent.value = event;
    // Force reactivity update by creating new array
    events.value = [...events.value, event];

    // Handle control messages
    if (event.type === "control") {
      console.log("[ExecutionObserver] Control message:", event.control);
      return;
    }

    // Handle event messages
    if (event.type === "event" && event.event) {
      const { event: execEvent } = event;

      // Update execution state based on event type
      switch (execEvent.event_type) {
        case "execution.started":
        case "execution.running":
        case "execution.completed":
        case "execution.failed":
        case "execution.cancelled":
          // Update execution status
          if (execution.value) {
            const oldStatus = execution.value.status;
            execution.value.status = execEvent.status as any;

            // Trigger status change callback
            if (oldStatus !== execEvent.status && onStatusChange) {
              onStatusChange(execEvent.status, execution.value);
            }

            // Trigger completion callback
            if (execEvent.status === "completed" && onComplete) {
              onComplete(execution.value);
            }

            // Trigger error callback
            if (execEvent.status === "failed" && execEvent.error && onError) {
              onError(execEvent.error, execution.value);
            }
          }
          break;

        case "node.started":
        case "node.completed":
        case "node.failed":
          // Handle node execution updates
          if (execEvent.node_id && onNodeUpdate) {
            onNodeUpdate(execEvent.node_id, execEvent);
          }
          break;

        case "wave.started":
        case "wave.completed":
          // Handle wave execution updates
          console.log("[ExecutionObserver] Wave event:", execEvent);
          break;
      }
    }
  });

  // Watch for WebSocket errors
  watch(wsError, (error) => {
    if (error) {
      console.error("[ExecutionObserver] WebSocket error:", error);
    }
  });

  // Connect/disconnect management
  const connect = () => {
    if (!isConnected.value) {
      wsConnect();
    }
  };

  const disconnect = () => {
    if (isConnected.value) {
      wsDisconnect();
    }
  };

  // Subscribe to specific event types
  const subscribe = (eventTypes: string[]) => {
    send({
      command: "subscribe",
      event_types: eventTypes,
    });
  };

  // Unsubscribe from specific event types
  const unsubscribe = (eventTypes: string[]) => {
    send({
      command: "unsubscribe",
      event_types: eventTypes,
    });
  };

  // Clear events history
  const clearEvents = () => {
    events.value = [];
    lastEvent.value = null;
  };

  // Set initial execution data (from API call)
  const setExecution = (exec: Execution) => {
    execution.value = exec;
    previousStatus.value = exec.status;
  };

  // Auto-connect if enabled
  if (autoConnect) {
    connect();
  }

  // Auto-disconnect on unmount
  onUnmounted(() => {
    disconnect();
  });

  return {
    // State
    execution,
    lastEvent,
    events,
    isConnected,
    reconnectAttempts,
    wsError,

    // Methods
    connect,
    disconnect,
    reconnect,
    subscribe,
    unsubscribe,
    clearEvents,
    setExecution,
    send,
  };
}

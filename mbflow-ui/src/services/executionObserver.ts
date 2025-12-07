import { ref, reactive } from "vue";
import type { Execution } from "@/types/execution";
import type { ExecutionEvent } from "@/composables/useExecutionObserver";

/**
 * Global execution observer service
 * Manages WebSocket connections for multiple executions
 */
class ExecutionObserverService {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectTimeout: NodeJS.Timeout | null = null;
  private pingInterval: NodeJS.Timeout | null = null;

  // Reactive state
  public isConnected = ref(false);
  public executions = reactive<Map<string, Execution>>(new Map());
  public events = reactive<Map<string, ExecutionEvent[]>>(new Map());

  // Callbacks
  private statusChangeCallbacks = new Map<
    string,
    Array<(status: string) => void>
  >();
  private completeCallbacks = new Map<string, Array<() => void>>();
  private errorCallbacks = new Map<string, Array<(error: string) => void>>();

  constructor() {
    this.connect();
  }

  /**
   * Connect to WebSocket server
   */
  private connect() {
    const baseUrl = import.meta.env.VITE_WS_URL || "/ws";

    let wsUrl = baseUrl;
    if (!wsUrl.startsWith("ws://") && !wsUrl.startsWith("wss://")) {
      const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      wsUrl = `${protocol}//${window.location.host}${baseUrl}`;
    }

    // Connect to all executions endpoint (no execution_id filter)
    wsUrl = `${wsUrl}/executions`;

    try {
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        this.isConnected.value = true;
        this.reconnectAttempts = 0;
        console.log("[ExecutionObserverService] Connected to WebSocket");

        // Send ping every 30 seconds
        this.pingInterval = setInterval(() => {
          if (this.ws?.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({ type: "ping" }));
          }
        }, 30000);
      };

      this.ws.onmessage = (event) => {
        try {
          const data = event.data;

          // Try to parse as single JSON first
          try {
            const message: ExecutionEvent = JSON.parse(data);
            this.handleMessage(message);
          } catch (parseError) {
            // If single parse fails, try to split multiple JSON objects
            // This can happen when multiple events are sent rapidly
            const jsonObjects = this.splitConcatenatedJSON(data);

            if (jsonObjects.length > 0) {
              jsonObjects.forEach((jsonStr) => {
                try {
                  const message: ExecutionEvent = JSON.parse(jsonStr);
                  this.handleMessage(message);
                } catch (e) {
                  console.error(
                    "[ExecutionObserverService] Failed to parse individual message:",
                    e,
                    "Data:",
                    jsonStr.substring(0, 100),
                  );
                }
              });
            } else {
              throw parseError; // Re-throw original error if splitting didn't help
            }
          }
        } catch (e) {
          console.error(
            "[ExecutionObserverService] Failed to parse message:",
            e,
            "Data:",
            typeof event.data === "string"
              ? event.data.substring(0, 200)
              : event.data,
          );
        }
      };

      this.ws.onerror = (event) => {
        console.error("[ExecutionObserverService] WebSocket error:", event);
        this.isConnected.value = false;
      };

      this.ws.onclose = () => {
        this.isConnected.value = false;

        if (this.pingInterval) {
          clearInterval(this.pingInterval);
          this.pingInterval = null;
        }

        // Attempt to reconnect
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
          this.reconnectAttempts++;
          const delay = Math.min(
            1000 * Math.pow(2, this.reconnectAttempts),
            30000,
          );

          this.reconnectTimeout = setTimeout(() => {
            console.log(
              `[ExecutionObserverService] Reconnecting... (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`,
            );
            this.connect();
          }, delay);
        }
      };
    } catch (e) {
      console.error(
        "[ExecutionObserverService] Failed to create WebSocket:",
        e,
      );
      this.isConnected.value = false;
    }
  }

  /**
   * Split concatenated JSON objects into separate strings
   * Handles cases where multiple JSON objects are sent in one message
   */
  private splitConcatenatedJSON(data: string): string[] {
    const results: string[] = [];
    let depth = 0;
    let start = 0;
    let inString = false;
    let escapeNext = false;

    for (let i = 0; i < data.length; i++) {
      const char = data[i];

      if (escapeNext) {
        escapeNext = false;
        continue;
      }

      if (char === "\\") {
        escapeNext = true;
        continue;
      }

      if (char === '"' && !escapeNext) {
        inString = !inString;
        continue;
      }

      if (inString) {
        continue;
      }

      if (char === "{") {
        depth++;
      } else if (char === "}") {
        depth--;

        // Found complete JSON object
        if (depth === 0) {
          const jsonStr = data.substring(start, i + 1).trim();
          if (jsonStr) {
            results.push(jsonStr);
          }
          start = i + 1;
        }
      }
    }

    return results;
  }

  /**
   * Handle incoming WebSocket message
   */
  private handleMessage(message: ExecutionEvent) {
    if (message.type === "control") {
      console.log(
        "[ExecutionObserverService] Control message:",
        message.control,
      );
      return;
    }

    if (message.type === "event" && message.event) {
      const { event } = message;
      const executionId = event.execution_id;

      // Store event
      if (!this.events.has(executionId)) {
        this.events.set(executionId, []);
      }
      this.events.get(executionId)!.push(message);

      // Update execution state
      const execution = this.executions.get(executionId);
      if (execution) {
        const oldStatus = execution.status;

        // Update execution based on event type
        switch (event.event_type) {
          case "execution.started":
          case "execution.running":
          case "execution.completed":
          case "execution.failed":
          case "execution.cancelled":
            execution.status = event.status as any;

            if (event.error) {
              execution.error = event.error;
            }

            // Trigger callbacks
            if (oldStatus !== event.status) {
              this.triggerStatusChange(executionId, event.status);
            }

            if (event.status === "completed") {
              this.triggerComplete(executionId);
            }

            if (event.status === "failed" && event.error) {
              this.triggerError(executionId, event.error);
            }
            break;
        }
      }
    }
  }

  /**
   * Observe an execution
   */
  observe(execution: Execution) {
    this.executions.set(execution.id, execution);
    console.log(
      `[ExecutionObserverService] Observing execution: ${execution.id}`,
    );
  }

  /**
   * Stop observing an execution
   */
  unobserve(executionId: string) {
    this.executions.delete(executionId);
    this.events.delete(executionId);
    this.statusChangeCallbacks.delete(executionId);
    this.completeCallbacks.delete(executionId);
    this.errorCallbacks.delete(executionId);
    console.log(
      `[ExecutionObserverService] Stopped observing execution: ${executionId}`,
    );
  }

  /**
   * Register status change callback
   */
  onStatusChange(executionId: string, callback: (status: string) => void) {
    if (!this.statusChangeCallbacks.has(executionId)) {
      this.statusChangeCallbacks.set(executionId, []);
    }
    this.statusChangeCallbacks.get(executionId)!.push(callback);
  }

  /**
   * Register completion callback
   */
  onComplete(executionId: string, callback: () => void) {
    if (!this.completeCallbacks.has(executionId)) {
      this.completeCallbacks.set(executionId, []);
    }
    this.completeCallbacks.get(executionId)!.push(callback);
  }

  /**
   * Register error callback
   */
  onError(executionId: string, callback: (error: string) => void) {
    if (!this.errorCallbacks.has(executionId)) {
      this.errorCallbacks.set(executionId, []);
    }
    this.errorCallbacks.get(executionId)!.push(callback);
  }

  /**
   * Trigger status change callbacks
   */
  private triggerStatusChange(executionId: string, status: string) {
    const callbacks = this.statusChangeCallbacks.get(executionId) || [];
    callbacks.forEach((cb) => cb(status));
  }

  /**
   * Trigger completion callbacks
   */
  private triggerComplete(executionId: string) {
    const callbacks = this.completeCallbacks.get(executionId) || [];
    callbacks.forEach((cb) => cb());
  }

  /**
   * Trigger error callbacks
   */
  private triggerError(executionId: string, error: string) {
    const callbacks = this.errorCallbacks.get(executionId) || [];
    callbacks.forEach((cb) => cb(error));
  }

  /**
   * Get execution by ID
   */
  getExecution(executionId: string): Execution | undefined {
    return this.executions.get(executionId);
  }

  /**
   * Get events for execution
   */
  getEvents(executionId: string): ExecutionEvent[] {
    return this.events.get(executionId) || [];
  }

  /**
   * Disconnect from WebSocket
   */
  disconnect() {
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }

    if (this.pingInterval) {
      clearInterval(this.pingInterval);
      this.pingInterval = null;
    }

    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }

    this.isConnected.value = false;
    this.reconnectAttempts = 0;
  }

  /**
   * Reconnect to WebSocket
   */
  reconnect() {
    this.disconnect();
    this.reconnectAttempts = 0;
    this.connect();
  }
}

// Export singleton instance
export const executionObserver = new ExecutionObserverService();

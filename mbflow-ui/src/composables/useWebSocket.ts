import { ref, onMounted, onUnmounted, type Ref } from "vue";

export interface WebSocketMessage {
  type: string;
  data: any;
}

export function useWebSocket<T = any>(url: string | Ref<string>) {
  const data = ref<T | null>(null);
  const error = ref<Error | null>(null);
  const isConnected = ref(false);
  const reconnectAttempts = ref(0);
  const maxReconnectAttempts = 5;

  let ws: WebSocket | null = null;
  let reconnectTimeout: NodeJS.Timeout | null = null;
  let pingInterval: NodeJS.Timeout | null = null;

  const connect = () => {
    const wsUrl = typeof url === "string" ? url : url.value;
    if (!wsUrl) return;

    try {
      ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        isConnected.value = true;
        error.value = null;
        reconnectAttempts.value = 0;

        // Send ping every 30 seconds to keep connection alive
        pingInterval = setInterval(() => {
          if (ws?.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: "ping" }));
          }
        }, 30000);
      };

      ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);

          // Handle pong response
          if (message.type === "pong") {
            return;
          }

          data.value = message as T;
        } catch (e) {
          console.error("Failed to parse WebSocket message:", e);
          error.value = new Error("Failed to parse WebSocket message");
        }
      };

      ws.onerror = (event) => {
        console.error("WebSocket error:", event);
        error.value = new Error("WebSocket connection error");
        isConnected.value = false;
      };

      ws.onclose = () => {
        isConnected.value = false;

        if (pingInterval) {
          clearInterval(pingInterval);
          pingInterval = null;
        }

        // Attempt to reconnect if we haven't exceeded max attempts
        if (reconnectAttempts.value < maxReconnectAttempts) {
          reconnectAttempts.value++;
          const delay = Math.min(
            1000 * Math.pow(2, reconnectAttempts.value),
            30000,
          );

          reconnectTimeout = setTimeout(() => {
            console.log(
              `Reconnecting... (attempt ${reconnectAttempts.value}/${maxReconnectAttempts})`,
            );
            connect();
          }, delay);
        } else {
          error.value = new Error("Max reconnection attempts reached");
        }
      };
    } catch (e) {
      error.value =
        e instanceof Error
          ? e
          : new Error("Failed to create WebSocket connection");
      isConnected.value = false;
    }
  };

  const disconnect = () => {
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout);
      reconnectTimeout = null;
    }

    if (pingInterval) {
      clearInterval(pingInterval);
      pingInterval = null;
    }

    if (ws) {
      ws.close();
      ws = null;
    }

    isConnected.value = false;
    reconnectAttempts.value = 0;
  };

  const send = (message: any) => {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(message));
    } else {
      console.warn("WebSocket is not connected");
    }
  };

  const reconnect = () => {
    disconnect();
    reconnectAttempts.value = 0;
    connect();
  };

  onMounted(() => {
    connect();
  });

  onUnmounted(() => {
    disconnect();
  });

  return {
    data,
    error,
    isConnected,
    reconnectAttempts,
    send,
    connect,
    disconnect,
    reconnect,
  };
}

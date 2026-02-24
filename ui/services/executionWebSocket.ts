/**
 * Centralized WebSocket service for execution monitoring.
 * Single source of truth for all WebSocket connections to execution events.
 */

import type { ExecutionEvent } from '@/types/execution';

export type ExecutionEventHandler = (event: ExecutionEvent) => void;

interface ConnectionState {
  ws: WebSocket | null;
  pingInterval: ReturnType<typeof setInterval> | null;
  handlers: Set<ExecutionEventHandler>;
  executionId: string;
}

class ExecutionWebSocketService {
  private connections: Map<string, ConnectionState> = new Map();
  private globalHandlers: Set<ExecutionEventHandler> = new Set();

  /**
   * Build WebSocket URL for execution monitoring
   */
  private buildUrl(executionId?: string): string {
    const baseUrl = (import.meta as any).env?.VITE_WS_URL || '/ws';
    let wsUrl = baseUrl;

    if (!wsUrl.startsWith('ws://') && !wsUrl.startsWith('wss://')) {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      wsUrl = `${protocol}//${window.location.host}${baseUrl}`;
    }

    const url = `${wsUrl}/executions`;
    return executionId ? `${url}?execution_id=${executionId}` : url;
  }

  /**
   * Connect to WebSocket for a specific execution
   */
  connect(executionId: string, handler?: ExecutionEventHandler): () => void {
    // Check if already connected
    let state = this.connections.get(executionId);

    if (state?.ws?.readyState === WebSocket.OPEN) {
      // Already connected, just add handler
      if (handler) {
        state.handlers.add(handler);
      }
      return () => this.removeHandler(executionId, handler);
    }

    // Create new connection
    const url = this.buildUrl(executionId);
    console.log('[ExecutionWS] Connecting:', url);

    const ws = new WebSocket(url);
    const handlers = new Set<ExecutionEventHandler>();
    if (handler) {
      handlers.add(handler);
    }

    state = {
      ws,
      pingInterval: null,
      handlers,
      executionId
    };

    this.connections.set(executionId, state);

    ws.onopen = () => {
      console.log('[ExecutionWS] Connected:', executionId);

      // Start ping interval
      state!.pingInterval = setInterval(() => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'ping' }));
        }
      }, 30000);
    };

    ws.onmessage = (event) => {
      try {
        const message: ExecutionEvent = JSON.parse(event.data);

        // Ignore pong
        if ((message as any).type === 'pong') return;

        // Notify all handlers for this execution
        const currentState = this.connections.get(executionId);
        if (currentState) {
          currentState.handlers.forEach(h => h(message));
        }

        // Notify global handlers
        this.globalHandlers.forEach(h => h(message));

      } catch (e) {
        console.error('[ExecutionWS] Parse error:', e);
      }
    };

    ws.onerror = (error) => {
      console.error('[ExecutionWS] Error:', error);
    };

    ws.onclose = (event) => {
      console.log('[ExecutionWS] Disconnected:', executionId, 'code:', event.code);
      this.cleanup(executionId);
    };

    // Return unsubscribe function
    return () => this.removeHandler(executionId, handler);
  }

  /**
   * Remove a specific handler from execution
   */
  private removeHandler(executionId: string, handler?: ExecutionEventHandler): void {
    const state = this.connections.get(executionId);
    if (state && handler) {
      state.handlers.delete(handler);

      // If no more handlers, disconnect
      if (state.handlers.size === 0) {
        this.disconnect(executionId);
      }
    }
  }

  /**
   * Disconnect from a specific execution
   */
  disconnect(executionId: string): void {
    const state = this.connections.get(executionId);
    if (!state) return;

    console.log('[ExecutionWS] Disconnecting:', executionId);
    this.cleanup(executionId);

    if (state.ws && state.ws.readyState === WebSocket.OPEN) {
      state.ws.close();
    }
  }

  /**
   * Cleanup connection state
   */
  private cleanup(executionId: string): void {
    const state = this.connections.get(executionId);
    if (!state) return;

    if (state.pingInterval) {
      clearInterval(state.pingInterval);
    }

    this.connections.delete(executionId);
  }

  /**
   * Check if connected to a specific execution
   */
  isConnected(executionId: string): boolean {
    const state = this.connections.get(executionId);
    return state?.ws?.readyState === WebSocket.OPEN;
  }

  /**
   * Add global handler for all execution events
   */
  addGlobalHandler(handler: ExecutionEventHandler): () => void {
    this.globalHandlers.add(handler);
    return () => this.globalHandlers.delete(handler);
  }

  /**
   * Disconnect all connections
   */
  disconnectAll(): void {
    this.connections.forEach((_, executionId) => {
      this.disconnect(executionId);
    });
  }
}

// Singleton instance
export const executionWS = new ExecutionWebSocketService();

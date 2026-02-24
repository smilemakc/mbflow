/**
 * React hook for execution WebSocket - wraps the singleton service
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import { executionWS, type ExecutionEventHandler } from '@/services/executionWebSocket';
import type { ExecutionEvent } from '@/types/execution';

export interface UseExecutionWSOptions {
  /** Execution ID to monitor */
  executionId?: string;
  /** Auto-connect when executionId is provided */
  autoConnect?: boolean;
  /** Event handler */
  onEvent?: ExecutionEventHandler;
}

export interface UseExecutionWSReturn {
  /** Whether WebSocket is connected */
  isConnected: boolean;
  /** Last received event */
  lastEvent: ExecutionEvent | null;
  /** Connect to execution */
  connect: () => void;
  /** Disconnect from execution */
  disconnect: () => void;
}

/**
 * Hook to subscribe to execution WebSocket events
 *
 * @example
 * ```tsx
 * const { isConnected, lastEvent } = useExecutionWS({
 *   executionId: 'exec-123',
 *   onEvent: (event) => console.log(event)
 * });
 * ```
 */
export function useExecutionWS(options: UseExecutionWSOptions = {}): UseExecutionWSReturn {
  const { executionId, autoConnect = true, onEvent } = options;

  const [isConnected, setIsConnected] = useState(false);
  const [lastEvent, setLastEvent] = useState<ExecutionEvent | null>(null);
  const unsubscribeRef = useRef<(() => void) | null>(null);
  const onEventRef = useRef(onEvent);

  // Keep onEvent ref updated
  useEffect(() => {
    onEventRef.current = onEvent;
  }, [onEvent]);

  // Event handler
  const handleEvent = useCallback((event: ExecutionEvent) => {
    setLastEvent(event);
    onEventRef.current?.(event);
  }, []);

  // Connect function
  const connect = useCallback(() => {
    if (!executionId) return;

    // Cleanup previous connection
    if (unsubscribeRef.current) {
      unsubscribeRef.current();
    }

    unsubscribeRef.current = executionWS.connect(executionId, handleEvent);
    setIsConnected(true);

    // Poll connection status
    const checkConnection = setInterval(() => {
      const connected = executionWS.isConnected(executionId);
      setIsConnected(connected);
      if (!connected) {
        clearInterval(checkConnection);
      }
    }, 1000);

    // Store interval cleanup
    const originalUnsubscribe = unsubscribeRef.current;
    unsubscribeRef.current = () => {
      clearInterval(checkConnection);
      originalUnsubscribe();
    };
  }, [executionId, handleEvent]);

  // Disconnect function
  const disconnect = useCallback(() => {
    if (unsubscribeRef.current) {
      unsubscribeRef.current();
      unsubscribeRef.current = null;
    }
    if (executionId) {
      executionWS.disconnect(executionId);
    }
    setIsConnected(false);
  }, [executionId]);

  // Auto-connect effect
  useEffect(() => {
    if (autoConnect && executionId) {
      connect();
    }

    return () => {
      if (unsubscribeRef.current) {
        unsubscribeRef.current();
        unsubscribeRef.current = null;
      }
    };
  }, [executionId, autoConnect]); // eslint-disable-line react-hooks/exhaustive-deps

  return {
    isConnected,
    lastEvent,
    connect,
    disconnect
  };
}

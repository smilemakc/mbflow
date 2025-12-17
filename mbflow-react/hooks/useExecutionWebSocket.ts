import {useCallback, useEffect, useRef, useState} from 'react';
import {executionWS} from '@/services/executionWebSocket';
import {Execution, ExecutionEvent, ExecutionStatus, NodeExecution} from '@/types/execution';
import {useToast} from '@/hooks/useToast';

interface UseExecutionWebSocketOptions {
    executionId: string | undefined;
    onExecutionUpdate: (updates: Partial<Execution>) => void;
    onNodeUpdate?: (nodeId: string, update: any) => void;
}

export function useExecutionWebSocket({
    executionId,
    onExecutionUpdate,
    onNodeUpdate
}: UseExecutionWebSocketOptions) {
    const [wsConnected, setWsConnected] = useState(false);
    const {showToast} = useToast();
    const hasShownCompletionToastRef = useRef(false);
    const unsubscribeRef = useRef<(() => void) | null>(null);

    const handleWsEvent = useCallback((event: ExecutionEvent) => {
        if (event.type === 'control') {
            console.log('[useExecutionWebSocket] Control message:', event.control);
            return;
        }

        if (event.type !== 'event' || !event.event) return;

        const execEvent = event.event;
        console.log('[useExecutionWebSocket] Event:', execEvent.event_type, execEvent);

        switch (execEvent.event_type) {
            case 'execution.completed':
                onExecutionUpdate({status: 'completed' as ExecutionStatus});
                if (!hasShownCompletionToastRef.current) {
                    hasShownCompletionToastRef.current = true;
                    showToast({type: 'success', title: 'Execution completed'});
                }
                break;

            case 'execution.failed':
                onExecutionUpdate({
                    status: 'failed' as ExecutionStatus,
                    error: execEvent.error
                });
                if (!hasShownCompletionToastRef.current) {
                    hasShownCompletionToastRef.current = true;
                    showToast({type: 'error', title: `Execution failed: ${execEvent.error || 'Unknown error'}`});
                }
                break;

            case 'node.started':
            case 'node.completed':
            case 'node.failed':
                if (execEvent.node_id && onNodeUpdate) {
                    const nodeExec: NodeExecution = {
                        id: `${execEvent.execution_id}_${execEvent.node_id}`,
                        execution_id: execEvent.execution_id,
                        node_id: execEvent.node_id,
                        node_name: execEvent.node_name,
                        node_type: execEvent.node_type,
                        status: (execEvent.event_type === 'node.started' ? 'running' :
                            execEvent.event_type === 'node.completed' ? 'completed' : 'failed') as ExecutionStatus,
                        started_at: execEvent.timestamp,
                        completed_at: execEvent.event_type !== 'node.started' ? execEvent.timestamp : undefined,
                        duration: execEvent.duration_ms,
                        error: execEvent.error,
                        input: execEvent.input,
                        output: execEvent.output
                    };
                    onNodeUpdate(execEvent.node_id, nodeExec);
                }
                break;
        }
    }, [onExecutionUpdate, onNodeUpdate, showToast]);

    const connect = useCallback((execId: string) => {
        if (unsubscribeRef.current) {
            unsubscribeRef.current();
        }

        hasShownCompletionToastRef.current = false;
        setWsConnected(true);
        unsubscribeRef.current = executionWS.connect(execId, handleWsEvent);

        const checkInterval = setInterval(() => {
            const connected = executionWS.isConnected(execId);
            setWsConnected(connected);
            if (!connected) {
                clearInterval(checkInterval);
            }
        }, 2000);

        const originalUnsubscribe = unsubscribeRef.current;
        unsubscribeRef.current = () => {
            clearInterval(checkInterval);
            originalUnsubscribe();
        };
    }, [handleWsEvent]);

    const disconnect = useCallback(() => {
        if (unsubscribeRef.current) {
            unsubscribeRef.current();
            unsubscribeRef.current = null;
        }
        if (executionId) {
            executionWS.disconnect(executionId);
        }
        setWsConnected(false);
    }, [executionId]);

    useEffect(() => {
        return () => {
            disconnect();
        };
    }, [disconnect]);

    return {
        wsConnected,
        connect,
        disconnect
    };
}

import {useCallback, useEffect, useState} from 'react';
import {executionService} from '@/services/executionService';
import {workflowService} from '@/services/workflowService';
import {Execution} from '@/types/execution';
import {DAG} from '@/types';
import {useToast} from '@/hooks/useToast';

export function useExecutionData(executionId: string | undefined) {
    const [execution, setExecution] = useState<Execution | null>(null);
    const [workflow, setWorkflow] = useState<DAG | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const {showToast} = useToast();

    const fetchExecution = useCallback(async (showLoadingState = true) => {
        if (!executionId) return;

        if (showLoadingState) {
            setIsLoading(true);
        }

        setError(null);

        try {
            const data = await executionService.getStatus(executionId);
            setExecution(data);

            if (showLoadingState && data.workflow_id) {
                try {
                    const wf = await workflowService.getById(data.workflow_id);
                    setWorkflow(wf);
                } catch (err) {
                    console.error('Failed to fetch workflow:', err);
                }
            }
        } catch (err) {
            console.error('Failed to fetch execution:', err);
            const errorMessage = err instanceof Error ? err.message : 'Failed to load execution details';
            setError(errorMessage);
            if (showLoadingState) {
                showToast({type: 'error', title: errorMessage});
            }
        } finally {
            if (showLoadingState) {
                setIsLoading(false);
            }
        }
    }, [executionId, showToast]);

    useEffect(() => {
        fetchExecution(true);
    }, [executionId]);

    return {
        execution,
        workflow,
        isLoading,
        error,
        refetch: fetchExecution,
        setExecution
    };
}

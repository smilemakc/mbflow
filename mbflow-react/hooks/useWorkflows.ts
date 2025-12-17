import { useState, useEffect, useCallback } from 'react';
import { DAG } from '@/types';
import { workflowService } from '@/services/workflowService';
import { toast } from '@/lib/toast';

interface UseWorkflowsResult {
  workflows: DAG[];
  isLoading: boolean;
  error: string | null;
  loadWorkflows: () => Promise<void>;
  cloneWorkflow: (workflow: DAG) => Promise<void>;
  deleteWorkflow: (id: string) => Promise<void>;
}

interface UseWorkflowsOptions {
  autoLoad?: boolean;
  onCloneError?: (error: unknown) => void;
  onDeleteError?: (error: unknown) => void;
  translations?: {
    cloneFailed: string;
    cloneMessage: string;
    deleteFailed: string;
    deleteMessage: string;
  };
}

export const useWorkflows = (options: UseWorkflowsOptions = {}): UseWorkflowsResult => {
  const { autoLoad = true, onCloneError, onDeleteError, translations } = options;

  const [workflows, setWorkflows] = useState<DAG[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadWorkflows = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const data = await workflowService.getAll();
      setWorkflows(data);
    } catch (err) {
      console.error('Failed to load workflows:', err);
      setError('Failed to load workflows. Please try again.');
    } finally {
      setIsLoading(false);
    }
  }, []);

  const cloneWorkflow = useCallback(
    async (workflow: DAG) => {
      try {
        const cloned = await workflowService.create(
          `${workflow.name} (Copy)`,
          workflow.description
        );

        if (workflow.nodes.length > 0 || workflow.edges.length > 0) {
          await workflowService.save({
            id: cloned.id,
            name: cloned.name,
            description: cloned.description,
            nodes: workflow.nodes,
            edges: workflow.edges,
          });
        }

        await loadWorkflows();
      } catch (err) {
        console.error('Failed to clone workflow:', err);
        if (translations) {
          toast.error(translations.cloneFailed, translations.cloneMessage);
        }
        if (onCloneError) {
          onCloneError(err);
        }
      }
    },
    [loadWorkflows, onCloneError, translations]
  );

  const deleteWorkflow = useCallback(
    async (id: string) => {
      try {
        await workflowService.delete(id);
        await loadWorkflows();
      } catch (err) {
        console.error('Failed to delete workflow:', err);
        if (translations) {
          toast.error(translations.deleteFailed, translations.deleteMessage);
        }
        if (onDeleteError) {
          onDeleteError(err);
        }
      }
    },
    [loadWorkflows, onDeleteError, translations]
  );

  useEffect(() => {
    if (autoLoad) {
      loadWorkflows();
    }
  }, [autoLoad, loadWorkflows]);

  return {
    workflows,
    isLoading,
    error,
    loadWorkflows,
    cloneWorkflow,
    deleteWorkflow,
  };
};

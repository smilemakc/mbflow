import { useState, useEffect, useMemo } from 'react';
import { DAG, WorkflowStatus } from '@/types';

export interface UseWorkflowFiltersResult {
  filteredWorkflows: DAG[];
  searchQuery: string;
  setSearchQuery: (query: string) => void;
  statusFilter: WorkflowStatus | 'all';
  setStatusFilter: (status: WorkflowStatus | 'all') => void;
  hasActiveFilters: boolean;
}

export const useWorkflowFilters = (workflows: DAG[]): UseWorkflowFiltersResult => {
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<WorkflowStatus | 'all'>('all');

  const filteredWorkflows = useMemo(() => {
    let filtered = [...workflows];

    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(
        (w) =>
          w.name.toLowerCase().includes(query) ||
          (w.description && w.description.toLowerCase().includes(query))
      );
    }

    if (statusFilter !== 'all') {
      filtered = filtered.filter((w) => w.status === statusFilter);
    }

    return filtered;
  }, [workflows, searchQuery, statusFilter]);

  const hasActiveFilters = searchQuery !== '' || statusFilter !== 'all';

  return {
    filteredWorkflows,
    searchQuery,
    setSearchQuery,
    statusFilter,
    setStatusFilter,
    hasActiveFilters,
  };
};

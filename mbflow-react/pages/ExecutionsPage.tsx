import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  AlertCircle,
  CheckCircle,
  Clock,
  ExternalLink,
  Loader2,
  Pause,
  RefreshCw,
  XCircle,
} from 'lucide-react';
import { useTranslation } from '@/store/translations';
import { executionService } from '@/services/executionService';
import { workflowService } from '@/services/workflowService';
import { Execution, ExecutionStatus } from '@/types/execution';
import { DAG } from '@/types';
import { Button } from '@/components/ui';
import { DataTable, Column } from '@/components/ui/DataTable';
import { FilterBar, FilterSelect } from '@/components/ui/FilterBar';
import { StatusBadge } from '@/components/ui/StatusBadge';
import { useTableData } from '@/hooks/useTableData';

interface ExecutionFilters {
  workflow_id?: string;
  status?: ExecutionStatus;
}

export const ExecutionsPage: React.FC = () => {
  const t = useTranslation();
  const navigate = useNavigate();
  const [workflows, setWorkflows] = useState<DAG[]>([]);
  const [retryingId, setRetryingId] = useState<string | null>(null);

  const table = useTableData<Execution, ExecutionFilters>({
    fetchFn: async ({ limit, offset, filters }) => {
      const response = await executionService.getAll({
        ...filters,
        limit,
        offset,
      });
      return {
        items: response.executions,
        total: response.total,
      };
    },
    initialLimit: 20,
    initialFilters: {},
  });

  useEffect(() => {
    fetchWorkflows();
  }, []);

  const fetchWorkflows = async () => {
    try {
      const data = await workflowService.getAll();
      setWorkflows(data);
    } catch (error) {
      console.error('Failed to fetch workflows:', error);
    }
  };

  const handleRetry = async (executionId: string) => {
    setRetryingId(executionId);
    try {
      await executionService.retry(executionId);
      table.refresh();
    } catch (error) {
      console.error('Failed to retry execution:', error);
    } finally {
      setRetryingId(null);
    }
  };

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    return new Intl.DateTimeFormat('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    }).format(date);
  };

  const formatDuration = (execution: Execution) => {
    if (!execution.completed_at) {
      return '-';
    }
    const start = new Date(execution.started_at).getTime();
    const end = new Date(execution.completed_at).getTime();
    const durationMs = end - start;

    if (durationMs < 1000) return `${durationMs}ms`;
    if (durationMs < 60000) return `${(durationMs / 1000).toFixed(1)}s`;
    return `${(durationMs / 60000).toFixed(1)}m`;
  };

  const getStatusIcon = (status: ExecutionStatus) => {
    switch (status) {
      case 'completed':
        return <CheckCircle size={16} />;
      case 'failed':
        return <XCircle size={16} />;
      case 'running':
        return <Loader2 className="animate-spin" size={16} />;
      case 'pending':
        return <Clock size={16} />;
      case 'cancelled':
        return <Pause size={16} />;
      default:
        return <AlertCircle size={16} />;
    }
  };

  const getStatusVariant = (status: ExecutionStatus) => {
    switch (status) {
      case 'completed':
        return 'success';
      case 'failed':
        return 'error';
      case 'running':
        return 'info';
      case 'pending':
        return 'warning';
      case 'cancelled':
        return 'neutral';
      default:
        return 'neutral';
    }
  };

  const getWorkflowName = (workflowId: string | undefined) => {
    if (!workflowId) return 'Unknown';
    const workflow = workflows.find((w) => w.id === workflowId);
    return workflow?.name || workflowId.substring(0, 8);
  };

  const columns: Column<Execution>[] = [
    {
      key: 'id',
      header: t.executions.table.id,
      width: '100px',
      render: (execution) => (
        <span className="font-mono text-xs text-slate-500 dark:text-slate-400">
          {execution.id.substring(0, 8)}
        </span>
      ),
    },
    {
      key: 'workflow_id',
      header: t.executions.table.workflow,
      render: (execution) => (
        <div>
          <span className="font-medium text-slate-900 dark:text-slate-200">
            {getWorkflowName(execution.workflow_id)}
          </span>
          <div className="text-xs text-slate-400 font-mono mt-0.5">
            {execution.workflow_id.substring(0, 8)}
          </div>
        </div>
      ),
    },
    {
      key: 'status',
      header: t.executions.table.status,
      render: (execution) => (
        <StatusBadge
          status={t.executions.status[execution.status] || execution.status}
          variant={getStatusVariant(execution.status)}
          icon={getStatusIcon(execution.status)}
        />
      ),
    },
    {
      key: 'started_at',
      header: t.executions.table.startedAt,
      render: (execution) => (
        <span className="text-slate-600 dark:text-slate-300">{formatDate(execution.started_at)}</span>
      ),
    },
    {
      key: 'duration',
      header: t.executions.table.duration,
      render: (execution) => (
        <span className="text-slate-500 dark:text-slate-400 font-mono text-xs">
          {formatDuration(execution)}
        </span>
      ),
    },
    {
      key: 'triggered_by',
      header: t.executions.table.triggeredBy,
      render: (execution) => (
        <span className="inline-flex items-center px-2 py-0.5 rounded-full bg-slate-100 dark:bg-slate-800 text-xs text-slate-600 dark:text-slate-300">
          {execution.triggered_by || 'System'}
        </span>
      ),
    },
  ];

  return (
    <div className="flex-1 h-full overflow-y-auto bg-slate-50 dark:bg-slate-950 p-6 md:p-8">
      <div className="max-w-7xl mx-auto space-y-6">
        <div className="flex justify-between items-start">
          <div>
            <h1 className="text-2xl font-bold text-slate-900 dark:text-white">{t.executions.title}</h1>
            <p className="text-slate-500 dark:text-slate-400 mt-1">{t.executions.subtitle}</p>
          </div>
          <Button
            onClick={table.refresh}
            variant="outline"
            size="sm"
            icon={<RefreshCw size={16} />}
            disabled={table.loading}
          >
            Refresh
          </Button>
        </div>

        <FilterBar>
          <FilterSelect
            label={t.executions.table.workflow}
            value={table.filters.workflow_id || ''}
            onChange={(value) =>
              table.setFilters({
                ...table.filters,
                workflow_id: value || undefined,
              })
            }
            options={[
              { label: t.executions.allWorkflows, value: '' },
              ...workflows.map((workflow) => ({
                label: workflow.name,
                value: workflow.id,
              })),
            ]}
          />
          <FilterSelect
            label={t.executions.table.status}
            value={table.filters.status || ''}
            onChange={(value) =>
              table.setFilters({
                ...table.filters,
                status: (value as ExecutionStatus) || undefined,
              })
            }
            options={[
              { label: t.executions.allStatuses, value: '' },
              { label: t.executions.status.pending, value: 'pending' },
              { label: t.executions.status.running, value: 'running' },
              { label: t.executions.status.completed, value: 'completed' },
              { label: t.executions.status.failed, value: 'failed' },
              { label: t.executions.status.cancelled, value: 'cancelled' },
            ]}
          />
        </FilterBar>

        <DataTable
          data={table.items}
          columns={columns}
          keyExtractor={(execution) => execution.id}
          loading={table.loading}
          error={table.error}
          emptyIcon={AlertCircle}
          emptyTitle={t.executions.noData}
          emptyDescription="No executions found with the current filters"
          onRowClick={(execution) => navigate(`/executions/${execution.id}`)}
          actions={(execution) => (
            <>
              <Button
                onClick={() => navigate(`/executions/${execution.id}`)}
                variant="ghost"
                size="sm"
                icon={<ExternalLink size={16} />}
                title={t.executions.actions.viewDetails}
              />
              {execution.status === 'failed' && (
                <Button
                  onClick={() => handleRetry(execution.id)}
                  disabled={retryingId === execution.id}
                  variant="ghost"
                  size="sm"
                  loading={retryingId === execution.id}
                  icon={<RefreshCw size={16} />}
                  title={t.executions.actions.retry}
                />
              )}
            </>
          )}
          pagination={{
            offset: table.offset,
            limit: table.limit,
            total: table.total,
            onOffsetChange: table.setOffset,
          }}
        />
      </div>
    </div>
  );
};

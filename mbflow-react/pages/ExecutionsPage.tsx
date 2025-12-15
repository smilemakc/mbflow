import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Clock,
  CheckCircle,
  XCircle,
  RefreshCw,
  Loader2,
  Filter,
  ChevronDown,
  AlertCircle,
  Pause,
  ExternalLink
} from 'lucide-react';
import { useTranslation } from '@/store/translations';
import { executionService } from '@/services/executionService';
import { workflowService } from '@/services/workflowService';
import { Execution, ExecutionStatus } from '@/types/execution';
import { DAG } from '@/types';
import { Button } from '@/components/ui';

interface ExecutionFilters {
  workflow_id?: string;
  status?: ExecutionStatus;
  from?: string;
  to?: string;
}

export const ExecutionsPage: React.FC = () => {
  const t = useTranslation();
  const navigate = useNavigate();
  const [executions, setExecutions] = useState<Execution[]>([]);
  const [workflows, setWorkflows] = useState<DAG[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [limit] = useState(20);
  const [offset, setOffset] = useState(0);
  const [filters, setFilters] = useState<ExecutionFilters>({});
  const [showFilters, setShowFilters] = useState(false);
  const [retryingId, setRetryingId] = useState<string | null>(null);

  useEffect(() => {
    fetchWorkflows();
  }, []);

  useEffect(() => {
    fetchExecutions();
  }, [offset, filters]);

  const fetchWorkflows = async () => {
    try {
      const data = await workflowService.getAll();
      setWorkflows(data);
    } catch (error) {
      console.error('Failed to fetch workflows:', error);
    }
  };

  const fetchExecutions = async () => {
    setIsLoading(true);
    try {
      const response = await executionService.getAll({
        ...filters,
        limit,
        offset,
      });
      setExecutions(response.executions);
      setTotal(response.total);
    } catch (error) {
      console.error('Failed to fetch executions:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleRetry = async (executionId: string) => {
    setRetryingId(executionId);
    try {
      await executionService.retry(executionId);
      await fetchExecutions();
    } catch (error) {
      console.error('Failed to retry execution:', error);
    } finally {
      setRetryingId(null);
    }
  };

  const handleApplyFilters = () => {
    setOffset(0);
    setShowFilters(false);
  };

  const handleClearFilters = () => {
    setFilters({});
    setOffset(0);
    setShowFilters(false);
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
        return <CheckCircle className="text-green-600 dark:text-green-400" size={16} />;
      case 'failed':
        return <XCircle className="text-red-600 dark:text-red-400" size={16} />;
      case 'running':
        return <Loader2 className="text-blue-600 dark:text-blue-400 animate-spin" size={16} />;
      case 'pending':
        return <Clock className="text-yellow-600 dark:text-yellow-400" size={16} />;
      case 'cancelled':
        return <Pause className="text-gray-600 dark:text-gray-400" size={16} />;
      default:
        return <AlertCircle className="text-gray-600 dark:text-gray-400" size={16} />;
    }
  };

  const getStatusBadgeClass = (status: ExecutionStatus) => {
    switch (status) {
      case 'completed':
        return 'bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-400 dark:border-green-900/30';
      case 'failed':
        return 'bg-red-50 text-red-700 border-red-200 dark:bg-red-900/20 dark:text-red-400 dark:border-red-900/30';
      case 'running':
        return 'bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-900/20 dark:text-blue-400 dark:border-blue-900/30';
      case 'pending':
        return 'bg-yellow-50 text-yellow-700 border-yellow-200 dark:bg-yellow-900/20 dark:text-yellow-400 dark:border-yellow-900/30';
      case 'cancelled':
        return 'bg-gray-50 text-gray-700 border-gray-200 dark:bg-gray-900/20 dark:text-gray-400 dark:border-gray-900/30';
      default:
        return 'bg-gray-50 text-gray-700 border-gray-200 dark:bg-gray-900/20 dark:text-gray-400 dark:border-gray-900/30';
    }
  };

  const getStatusText = (status: ExecutionStatus) => {
    return t.executions.status[status] || status;
  };

  const getWorkflowName = (workflowId: string | undefined) => {
    if (!workflowId) return 'Unknown';
    const workflow = workflows.find(w => w.id === workflowId);
    return workflow?.name || workflowId.substring(0, 8);
  };

  return (
    <div className="flex-1 h-full overflow-y-auto bg-slate-50 dark:bg-slate-950 p-6 md:p-8">
      <div className="max-w-7xl mx-auto space-y-6">

        {/* Header */}
        <div className="flex justify-between items-start">
          <div>
            <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
              {t.executions.title}
            </h1>
            <p className="text-slate-500 dark:text-slate-400 mt-1">
              {t.executions.subtitle}
            </p>
          </div>

          <Button
            onClick={() => setShowFilters(!showFilters)}
            variant="outline"
            size="sm"
            icon={<Filter size={16} />}
          >
            {t.executions.filters}
            <ChevronDown
              size={16}
              className={`transition-transform ${showFilters ? 'rotate-180' : ''}`}
            />
          </Button>
        </div>

        {/* Filters Panel */}
        {showFilters && (
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-6 space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">

              {/* Workflow Filter */}
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                  {t.executions.table.workflow}
                </label>
                <select
                  value={filters.workflow_id || ''}
                  onChange={(e) => setFilters({ ...filters, workflow_id: e.target.value || undefined })}
                  className="w-full px-3 py-2 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                >
                  <option value="">{t.executions.allWorkflows}</option>
                  {workflows.map(workflow => (
                    <option key={workflow.id} value={workflow.id}>
                      {workflow.name}
                    </option>
                  ))}
                </select>
              </div>

              {/* Status Filter */}
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                  {t.executions.table.status}
                </label>
                <select
                  value={filters.status || ''}
                  onChange={(e) => setFilters({ ...filters, status: e.target.value as ExecutionStatus || undefined })}
                  className="w-full px-3 py-2 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                >
                  <option value="">{t.executions.allStatuses}</option>
                  <option value="pending">{t.executions.status.pending}</option>
                  <option value="running">{t.executions.status.running}</option>
                  <option value="completed">{t.executions.status.completed}</option>
                  <option value="failed">{t.executions.status.failed}</option>
                  <option value="cancelled">{t.executions.status.cancelled}</option>
                </select>
              </div>

              {/* Date Range */}
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                  {t.executions.dateRange}
                </label>
                <input
                  type="date"
                  value={filters.from || ''}
                  onChange={(e) => setFilters({ ...filters, from: e.target.value || undefined })}
                  className="w-full px-3 py-2 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>
            </div>

            {/* Filter Actions */}
            <div className="flex justify-end gap-3">
              <Button
                onClick={handleClearFilters}
                variant="ghost"
                size="sm"
              >
                {t.executions.clearFilters}
              </Button>
              <Button
                onClick={handleApplyFilters}
                variant="primary"
                size="sm"
              >
                {t.executions.applyFilters}
              </Button>
            </div>
          </div>
        )}

        {/* Executions Table */}
        <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl shadow-sm overflow-hidden">

          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="animate-spin text-blue-600 dark:text-blue-400" size={32} />
            </div>
          ) : executions.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-slate-500 dark:text-slate-400">
              <AlertCircle size={48} className="mb-4" />
              <p className="text-lg font-medium">{t.executions.noData}</p>
            </div>
          ) : (
            <>
              <div className="overflow-x-auto">
                <table className="w-full text-sm text-left">
                  <thead className="text-xs text-slate-500 uppercase bg-slate-50 dark:bg-slate-900/50 border-b border-slate-200 dark:border-slate-800">
                    <tr>
                      <th className="px-6 py-3 font-medium">{t.executions.table.id}</th>
                      <th className="px-6 py-3 font-medium">{t.executions.table.workflow}</th>
                      <th className="px-6 py-3 font-medium">{t.executions.table.status}</th>
                      <th className="px-6 py-3 font-medium">{t.executions.table.startedAt}</th>
                      <th className="px-6 py-3 font-medium">{t.executions.table.duration}</th>
                      <th className="px-6 py-3 font-medium">{t.executions.table.triggeredBy}</th>
                      <th className="px-6 py-3 font-medium text-right">{t.executions.table.actions}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                    {executions.map((execution) => (
                      <tr
                        key={execution.id}
                        className="hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors cursor-pointer group"
                        onClick={() => navigate(`/executions/${execution.id}`)}
                      >
                        <td className="px-6 py-4">
                          <span className="font-mono text-xs text-slate-500 dark:text-slate-400">
                            {execution.id.substring(0, 8)}
                          </span>
                        </td>
                        <td className="px-6 py-4">
                          <span className="font-medium text-slate-900 dark:text-slate-200">
                            {getWorkflowName(execution.workflow_id)}
                          </span>
                          <div className="text-xs text-slate-400 font-mono mt-0.5">
                            {execution.workflow_id.substring(0, 8)}
                          </div>
                        </td>
                        <td className="px-6 py-4">
                          <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border ${getStatusBadgeClass(execution.status)}`}>
                            {getStatusIcon(execution.status)}
                            {getStatusText(execution.status)}
                          </span>
                        </td>
                        <td className="px-6 py-4 text-slate-600 dark:text-slate-300">
                          {formatDate(execution.started_at)}
                        </td>
                        <td className="px-6 py-4 text-slate-500 dark:text-slate-400 font-mono text-xs">
                          {formatDuration(execution)}
                        </td>
                        <td className="px-6 py-4">
                          <span className="inline-flex items-center px-2 py-0.5 rounded-full bg-slate-100 dark:bg-slate-800 text-xs text-slate-600 dark:text-slate-300">
                            {execution.triggered_by || 'System'}
                          </span>
                        </td>
                        <td className="px-6 py-4 text-right">
                          <div className="flex justify-end gap-2" onClick={(e) => e.stopPropagation()}>
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
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              {/* Pagination */}
              <div className="px-6 py-4 border-t border-slate-200 dark:border-slate-800 flex items-center justify-between">
                <p className="text-sm text-slate-500 dark:text-slate-400">
                  {t.executions.pagination.showing} {offset + 1}-{Math.min(offset + limit, total)} {t.executions.pagination.of} {total} {t.executions.pagination.results}
                </p>
                <div className="flex gap-2">
                  <Button
                    onClick={() => setOffset(Math.max(0, offset - limit))}
                    disabled={offset === 0}
                    variant="outline"
                    size="sm"
                  >
                    Previous
                  </Button>
                  <Button
                    onClick={() => setOffset(offset + limit)}
                    disabled={offset + limit >= total}
                    variant="outline"
                    size="sm"
                  >
                    Next
                  </Button>
                </div>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
};

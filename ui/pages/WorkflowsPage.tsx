import React, { useState, useEffect } from 'react';
import { BookOpen, GitBranch, Plus, Search, AlertCircle } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { WorkflowStatus } from '@/types';
import { WorkflowVariablesGuide } from '@/components/builder/WorkflowVariablesGuide.tsx';
import { Button, ConfirmModal } from '@/components/ui';
import { EmptyState } from '@/components/ui/EmptyState';
import { LoadingState } from '@/components/ui/LoadingState';
import { Pagination } from '@/components/ui/Pagination';
import { FilterBar, FilterSearch, FilterSelect } from '@/components/ui/FilterBar';
import { WorkflowCard } from '@/components/workflows/WorkflowCard';
import { useTranslation } from '../store/translations';
import { useWorkflows } from '@/hooks/useWorkflows';
import { useWorkflowFilters } from '@/hooks/useWorkflowFilters';
import { usePagination } from '@/hooks/usePagination';

export const WorkflowsPage: React.FC = () => {
  const t = useTranslation();
  const navigate = useNavigate();
  const [showVariablesGuide, setShowVariablesGuide] = useState(false);
  const [workflowToDelete, setWorkflowToDelete] = useState<{ id: string; name: string } | null>(
    null
  );

  const { workflows, isLoading, error, loadWorkflows, cloneWorkflow, deleteWorkflow } =
    useWorkflows({
      translations: {
        cloneFailed: t.workflows.errors.cloneFailed,
        cloneMessage: t.workflows.errors.cloneMessage,
        deleteFailed: t.workflows.errors.deleteFailed,
        deleteMessage: t.workflows.errors.deleteMessage,
      },
    });

  const { filteredWorkflows, searchQuery, setSearchQuery, statusFilter, setStatusFilter } =
    useWorkflowFilters(workflows);

  const pagination = usePagination(filteredWorkflows, 12);

  const handleCreateNew = () => {
    navigate('/builder');
  };

  const handleEdit = (workflowId: string) => {
    navigate(`/builder/${workflowId}`);
  };

  const handleDelete = async () => {
    if (!workflowToDelete) return;

    await deleteWorkflow(workflowToDelete.id);
    setWorkflowToDelete(null);
  };

  const formatDate = (dateStr: string): string => {
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

    if (diffDays === 0) return t.workflows.today;
    if (diffDays === 1) return t.workflows.yesterday;
    if (diffDays < 7) return `${diffDays} ${t.workflows.daysAgo}`;
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    });
  };

  const statusOptions = [
    { label: t.workflows.allStatus, value: 'all' as const },
    { label: t.workflows.draft, value: 'draft' as const },
    { label: t.workflows.active, value: 'active' as const },
    { label: t.workflows.inactive, value: 'inactive' as const },
    { label: t.workflows.archived, value: 'archived' as const },
  ];

  return (
    <div className="flex-1 h-full overflow-y-auto bg-slate-50 dark:bg-slate-950 p-6 md:p-8">
      <div className="max-w-7xl mx-auto space-y-6">
        <div className="flex justify-between items-end">
          <div>
            <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
              {t.workflows.title}
            </h1>
            <p className="text-slate-500 dark:text-slate-400 mt-1">{t.workflows.subtitle}</p>
          </div>
          <div className="flex items-center gap-3">
            <Button
              onClick={() => setShowVariablesGuide(!showVariablesGuide)}
              variant={showVariablesGuide ? 'primary' : 'outline'}
              size="sm"
              icon={<BookOpen size={16} />}
            >
              {t.workflows.variablesGuide}
            </Button>
            <Button
              onClick={handleCreateNew}
              variant="primary"
              size="sm"
              icon={<Plus size={16} />}
            >
              {t.workflows.createNew}
            </Button>
          </div>
        </div>

        {showVariablesGuide && (
          <WorkflowVariablesGuide isModal={true} onClose={() => setShowVariablesGuide(false)} />
        )}

        <FilterBar>
          <div className="flex-1 relative">
            <Search
              size={18}
              className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
            />
            <FilterSearch
              value={searchQuery}
              onChange={setSearchQuery}
              placeholder={t.workflows.searchPlaceholder}
              className="w-full pl-10"
            />
          </div>
          <FilterSelect
            value={statusFilter}
            onChange={setStatusFilter}
            options={statusOptions}
          />
        </FilterBar>

        <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-4">
          <p className="text-sm text-slate-600 dark:text-slate-400">
            {t.workflows.showing}{' '}
            <span className="font-medium text-slate-900 dark:text-white">
              {filteredWorkflows.length}
            </span>{' '}
            {filteredWorkflows.length !== 1 ? t.workflows.workflowsCount : t.workflows.workflow}
            {searchQuery && (
              <>
                {' '}
                {t.workflows.matching} "
                <span className="font-medium">{searchQuery}</span>"
              </>
            )}
          </p>
        </div>

        {isLoading && <LoadingState />}

        {error && !isLoading && (
          <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-900/30 rounded-xl p-6 text-center">
            <AlertCircle size={32} className="mx-auto mb-3 text-red-600 dark:text-red-400" />
            <p className="text-red-800 dark:text-red-400 font-medium">{error}</p>
            <div className="mt-4">
              <Button onClick={loadWorkflows} variant="danger" size="sm">
                {t.workflows.tryAgain}
              </Button>
            </div>
          </div>
        )}

        {!isLoading && !error && filteredWorkflows.length === 0 && (
          <EmptyState
            icon={GitBranch}
            title={
              searchQuery || statusFilter !== 'all'
                ? t.workflows.noWorkflowsFound
                : t.workflows.noWorkflowsYet
            }
            description={
              searchQuery || statusFilter !== 'all'
                ? t.workflows.adjustFilters
                : t.workflows.getStarted
            }
            action={
              !searchQuery && statusFilter === 'all'
                ? {
                    label: t.workflows.createFirst,
                    onClick: handleCreateNew,
                    icon: <Plus size={16} />,
                  }
                : undefined
            }
          />
        )}

        {!isLoading && !error && pagination.currentItems.length > 0 && (
          <>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {pagination.currentItems.map((workflow) => (
                <WorkflowCard
                  key={workflow.id}
                  workflow={workflow}
                  onEdit={handleEdit}
                  onClone={cloneWorkflow}
                  onDelete={(id, name) => setWorkflowToDelete({ id, name })}
                  formatDate={formatDate}
                  translations={{
                    edit: t.workflows.edit,
                    cloneTooltip: t.workflows.cloneTooltip,
                    deleteTooltip: t.workflows.deleteTooltip,
                    created: t.workflows.created,
                    updated: t.workflows.updated,
                    nodesCount: t.workflows.nodesCount,
                    node: t.workflows.node,
                  }}
                />
              ))}
            </div>

            {pagination.totalPages > 1 && (
              <Pagination
                currentPage={pagination.currentPage}
                totalPages={pagination.totalPages}
                onPageChange={pagination.goToPage}
                size="sm"
              />
            )}
          </>
        )}
      </div>

      <ConfirmModal
        isOpen={!!workflowToDelete}
        onClose={() => setWorkflowToDelete(null)}
        onConfirm={handleDelete}
        title={t.workflows.deleteModal.title}
        message={t.workflows.deleteModal.message}
        confirmText={t.workflows.deleteModal.confirm}
        variant="danger"
      />
    </div>
  );
};

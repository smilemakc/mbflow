import React from 'react';
import { Calendar, Clock, Copy, Edit, GitBranch, Trash2 } from 'lucide-react';
import { DAG, WorkflowStatus } from '@/types';
import { Button } from '@/components/ui';
import { StatusBadge } from '@/components/ui/StatusBadge';

interface WorkflowCardProps {
  workflow: DAG;
  onEdit: (id: string) => void;
  onClone: (workflow: DAG) => void;
  onDelete: (id: string, name: string) => void;
  formatDate: (date: string) => string;
  translations: {
    edit: string;
    cloneTooltip: string;
    deleteTooltip: string;
    created: string;
    updated: string;
    nodesCount: string;
    node: string;
  };
}

const STATUS_CONFIG: Record<
  WorkflowStatus,
  {
    bg: string;
    text: string;
    border: string;
    icon: React.ElementType;
  }
> = {
  draft: {
    bg: 'bg-slate-100 dark:bg-slate-800',
    text: 'text-slate-700 dark:text-slate-300',
    border: 'border-slate-200 dark:border-slate-700',
    icon: () => null,
  },
  active: {
    bg: 'bg-green-50 dark:bg-green-900/20',
    text: 'text-green-700 dark:text-green-400',
    border: 'border-green-200 dark:border-green-900/30',
    icon: () => null,
  },
  inactive: {
    bg: 'bg-orange-50 dark:bg-orange-900/20',
    text: 'text-orange-700 dark:text-orange-400',
    border: 'border-orange-200 dark:border-orange-900/30',
    icon: () => null,
  },
  archived: {
    bg: 'bg-slate-50 dark:bg-slate-900/20',
    text: 'text-slate-600 dark:text-slate-500',
    border: 'border-slate-200 dark:border-slate-800',
    icon: () => null,
  },
};

export const WorkflowCard: React.FC<WorkflowCardProps> = ({
  workflow,
  onEdit,
  onClone,
  onDelete,
  formatDate,
  translations,
}) => {
  const statusConfig = STATUS_CONFIG[workflow.status];

  return (
    <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl shadow-sm hover:shadow-md transition-all group">
      <div className="p-5 border-b border-slate-100 dark:border-slate-800">
        <div className="flex items-start justify-between mb-3">
          <h3 className="text-lg font-bold text-slate-900 dark:text-white line-clamp-1 flex-1 pr-2">
            {workflow.name}
          </h3>
          <span
            className={`inline-flex items-center px-2 py-1 rounded text-xs font-medium border ${statusConfig.bg} ${statusConfig.text} ${statusConfig.border} shrink-0`}
          >
            {workflow.status.charAt(0).toUpperCase() + workflow.status.slice(1)}
          </span>
        </div>

        {workflow.description && (
          <p className="text-sm text-slate-600 dark:text-slate-400 line-clamp-2">
            {workflow.description}
          </p>
        )}
      </div>

      <div className="p-5 space-y-3">
        <div className="flex items-center gap-4 text-sm">
          <div className="flex items-center text-slate-600 dark:text-slate-400">
            <GitBranch size={14} className="mr-1.5" />
            <span className="font-medium">{workflow.nodes?.length || 0}</span>
            <span className="ml-1">
              {workflow.nodes?.length !== 1 ? translations.nodesCount : translations.node}
            </span>
          </div>
        </div>

        <div className="space-y-1.5 text-xs">
          <div className="flex items-center text-slate-500 dark:text-slate-500">
            <Calendar size={12} className="mr-1.5" />
            <span>
              {translations.created} {formatDate(workflow.createdAt)}
            </span>
          </div>
          <div className="flex items-center text-slate-500 dark:text-slate-500">
            <Clock size={12} className="mr-1.5" />
            <span>
              {translations.updated} {formatDate(workflow.updatedAt)}
            </span>
          </div>
        </div>
      </div>

      <div className="p-4 bg-slate-50 dark:bg-slate-900/50 border-t border-slate-100 dark:border-slate-800 flex items-center gap-2">
        <Button
          onClick={() => onEdit(workflow.id)}
          variant="primary"
          size="sm"
          icon={<Edit size={14} />}
          className="flex-1"
        >
          {translations.edit}
        </Button>
        <Button
          onClick={() => onClone(workflow)}
          variant="outline"
          size="sm"
          icon={<Copy size={14} />}
          title={translations.cloneTooltip}
        />
        <Button
          onClick={() => onDelete(workflow.id, workflow.name)}
          variant="danger"
          size="sm"
          icon={<Trash2 size={14} />}
          title={translations.deleteTooltip}
        />
      </div>
    </div>
  );
};

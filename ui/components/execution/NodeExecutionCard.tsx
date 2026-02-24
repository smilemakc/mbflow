import React from 'react';
import {
  ChevronDown,
  ChevronRight,
  XCircle,
  ExternalLink,
  Terminal,
  FileJson,
  AlertTriangle,
  Layers,
  Workflow
} from 'lucide-react';
import { NodeExecution } from '@/types/execution';
import { CopyButton } from './CopyButton';
import { JsonViewer } from './JsonViewer';
import { getStatusIcon, getStatusBadgeClass, formatDuration } from './executionUtils';
import { useTranslation } from '@/store/translations';

interface NodeExecutionCardProps {
  nodeExec: NodeExecution;
  index: number;
  isExpanded: boolean;
  onToggle: () => void;
}

const getNodeTypeIcon = (nodeType?: string) => {
  switch (nodeType) {
    case 'http':
      return <ExternalLink size={14} />;
    case 'llm':
      return <Terminal size={14} />;
    case 'transform':
      return <FileJson size={14} />;
    case 'conditional':
      return <AlertTriangle size={14} />;
    case 'merge':
      return <Layers size={14} />;
    default:
      return <Workflow size={14} />;
  }
};

export const NodeExecutionCard: React.FC<NodeExecutionCardProps> = ({
  nodeExec,
  index,
  isExpanded,
  onToggle
}) => {
  const t = useTranslation();

  const statusText = t.executions?.status?.[nodeExec.status] || nodeExec.status;

  return (
    <div
      className={`bg-white dark:bg-slate-900 rounded-xl border transition-all duration-200 ${
        isExpanded
          ? 'border-blue-300 dark:border-blue-700 shadow-lg'
          : 'border-slate-200 dark:border-slate-800 hover:border-slate-300 dark:hover:border-slate-700'
      }`}
    >
      {/* Card Header */}
      <div
        className="flex items-center justify-between px-4 py-3 cursor-pointer"
        onClick={onToggle}
      >
        <div className="flex items-center gap-3">
          <div className="flex items-center justify-center w-8 h-8 rounded-lg bg-slate-100 dark:bg-slate-800 text-slate-500 dark:text-slate-400 text-sm font-bold">
            {index + 1}
          </div>
          <div className="flex items-center gap-2">
            <span className="text-slate-400 dark:text-slate-500">
              {getNodeTypeIcon(nodeExec.node_type)}
            </span>
            <div>
              <span className="font-medium text-slate-900 dark:text-white">
                {nodeExec.node_name || nodeExec.node_id}
              </span>
              {nodeExec.node_type && (
                <span className="ml-2 text-xs text-slate-400 dark:text-slate-500 bg-slate-100 dark:bg-slate-800 px-1.5 py-0.5 rounded">
                  {nodeExec.node_type}
                </span>
              )}
            </div>
          </div>
        </div>

        <div className="flex items-center gap-4">
          {/* Duration */}
          <span className="text-xs font-mono text-slate-500 dark:text-slate-400">
            {formatDuration(nodeExec.started_at, nodeExec.completed_at)}
          </span>

          {/* Status Badge */}
          <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border ${getStatusBadgeClass(nodeExec.status)}`}>
            {getStatusIcon(nodeExec.status)}
            {statusText}
          </span>

          {/* Expand/Collapse */}
          <div className="text-slate-400">
            {isExpanded ? <ChevronDown size={18} /> : <ChevronRight size={18} />}
          </div>
        </div>
      </div>

      {/* Card Body - Expanded Content */}
      {isExpanded && (
        <div className="px-4 pb-4 space-y-4 border-t border-slate-100 dark:border-slate-800 pt-4">
          {/* Error Section */}
          {nodeExec.error && (
            <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-900/30 rounded-lg p-4">
              <div className="flex items-start gap-3">
                <XCircle className="text-red-600 dark:text-red-400 mt-0.5 shrink-0" size={18} />
                <div className="flex-1 min-w-0">
                  <h4 className="text-sm font-semibold text-red-900 dark:text-red-300 mb-1">
                    Error
                  </h4>
                  <pre className="text-sm text-red-800 dark:text-red-400 font-mono whitespace-pre-wrap break-words">
                    {nodeExec.error}
                  </pre>
                </div>
                <CopyButton text={nodeExec.error} />
              </div>
            </div>
          )}

          {/* Metadata Row */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div>
              <label className="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                Node ID
              </label>
              <p className="text-sm font-mono text-slate-700 dark:text-slate-300 mt-1 break-all">
                {nodeExec.node_id}
              </p>
            </div>
            <div>
              <label className="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                Started At
              </label>
              <p className="text-sm text-slate-700 dark:text-slate-300 mt-1">
                {new Date(nodeExec.started_at).toLocaleString()}
              </p>
            </div>
            <div>
              <label className="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                Completed At
              </label>
              <p className="text-sm text-slate-700 dark:text-slate-300 mt-1">
                {nodeExec.completed_at ? new Date(nodeExec.completed_at).toLocaleString() : '-'}
              </p>
            </div>
            {nodeExec.retry_count !== undefined && nodeExec.retry_count > 0 && (
              <div>
                <label className="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                  Retry Count
                </label>
                <p className="text-sm text-orange-600 dark:text-orange-400 mt-1 font-medium">
                  {nodeExec.retry_count}
                </p>
              </div>
            )}
          </div>

          {/* Input/Output Grid */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <JsonViewer
              data={nodeExec.input}
              title="Input"
              defaultExpanded={true}
              maxHeight="250px"
            />
            <JsonViewer
              data={nodeExec.output}
              title="Output"
              defaultExpanded={true}
              maxHeight="250px"
            />
          </div>
        </div>
      )}
    </div>
  );
};

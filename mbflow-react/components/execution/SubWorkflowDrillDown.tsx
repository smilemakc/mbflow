import React from 'react';

interface ChildExecution {
  index: number;
  status: string;
  execution_id: string;
  item_key?: string;
  duration_ms?: number;
  error?: string;
}

interface SubWorkflowDrillDownProps {
  nodeName: string;
  children: ChildExecution[];
  onClose: () => void;
  onRetryFailed?: () => void;
}

const formatMs = (ms: number): string => {
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
};

const statusIcon = (status: string): string => {
  switch (status) {
    case 'completed': return '✓';
    case 'failed': return '✗';
    case 'running': return '⟳';
    default: return '◻';
  }
};

const statusColor = (status: string): string => {
  switch (status) {
    case 'completed': return 'text-green-600 dark:text-green-400';
    case 'failed': return 'text-red-600 dark:text-red-400';
    case 'running': return 'text-blue-600 dark:text-blue-400';
    default: return 'text-gray-400';
  }
};

export const SubWorkflowDrillDown: React.FC<SubWorkflowDrillDownProps> = ({
  nodeName, children, onClose, onRetryFailed,
}) => {
  const failedCount = children.filter(c => c.status === 'failed').length;

  return (
    <div className="fixed inset-y-0 right-0 w-96 bg-white dark:bg-gray-900 border-l border-gray-200 dark:border-gray-700 shadow-xl z-50 flex flex-col">
      {/* Header */}
      <div className="p-4 border-b border-gray-200 dark:border-gray-700 flex justify-between items-center">
        <div>
          <h3 className="font-medium text-sm">{nodeName}</h3>
          <p className="text-xs text-gray-500">{children.length} items</p>
        </div>
        <button onClick={onClose} className="text-gray-400 hover:text-gray-600 text-lg">&times;</button>
      </div>

      {/* Table */}
      <div className="flex-1 overflow-y-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 bg-gray-50 dark:bg-gray-800">
            <tr>
              <th className="px-3 py-2 text-left">#</th>
              <th className="px-3 py-2 text-left">Key</th>
              <th className="px-3 py-2 text-left">Status</th>
              <th className="px-3 py-2 text-right">Duration</th>
            </tr>
          </thead>
          <tbody>
            {children.map((child) => (
              <tr
                key={child.index}
                className="border-b border-gray-100 dark:border-gray-800 hover:bg-gray-50 dark:hover:bg-gray-800 cursor-pointer"
              >
                <td className="px-3 py-2 tabular-nums">{child.index}</td>
                <td className="px-3 py-2 truncate max-w-[120px]">{child.item_key || `Item ${child.index}`}</td>
                <td className="px-3 py-2">
                  <span className={statusColor(child.status)}>
                    {statusIcon(child.status)} {child.status}
                  </span>
                  {child.status === 'failed' && child.error && (
                    <span className="ml-1 text-red-500 truncate" title={child.error}>
                      — {child.error}
                    </span>
                  )}
                </td>
                <td className="px-3 py-2 text-right tabular-nums">
                  {child.duration_ms ? formatMs(child.duration_ms) : '—'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Footer */}
      {failedCount > 0 && onRetryFailed && (
        <div className="p-3 border-t border-gray-200 dark:border-gray-700">
          <button
            onClick={onRetryFailed}
            className="w-full px-3 py-1.5 text-xs bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 rounded hover:bg-red-100 dark:hover:bg-red-900/30"
          >
            Retry {failedCount} failed
          </button>
        </div>
      )}
    </div>
  );
};

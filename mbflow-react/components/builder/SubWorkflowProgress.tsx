import React from 'react';

interface SubWorkflowProgressProps {
  total: number;
  completed: number;
  failed: number;
  running: number;
}

export const SubWorkflowProgress: React.FC<SubWorkflowProgressProps> = ({
  total, completed, failed, running,
}) => {
  if (total === 0) return null;

  const pending = total - completed - failed - running;

  return (
    <div className="mt-2 px-1">
      {/* Progress bar */}
      <div className="flex items-center gap-2 mb-1">
        <div className="flex-1 h-1.5 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
          <div className="h-full flex">
            <div
              className="bg-green-500 transition-all duration-300"
              style={{ width: `${(completed / total) * 100}%` }}
            />
            <div
              className="bg-red-500 transition-all duration-300"
              style={{ width: `${(failed / total) * 100}%` }}
            />
            <div
              className="bg-blue-500 animate-pulse transition-all duration-300"
              style={{ width: `${(running / total) * 100}%` }}
            />
          </div>
        </div>
        <span className="text-[10px] text-gray-500 dark:text-gray-400 tabular-nums">
          {completed + failed}/{total}
        </span>
      </div>
      {/* Status counts */}
      <div className="flex gap-2 text-[9px] text-gray-500 dark:text-gray-400">
        {completed > 0 && <span className="text-green-600 dark:text-green-400">done:{completed}</span>}
        {running > 0 && <span className="text-blue-600 dark:text-blue-400">run:{running}</span>}
        {failed > 0 && <span className="text-red-600 dark:text-red-400">fail:{failed}</span>}
        {pending > 0 && <span>wait:{pending}</span>}
      </div>
    </div>
  );
};

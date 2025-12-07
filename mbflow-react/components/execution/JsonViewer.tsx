import React, { useState } from 'react';
import { ChevronDown, ChevronRight } from 'lucide-react';
import { CopyButton } from './CopyButton';

interface JsonViewerProps {
  data: any;
  title: string;
  defaultExpanded?: boolean;
  maxHeight?: string;
}

export const JsonViewer: React.FC<JsonViewerProps> = ({
  data,
  title,
  defaultExpanded = true,
  maxHeight = '300px'
}) => {
  const [isExpanded, setIsExpanded] = useState(defaultExpanded);
  const jsonString = JSON.stringify(data, null, 2);

  if (!data || (typeof data === 'object' && Object.keys(data).length === 0)) {
    return (
      <div className="bg-slate-50 dark:bg-slate-800/50 rounded-lg p-4 border border-slate-200 dark:border-slate-700">
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium text-slate-500 dark:text-slate-400">{title}</span>
        </div>
        <p className="text-sm text-slate-400 dark:text-slate-500 mt-2 italic">
          No data
        </p>
      </div>
    );
  }

  const fieldCount = typeof data === 'object' ? Object.keys(data).length : 1;

  return (
    <div className="bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700 overflow-hidden">
      <div
        className="flex items-center justify-between px-4 py-3 bg-slate-100 dark:bg-slate-800 cursor-pointer hover:bg-slate-150 dark:hover:bg-slate-750 transition-colors"
        onClick={() => setIsExpanded(!isExpanded)}
      >
        <div className="flex items-center gap-2">
          {isExpanded ? (
            <ChevronDown size={16} className="text-slate-500" />
          ) : (
            <ChevronRight size={16} className="text-slate-500" />
          )}
          {title && (
            <span className="text-sm font-medium text-slate-700 dark:text-slate-300">{title}</span>
          )}
          <span className="text-xs text-slate-400 dark:text-slate-500">
            ({fieldCount} {fieldCount === 1 ? 'field' : 'fields'})
          </span>
        </div>
        <CopyButton text={jsonString} />
      </div>
      {isExpanded && (
        <div className="p-4 overflow-auto" style={{ maxHeight }}>
          <pre className="text-xs font-mono text-slate-700 dark:text-slate-300 whitespace-pre-wrap break-words">
            {jsonString}
          </pre>
        </div>
      )}
    </div>
  );
};

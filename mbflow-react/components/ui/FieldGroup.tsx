import React, { useState } from 'react';
import { ChevronDown, ChevronRight } from 'lucide-react';

interface FieldGroupProps {
  title: string;
  children: React.ReactNode;
  collapsible?: boolean;
  defaultExpanded?: boolean;
  icon?: React.ReactNode;
}

export const FieldGroup: React.FC<FieldGroupProps> = ({
  title,
  children,
  collapsible = false,
  defaultExpanded = true,
  icon,
}) => {
  const [isExpanded, setIsExpanded] = useState(defaultExpanded);

  const handleToggle = () => {
    if (collapsible) {
      setIsExpanded(!isExpanded);
    }
  };

  return (
    <div className="border border-slate-200 dark:border-slate-800 rounded-lg overflow-hidden">
      <div
        className={`bg-slate-50 dark:bg-slate-900/50 px-4 py-3 flex items-center justify-between ${
          collapsible ? 'cursor-pointer hover:bg-slate-100 dark:hover:bg-slate-900' : ''
        }`}
        onClick={handleToggle}
      >
        <div className="flex items-center gap-2">
          {icon && <span className="text-slate-600 dark:text-slate-400">{icon}</span>}
          <h4 className="text-sm font-semibold text-slate-900 dark:text-white">{title}</h4>
        </div>
        {collapsible && (
          <div className="text-slate-400">
            {isExpanded ? <ChevronDown size={18} /> : <ChevronRight size={18} />}
          </div>
        )}
      </div>
      {isExpanded && (
        <div className="p-4 space-y-4 bg-white dark:bg-slate-900">{children}</div>
      )}
    </div>
  );
};

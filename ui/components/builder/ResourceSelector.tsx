import React from 'react';
import { useDagStore } from '@/store/dagStore';
import { Database } from 'lucide-react';

interface ResourceSelectorProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  label?: string;
  hint?: string;
  resourceType?: string;
  className?: string;
}

export const ResourceSelector: React.FC<ResourceSelectorProps> = ({
  value,
  onChange,
  placeholder = 'Select a resource...',
  label,
  hint,
  resourceType,
  className = '',
}) => {
  const { resources } = useDagStore();

  const filteredResources = resourceType
    ? resources.filter(r => r.resource_type === resourceType)
    : resources;

  return (
    <div className={className}>
      {label && (
        <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-1.5">
          {label}
        </label>
      )}
      <div className="relative">
        <Database className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400 pointer-events-none" />
        <select
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="w-full pl-10 pr-3 py-2 bg-white dark:bg-slate-950 border border-gray-300 dark:border-gray-700 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-colors"
        >
          <option value="">{placeholder}</option>
          {filteredResources.map(r => (
            <option key={r.resource_id} value={`{{resource.${r.alias}}}`}>
              {r.alias} {r.resource_name && `(${r.resource_name})`}
            </option>
          ))}
        </select>
      </div>
      {hint && (
        <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">{hint}</p>
      )}
      {filteredResources.length === 0 && (
        <p className="mt-1 text-xs text-amber-600 dark:text-amber-400">
          No resources attached. Add resources via the Resources panel.
        </p>
      )}
    </div>
  );
};

export default ResourceSelector;

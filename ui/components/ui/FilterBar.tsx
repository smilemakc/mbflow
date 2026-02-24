/**
 * FilterBar component
 * Composable filter bar for tables and lists with consistent styling.
 */

import React from 'react';

interface FilterBarProps {
  children: React.ReactNode;
  className?: string;
}

interface FilterSelectProps<T extends string> {
  label?: string;
  value: T;
  onChange: (value: T) => void;
  options: Array<{ label: string; value: T }>;
  placeholder?: string;
  className?: string;
}

interface FilterSearchProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
}

interface FilterResultsProps {
  total: number;
  filtered?: number;
  itemLabel?: string;
}

export const FilterBar: React.FC<FilterBarProps> = ({ children, className = '' }) => {
  return (
    <div
      className={`bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-4 flex flex-wrap gap-4 items-center ${className}`}
    >
      {children}
    </div>
  );
};

export function FilterSelect<T extends string>({
  label,
  value,
  onChange,
  options,
  placeholder,
  className = '',
}: FilterSelectProps<T>) {
  return (
    <div className={`flex items-center gap-2 ${className}`}>
      {label && (
        <span className="text-sm font-medium text-slate-700 dark:text-slate-300">
          {label}
        </span>
      )}
      <select
        value={value}
        onChange={(e) => onChange(e.target.value as T)}
        className="px-3 py-1.5 text-sm bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-slate-700 dark:text-slate-300"
      >
        {placeholder && <option value="">{placeholder}</option>}
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </div>
  );
}

export const FilterSearch: React.FC<FilterSearchProps> = ({
  value,
  onChange,
  placeholder = 'Search...',
  className = '',
}) => {
  return (
    <input
      type="text"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      className={`px-3 py-1.5 text-sm bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-slate-700 dark:text-slate-300 placeholder:text-slate-400 dark:placeholder:text-slate-500 ${className}`}
    />
  );
};

export const FilterResults: React.FC<FilterResultsProps> = ({
  total,
  filtered,
  itemLabel = 'items',
}) => {
  const displayCount = filtered !== undefined && filtered !== total;

  return (
    <div className="ml-auto text-sm text-slate-500 dark:text-slate-400">
      {displayCount ? (
        <>
          {filtered} of {total} {itemLabel}
        </>
      ) : (
        <>
          {total} {itemLabel}
        </>
      )}
    </div>
  );
};

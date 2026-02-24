import React from 'react';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { LoadingState } from './LoadingState';
import { EmptyState } from './EmptyState';
import { ErrorBanner } from './ErrorBanner';
import { Button } from './Button';

export interface Column<T> {
  key: string;
  header: string | React.ReactNode;
  width?: string;
  align?: 'left' | 'center' | 'right';
  sortable?: boolean;
  render?: (item: T, index: number) => React.ReactNode;
}

export interface DataTableProps<T> {
  data: T[];
  columns: Column<T>[];
  keyExtractor: (item: T) => string;

  loading?: boolean;
  error?: string | null;

  emptyIcon?: React.ElementType;
  emptyTitle?: string;
  emptyDescription?: string;
  emptyAction?: {
    label: string;
    onClick: () => void;
    icon?: React.ReactNode;
  };

  onRowClick?: (item: T) => void;
  rowClassName?: (item: T) => string;

  actions?: (item: T) => React.ReactNode;
  actionsHeader?: string;

  pagination?: {
    offset: number;
    limit: number;
    total: number;
    onOffsetChange: (offset: number) => void;
  };

  className?: string;
  compact?: boolean;
}

export const DataTable = <T,>({
  data,
  columns,
  keyExtractor,
  loading = false,
  error = null,
  emptyIcon,
  emptyTitle = 'No data found',
  emptyDescription = 'There are no items to display',
  emptyAction,
  onRowClick,
  rowClassName,
  actions,
  actionsHeader = 'Actions',
  pagination,
  className = '',
  compact = false,
}: DataTableProps<T>) => {
  const hasActions = !!actions;
  const paddingClass = compact ? 'px-3 py-2' : 'px-6 py-4';
  const headerPaddingClass = compact ? 'px-3 py-2' : 'px-4 py-3';

  const getAlignClass = (align?: 'left' | 'center' | 'right') => {
    switch (align) {
      case 'center':
        return 'text-center';
      case 'right':
        return 'text-right';
      default:
        return 'text-left';
    }
  };

  const handlePrevPage = () => {
    if (!pagination) return;
    const newOffset = Math.max(0, pagination.offset - pagination.limit);
    pagination.onOffsetChange(newOffset);
  };

  const handleNextPage = () => {
    if (!pagination) return;
    if (pagination.offset + pagination.limit < pagination.total) {
      pagination.onOffsetChange(pagination.offset + pagination.limit);
    }
  };

  if (error) {
    return (
      <div className={className}>
        <ErrorBanner message={error} />
      </div>
    );
  }

  if (loading && data.length === 0) {
    return (
      <div className={`bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl ${className}`}>
        <LoadingState />
      </div>
    );
  }

  if (!loading && data.length === 0 && emptyIcon) {
    return (
      <div className={className}>
        <EmptyState
          icon={emptyIcon}
          title={emptyTitle}
          description={emptyDescription}
          action={emptyAction}
        />
      </div>
    );
  }

  return (
    <div className={`bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl shadow-sm overflow-hidden ${className}`}>
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead className="bg-slate-50 dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700">
            <tr>
              {columns.map((column) => (
                <th
                  key={column.key}
                  className={`${headerPaddingClass} text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider ${getAlignClass(column.align)}`}
                  style={column.width ? { width: column.width } : undefined}
                >
                  {column.header}
                </th>
              ))}
              {hasActions && (
                <th className={`${headerPaddingClass} text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider text-right`}>
                  {actionsHeader}
                </th>
              )}
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
            {data.map((item, index) => {
              const key = keyExtractor(item);
              const clickable = !!onRowClick;
              const customRowClass = rowClassName?.(item) || '';
              const rowClass = `
                ${clickable ? 'cursor-pointer' : ''}
                hover:bg-slate-50 dark:hover:bg-slate-800/50
                transition-colors
                ${customRowClass}
              `.trim();

              return (
                <tr
                  key={key}
                  className={rowClass}
                  onClick={clickable ? () => onRowClick(item) : undefined}
                >
                  {columns.map((column) => (
                    <td
                      key={`${key}-${column.key}`}
                      className={`${paddingClass} ${getAlignClass(column.align)}`}
                      style={column.width ? { width: column.width } : undefined}
                    >
                      {column.render ? (
                        column.render(item, index)
                      ) : (
                        <span className="text-sm text-slate-900 dark:text-slate-200">
                          {String((item as any)[column.key] ?? '')}
                        </span>
                      )}
                    </td>
                  ))}
                  {hasActions && (
                    <td
                      className={`${paddingClass} text-right`}
                      onClick={(e) => e.stopPropagation()}
                    >
                      <div className="flex items-center justify-end gap-2">
                        {actions(item)}
                      </div>
                    </td>
                  )}
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {pagination && pagination.total > 0 && (
        <div className="flex items-center justify-between px-6 py-4 border-t border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/50">
          <span className="text-sm text-slate-500 dark:text-slate-400">
            Showing {pagination.offset + 1} - {Math.min(pagination.offset + pagination.limit, pagination.total)} of {pagination.total}
          </span>
          <div className="flex items-center gap-2">
            <Button
              onClick={handlePrevPage}
              disabled={pagination.offset === 0}
              variant="outline"
              size="sm"
              icon={<ChevronLeft size={14} />}
            >
              Previous
            </Button>
            <Button
              onClick={handleNextPage}
              disabled={pagination.offset + pagination.limit >= pagination.total}
              variant="outline"
              size="sm"
              icon={<ChevronRight size={14} />}
            >
              Next
            </Button>
          </div>
        </div>
      )}
    </div>
  );
};

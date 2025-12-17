import React from 'react';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { Button } from './Button';

export interface PaginationProps {
  // For page-based (client-side) pagination
  currentPage?: number;
  totalPages?: number;
  onPageChange?: (page: number) => void;

  // For offset-based (server-side) pagination
  offset?: number;
  limit?: number;
  total?: number;
  onOffsetChange?: (offset: number) => void;

  // Display options
  showPageNumbers?: boolean;
  maxVisiblePages?: number;
  showItemCount?: boolean;
  itemLabel?: string;

  // Styling
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

const cn = (...classes: (string | undefined | null | false)[]): string => {
  return classes.filter(Boolean).join(' ');
};

export const Pagination: React.FC<PaginationProps> = ({
  currentPage,
  totalPages,
  onPageChange,
  offset,
  limit,
  total,
  onOffsetChange,
  showPageNumbers = false,
  maxVisiblePages = 5,
  showItemCount = false,
  itemLabel = 'items',
  size = 'sm',
  className,
}) => {
  // Detect mode
  const isOffsetMode = offset !== undefined && limit !== undefined && total !== undefined && onOffsetChange;
  const isPageMode = currentPage !== undefined && totalPages !== undefined && onPageChange;

  if (!isOffsetMode && !isPageMode) {
    console.warn('Pagination: Either page-based or offset-based props must be provided');
    return null;
  }

  // Calculate values for offset mode
  const currentPageOffset = isOffsetMode ? Math.floor(offset / limit) + 1 : 0;
  const totalPagesOffset = isOffsetMode ? Math.ceil(total / limit) : 0;
  const startItem = isOffsetMode ? offset + 1 : (currentPage! - 1) * (limit || 0) + 1;
  const endItem = isOffsetMode
    ? Math.min(offset + limit, total)
    : Math.min(currentPage! * (limit || 0), total || 0);

  // Use appropriate values based on mode
  const page = isOffsetMode ? currentPageOffset : currentPage!;
  const pages = isOffsetMode ? totalPagesOffset : totalPages!;

  const handlePrevious = () => {
    if (isOffsetMode && offset > 0) {
      onOffsetChange(Math.max(0, offset - limit));
    } else if (isPageMode && currentPage! > 1) {
      onPageChange(currentPage! - 1);
    }
  };

  const handleNext = () => {
    if (isOffsetMode && offset + limit < total) {
      onOffsetChange(offset + limit);
    } else if (isPageMode && currentPage! < totalPages!) {
      onPageChange(currentPage! + 1);
    }
  };

  const handlePageClick = (pageNum: number) => {
    if (isOffsetMode) {
      onOffsetChange((pageNum - 1) * limit);
    } else if (isPageMode) {
      onPageChange(pageNum);
    }
  };

  const isPreviousDisabled = page === 1;
  const isNextDisabled = page === pages;

  // Calculate visible page numbers
  const getVisiblePages = (): number[] => {
    if (pages <= maxVisiblePages) {
      return Array.from({ length: pages }, (_, i) => i + 1);
    }

    const half = Math.floor(maxVisiblePages / 2);
    let start = Math.max(1, page - half);
    let end = Math.min(pages, start + maxVisiblePages - 1);

    if (end - start + 1 < maxVisiblePages) {
      start = Math.max(1, end - maxVisiblePages + 1);
    }

    return Array.from({ length: end - start + 1 }, (_, i) => start + i);
  };

  const visiblePages = showPageNumbers ? getVisiblePages() : [];
  const showLeftEllipsis = showPageNumbers && visiblePages[0] > 1;
  const showRightEllipsis = showPageNumbers && visiblePages[visiblePages.length - 1] < pages;

  // Single page - no pagination needed
  if (pages <= 1) {
    if (showItemCount && total !== undefined && total > 0) {
      return (
        <div className={cn('flex items-center justify-center', className)}>
          <p className="text-sm text-slate-500 dark:text-slate-400">
            Showing {total} {itemLabel}
          </p>
        </div>
      );
    }
    return null;
  }

  return (
    <div className={cn('flex items-center justify-between gap-4', className)}>
      {/* Item count */}
      {showItemCount && total !== undefined && (
        <p className="text-sm text-slate-500 dark:text-slate-400">
          Showing {startItem}-{endItem} of {total} {itemLabel}
        </p>
      )}

      {/* Navigation */}
      <div className={cn('flex items-center gap-2', !showItemCount && 'mx-auto')}>
        {/* Previous button */}
        <Button
          onClick={handlePrevious}
          disabled={isPreviousDisabled}
          variant="outline"
          size={size}
          icon={<ChevronLeft size={size === 'sm' ? 14 : size === 'md' ? 16 : 18} />}
          aria-label="Previous page"
        >
          Previous
        </Button>

        {/* Page numbers */}
        {showPageNumbers && (
          <div className="flex items-center gap-1">
            {/* First page */}
            {showLeftEllipsis && (
              <>
                <Button
                  onClick={() => handlePageClick(1)}
                  variant={page === 1 ? 'primary' : 'outline'}
                  size={size}
                  aria-label="Go to page 1"
                  aria-current={page === 1 ? 'page' : undefined}
                >
                  1
                </Button>
                <span className="text-slate-400 px-1">...</span>
              </>
            )}

            {/* Visible pages */}
            {visiblePages.map((pageNum) => (
              <Button
                key={pageNum}
                onClick={() => handlePageClick(pageNum)}
                variant={page === pageNum ? 'primary' : 'outline'}
                size={size}
                aria-label={`Go to page ${pageNum}`}
                aria-current={page === pageNum ? 'page' : undefined}
              >
                {pageNum}
              </Button>
            ))}

            {/* Last page */}
            {showRightEllipsis && (
              <>
                <span className="text-slate-400 px-1">...</span>
                <Button
                  onClick={() => handlePageClick(pages)}
                  variant={page === pages ? 'primary' : 'outline'}
                  size={size}
                  aria-label={`Go to page ${pages}`}
                  aria-current={page === pages ? 'page' : undefined}
                >
                  {pages}
                </Button>
              </>
            )}
          </div>
        )}

        {/* Next button */}
        <Button
          onClick={handleNext}
          disabled={isNextDisabled}
          variant="outline"
          size={size}
          icon={<ChevronRight size={size === 'sm' ? 14 : size === 'md' ? 16 : 18} />}
          iconPosition="right"
          aria-label="Next page"
        >
          Next
        </Button>
      </div>
    </div>
  );
};

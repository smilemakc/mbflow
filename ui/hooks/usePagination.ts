import { useState, useEffect, useMemo } from 'react';

export interface UsePaginationResult<T> {
  currentItems: T[];
  currentPage: number;
  totalPages: number;
  goToPage: (page: number) => void;
  nextPage: () => void;
  prevPage: () => void;
  hasNextPage: boolean;
  hasPrevPage: boolean;
}

/**
 * Generic client-side pagination hook.
 * Handles pagination for in-memory arrays.
 *
 * @param items - Array of items to paginate
 * @param itemsPerPage - Number of items per page (default: 12)
 * @returns Pagination state and controls
 *
 * @example
 * const pagination = usePagination(workflows, 12);
 * return (
 *   <>
 *     {pagination.currentItems.map(item => ...)}
 *     <button onClick={pagination.nextPage} disabled={!pagination.hasNextPage}>Next</button>
 *   </>
 * );
 */
export function usePagination<T>(
  items: T[],
  itemsPerPage: number = 12
): UsePaginationResult<T> {
  const [currentPage, setCurrentPage] = useState(1);

  const totalPages = Math.ceil(items.length / itemsPerPage);

  useEffect(() => {
    if (currentPage > totalPages && totalPages > 0) {
      setCurrentPage(totalPages);
    }
  }, [totalPages, currentPage]);

  const currentItems = useMemo(() => {
    const startIndex = (currentPage - 1) * itemsPerPage;
    const endIndex = startIndex + itemsPerPage;
    return items.slice(startIndex, endIndex);
  }, [items, currentPage, itemsPerPage]);

  const goToPage = (page: number) => {
    const validPage = Math.max(1, Math.min(page, totalPages));
    setCurrentPage(validPage);
  };

  const nextPage = () => {
    if (currentPage < totalPages) {
      setCurrentPage(currentPage + 1);
    }
  };

  const prevPage = () => {
    if (currentPage > 1) {
      setCurrentPage(currentPage - 1);
    }
  };

  return {
    currentItems,
    currentPage,
    totalPages,
    goToPage,
    nextPage,
    prevPage,
    hasNextPage: currentPage < totalPages,
    hasPrevPage: currentPage > 1,
  };
}

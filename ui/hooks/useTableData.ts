import { useState, useEffect, useCallback, useRef } from 'react';
import { getErrorMessage } from '@/lib/api';

export interface UseTableDataOptions<T, F> {
  fetchFn: (params: { limit: number; offset: number; filters?: F }) => Promise<{ items: T[]; total: number }>;
  initialLimit?: number;
  initialFilters?: F;
}

export interface UseTableDataResult<T, F> {
  items: T[];
  loading: boolean;
  error: string | null;
  total: number;
  limit: number;
  offset: number;
  filters: F;
  setFilters: (filters: F) => void;
  setOffset: (offset: number) => void;
  refresh: () => void;
}

/**
 * Hook for table data fetching with offset-based pagination.
 * Handles loading, error states, and pagination for server-side data.
 *
 * @param options - Configuration options
 * @returns Table data state and controls
 *
 * @example
 * const table = useTableData({
 *   fetchFn: async ({ limit, offset, filters }) => {
 *     const response = await api.list({ limit, offset, ...filters });
 *     return { items: response.data, total: response.total };
 *   },
 *   initialLimit: 20,
 *   initialFilters: { status: 'active' },
 * });
 *
 * // Change filters (resets to first page)
 * table.setFilters({ status: 'inactive' });
 *
 * // Navigate pages
 * table.setOffset(table.offset + table.limit);
 */
export function useTableData<T, F = Record<string, any>>(
  options: UseTableDataOptions<T, F>
): UseTableDataResult<T, F> {
  const { fetchFn, initialLimit = 20, initialFilters } = options;

  const [items, setItems] = useState<T[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [total, setTotal] = useState(0);
  const [limit] = useState(initialLimit);
  const [offset, setOffset] = useState(0);
  const [filters, setFiltersState] = useState<F>(initialFilters || ({} as F));
  const [refreshTrigger, setRefreshTrigger] = useState(0);

  // Use ref for fetchFn to avoid infinite loops when inline functions are passed
  const fetchFnRef = useRef(fetchFn);
  fetchFnRef.current = fetchFn;

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetchFnRef.current({ limit, offset, filters });
      setItems(response.items);
      setTotal(response.total);
    } catch (error: unknown) {
      setError(getErrorMessage(error));
      setItems([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [limit, offset, filters]);

  useEffect(() => {
    fetchData();
  }, [fetchData, refreshTrigger]);

  const setFilters = (newFilters: F) => {
    setFiltersState(newFilters);
    setOffset(0);
  };

  const refresh = useCallback(() => {
    setRefreshTrigger((prev) => prev + 1);
  }, []);

  return {
    items,
    loading,
    error,
    total,
    limit,
    offset,
    filters,
    setFilters,
    setOffset,
    refresh,
  };
}

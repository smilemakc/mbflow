import { describe, expect, it, vi } from 'vitest';
import { act, renderHook, waitFor } from '@testing-library/react';
import { usePagination } from './usePagination';
import { useTableData } from './useTableData';
import { apiError } from '@/test/apiMock';

describe('usePagination', () => {
  const items = Array.from({ length: 25 }, (_, i) => i);

  it('pages through items and clamps navigation', () => {
    const { result } = renderHook(() => usePagination(items, 10));
    expect(result.current).toMatchObject({ currentPage: 1, totalPages: 3, hasPrevPage: false, hasNextPage: true });
    expect(result.current.currentItems).toEqual(items.slice(0, 10));

    act(() => result.current.prevPage());
    expect(result.current.currentPage).toBe(1);
    act(() => result.current.nextPage());
    act(() => result.current.nextPage());
    act(() => result.current.nextPage());
    expect(result.current.currentPage).toBe(3);
    expect(result.current.currentItems).toEqual([20, 21, 22, 23, 24]);
    act(() => result.current.prevPage());
    expect(result.current.currentPage).toBe(2);
    act(() => result.current.goToPage(99));
    expect(result.current.currentPage).toBe(3);
    act(() => result.current.goToPage(-1));
    expect(result.current.currentPage).toBe(1);
  });

  it('moves back when the list shrinks', () => {
    const { result, rerender } = renderHook(({ list }) => usePagination(list), { initialProps: { list: items } });
    act(() => result.current.goToPage(3));
    rerender({ list: items.slice(0, 5) });
    expect(result.current.currentPage).toBe(1);
  });
});

describe('useTableData', () => {
  it('fetches, filters, pages and refreshes', async () => {
    const fetchFn = vi.fn(async ({ offset, filters }: { limit: number; offset: number; filters?: { q?: string } }) => ({
      items: [`${filters?.q ?? 'all'}@${offset}`],
      total: 40,
    }));
    const { result } = renderHook(() => useTableData({ fetchFn, initialLimit: 10 }));
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current).toMatchObject({ items: ['all@0'], total: 40, limit: 10, offset: 0, error: null });

    act(() => result.current.setOffset(10));
    await waitFor(() => expect(result.current.items).toEqual(['all@10']));

    act(() => result.current.setFilters({ q: 'x' }));
    await waitFor(() => expect(result.current.items).toEqual(['x@0']));

    const calls = fetchFn.mock.calls.length;
    act(() => result.current.refresh());
    await waitFor(() => expect(fetchFn.mock.calls.length).toBe(calls + 1));
  });

  it('exposes errors and clears items', async () => {
    const fetchFn = vi.fn().mockRejectedValue(apiError(500, { message: 'broken' }));
    const { result } = renderHook(() => useTableData({ fetchFn }));
    await waitFor(() => expect(result.current.error).toBe('broken'));
    expect(result.current).toMatchObject({ items: [], total: 0, loading: false, limit: 20 });
  });
});

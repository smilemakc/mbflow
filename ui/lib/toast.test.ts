import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { toast, useToastStore } from './toast';

describe('toast store', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    useToastStore.getState().clearAll();
  });
  afterEach(() => vi.useRealTimers());

  it('adds typed toasts with default durations', () => {
    toast.success('ok', 'done');
    toast.error('bad');
    toast.warning('careful');
    toast.info('fyi');
    const toasts = useToastStore.getState().toasts;
    expect(toasts.map((t) => [t.type, t.title, t.duration])).toEqual([
      ['success', 'ok', 5000],
      ['error', 'bad', 7000],
      ['warning', 'careful', 5000],
      ['info', 'fyi', 5000],
    ]);
    expect(toasts[0].message).toBe('done');
    expect(new Set(toasts.map((t) => t.id)).size).toBe(4);
  });

  it('auto-dismisses after the duration', () => {
    toast.success('bye');
    vi.advanceTimersByTime(4999);
    expect(useToastStore.getState().toasts).toHaveLength(1);
    vi.advanceTimersByTime(1);
    expect(useToastStore.getState().toasts).toHaveLength(0);
  });

  it('keeps toasts with zero duration until dismissed', () => {
    const id = useToastStore.getState().addToast({ type: 'info', title: 'sticky', duration: 0 });
    vi.advanceTimersByTime(60_000);
    expect(useToastStore.getState().toasts).toHaveLength(1);
    toast.dismiss(id);
    expect(useToastStore.getState().toasts).toHaveLength(0);
  });

  it('clears all toasts', () => {
    toast.info('a');
    toast.info('b');
    toast.clearAll();
    expect(useToastStore.getState().toasts).toEqual([]);
  });
});

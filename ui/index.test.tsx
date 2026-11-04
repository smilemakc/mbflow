import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const renderSpy = vi.hoisted(() => vi.fn());
const createRootSpy = vi.hoisted(() => vi.fn(() => ({ render: renderSpy })));

vi.mock('react-dom/client', () => ({ createRoot: createRootSpy }));
vi.mock('./App', () => ({ default: () => null }));

describe('index entry point', () => {
  const originalOnError = window.onerror;

  beforeEach(() => {
    vi.resetModules();
    renderSpy.mockClear();
    createRootSpy.mockClear();
    document.body.innerHTML = '';
  });

  afterEach(() => {
    window.onerror = originalOnError;
  });

  it('mounts the app into #root in strict mode', async () => {
    const root = document.createElement('div');
    root.id = 'root';
    document.body.appendChild(root);

    await import('./index');

    expect(createRootSpy).toHaveBeenCalledWith(root);
    expect(renderSpy).toHaveBeenCalledTimes(1);
    const element = renderSpy.mock.calls[0][0];
    expect(element.type).toBe((await import('react')).StrictMode);
  });

  it('fails loudly without a root element', async () => {
    await expect(import('./index')).rejects.toThrow('Could not find root element to mount to');
    expect(createRootSpy).not.toHaveBeenCalled();
  });

  it('swallows benign ResizeObserver errors and forwards the rest', async () => {
    const previous = vi.fn(() => 'handled');
    window.onerror = previous as never;
    document.body.innerHTML = '<div id="root"></div>';
    await import('./index');

    expect(window.onerror!('ResizeObserver loop limit exceeded')).toBe(true);
    expect(window.onerror!('ResizeObserver loop completed with undelivered notifications')).toBe(true);
    expect(previous).not.toHaveBeenCalled();

    expect(window.onerror!('Real failure', 'app.js', 1, 2)).toBe('handled');
    expect(previous).toHaveBeenCalledWith('Real failure', 'app.js', 1, 2, undefined);

    // Error events from ResizeObserver are stopped before other listeners see them.
    const later = vi.fn();
    window.addEventListener('error', later);
    window.dispatchEvent(new ErrorEvent('error', { message: 'ResizeObserver loop limit exceeded' }));
    expect(later).not.toHaveBeenCalled();
    window.dispatchEvent(new ErrorEvent('error', { message: 'other' }));
    expect(later).toHaveBeenCalledTimes(1);
    window.removeEventListener('error', later);
  });

  it('returns false for unhandled errors when no previous handler exists', async () => {
    window.onerror = null;
    document.body.innerHTML = '<div id="root"></div>';
    await import('./index');
    expect(window.onerror!('Real failure')).toBe(false);
  });
});

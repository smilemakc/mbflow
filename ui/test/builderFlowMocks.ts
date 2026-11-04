/**
 * Test helper for the builder area: browser APIs reactflow needs that jsdom lacks,
 * plus store reset utilities. Import it before rendering reactflow components.
 */
import { vi } from 'vitest';
import { useDagStore } from '@/store/dagStore';
import { useUIStore } from '@/store/uiStore';
import { useToastStore } from '@/lib/toast';
import { NodeStatus, NodeType, type AppEdge, type AppNode } from '@/types';

class ResizeObserverMock {
  private readonly callback: ResizeObserverCallback;
  constructor(callback: ResizeObserverCallback) {
    this.callback = callback;
  }
  observe(target: Element) {
    this.callback([{ target, contentRect: { width: 1000, height: 800 } } as unknown as ResizeObserverEntry], this as unknown as ResizeObserver);
  }
  unobserve() {}
  disconnect() {}
}

class DOMMatrixReadOnlyMock {
  m22: number;
  constructor(transform?: string) {
    const scale = transform?.match(/scale\(([\d.]+)\)/)?.[1];
    this.m22 = scale !== undefined ? +scale : 1;
  }
}

const g = globalThis as Record<string, unknown>;
if (!g.__builderFlowMocks) {
  g.__builderFlowMocks = true;
  g.ResizeObserver = ResizeObserverMock;
  g.DOMMatrixReadOnly = DOMMatrixReadOnlyMock;
  Object.defineProperties(HTMLElement.prototype, {
    offsetHeight: { configurable: true, get() { return parseFloat((this as HTMLElement).style.height) || 800; } },
    offsetWidth: { configurable: true, get() { return parseFloat((this as HTMLElement).style.width) || 1000; } },
  });
  Object.defineProperty(HTMLElement.prototype, 'getBoundingClientRect', {
    configurable: true,
    value() {
      return { x: 0, y: 0, top: 0, left: 0, right: 1000, bottom: 800, width: 1000, height: 800, toJSON() {} };
    },
  });
  if (!document.elementFromPoint) {
    // Used by reactflow while dragging a connection; nothing is under the pointer in jsdom.
    document.elementFromPoint = () => null;
  }
  if (!document.elementsFromPoint) {
    document.elementsFromPoint = () => [];
  }
  (SVGElement.prototype as unknown as { getBBox: () => DOMRect }).getBBox = () =>
    ({ x: 0, y: 0, width: 0, height: 0 }) as DOMRect;
}

const initialDag = useDagStore.getState();
const initialUi = useUIStore.getState();

/** Resets the dag/ui/toast stores, forces English and silences store console noise. */
export const resetBuilderStores = () => {
  useDagStore.setState(initialDag, true);
  useUIStore.setState({ ...initialUi, language: 'en' }, true);
  useToastStore.getState().clearAll();
  vi.spyOn(console, 'log').mockImplementation(() => {});
  vi.spyOn(console, 'error').mockImplementation(() => {});
};

export const makeNode = (id: string, type: NodeType = NodeType.HTTP, data: Record<string, unknown> = {}): AppNode =>
  ({
    id,
    type: 'custom',
    position: { x: 10, y: 20 },
    data: { label: `Label ${id}`, type, status: NodeStatus.IDLE, config: {}, ...data },
  }) as AppNode;

export const makeEdge = (id: string, source: string, target: string, extra: Partial<AppEdge> = {}): AppEdge =>
  ({ id, source, target, ...extra }) as AppEdge;

/** Installs a clipboard stub and returns its writeText mock. */
export const stubClipboard = () => {
  const writeText = vi.fn().mockResolvedValue(undefined);
  Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } });
  return writeText;
};

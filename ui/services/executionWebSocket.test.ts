import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { executionWS } from './executionWebSocket';
import { TOKEN_KEY } from '@/lib/authStorage';

class FakeWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;
  static instances: FakeWebSocket[] = [];

  readyState = FakeWebSocket.CONNECTING;
  sent: string[] = [];
  onopen: (() => void) | null = null;
  onmessage: ((e: { data: string }) => void) | null = null;
  onerror: ((e: unknown) => void) | null = null;
  onclose: ((e: { code: number }) => void) | null = null;

  constructor(public url: string) {
    FakeWebSocket.instances.push(this);
  }
  send(data: string) {
    this.sent.push(data);
  }
  close = vi.fn(() => {
    this.readyState = FakeWebSocket.CLOSED;
  });
  open() {
    this.readyState = FakeWebSocket.OPEN;
    this.onopen?.();
  }
  message(data: unknown) {
    this.onmessage?.({ data: typeof data === 'string' ? data : JSON.stringify(data) });
  }
}

describe('executionWS', () => {
  const originalWS = globalThis.WebSocket;

  beforeEach(() => {
    vi.useFakeTimers();
    FakeWebSocket.instances = [];
    globalThis.WebSocket = FakeWebSocket as unknown as typeof WebSocket;
    vi.spyOn(console, 'log').mockImplementation(() => {});
    vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    executionWS.disconnectAll();
    globalThis.WebSocket = originalWS;
    vi.useRealTimers();
  });

  it('builds the URL from the page location with execution id and token', () => {
    localStorage.setItem(TOKEN_KEY, 'tok');
    executionWS.connect('e1', () => {});
    const url = new URL(FakeWebSocket.instances[0].url);
    expect(url.protocol).toBe('ws:');
    expect(url.host).toBe(window.location.host);
    expect(url.pathname).toBe('/ws/executions');
    expect(url.searchParams.get('execution_id')).toBe('e1');
    expect(url.searchParams.get('token')).toBe('tok');
  });

  it('omits the query when there is nothing to send', () => {
    executionWS.connect('', () => {});
    expect(FakeWebSocket.instances[0].url.endsWith('/ws/executions')).toBe(true);
  });

  it('delivers events to execution and global handlers, ignoring pongs and bad JSON', () => {
    const handler = vi.fn();
    const global = vi.fn();
    const removeGlobal = executionWS.addGlobalHandler(global);
    executionWS.connect('e1', handler);
    const ws = FakeWebSocket.instances[0];
    ws.open();
    expect(executionWS.isConnected('e1')).toBe(true);

    ws.message({ type: 'pong' });
    ws.message('not json');
    ws.message({ type: 'event', event: { event_type: 'node.started' } });

    expect(handler).toHaveBeenCalledTimes(1);
    expect(global).toHaveBeenCalledTimes(1);
    removeGlobal();
    ws.message({ type: 'event' });
    expect(global).toHaveBeenCalledTimes(1);
  });

  it('pings every 30 seconds while open', () => {
    executionWS.connect('e1', () => {});
    const ws = FakeWebSocket.instances[0];
    ws.open();
    vi.advanceTimersByTime(30_000);
    expect(ws.sent).toEqual([JSON.stringify({ type: 'ping' })]);
    ws.readyState = FakeWebSocket.CLOSING;
    vi.advanceTimersByTime(30_000);
    expect(ws.sent).toHaveLength(1);
  });

  it('reuses an open connection and disconnects when the last handler leaves', () => {
    const a = vi.fn();
    const b = vi.fn();
    const offA = executionWS.connect('e1', a);
    FakeWebSocket.instances[0].open();
    const offB = executionWS.connect('e1', b);
    expect(FakeWebSocket.instances).toHaveLength(1);

    // Re-connecting without a handler keeps the existing ones.
    executionWS.connect('e1')();

    FakeWebSocket.instances[0].message({ type: 'event' });
    expect(a).toHaveBeenCalledOnce();
    expect(b).toHaveBeenCalledOnce();

    offA();
    expect(executionWS.isConnected('e1')).toBe(true);
    offB();
    expect(executionWS.isConnected('e1')).toBe(false);
    expect(FakeWebSocket.instances[0].close).toHaveBeenCalled();
  });

  it('cleans up on server close and on errors', () => {
    executionWS.connect('e1', () => {});
    const ws = FakeWebSocket.instances[0];
    ws.open();
    ws.onerror?.(new Event('error'));
    ws.onclose?.({ code: 1006 });
    expect(executionWS.isConnected('e1')).toBe(false);
    // Disconnecting an unknown or already-closed execution is a no-op.
    executionWS.disconnect('e1');
    executionWS.disconnect('missing');
    // A message arriving after cleanup only reaches global handlers.
    ws.message({ type: 'event' });
  });

  it('does not call close on a socket that never opened', () => {
    executionWS.connect('e1', () => {});
    executionWS.disconnect('e1');
    expect(FakeWebSocket.instances[0].close).not.toHaveBeenCalled();
  });

  it('uses wss on https pages', () => {
    const original = window.location;
    Object.defineProperty(window, 'location', { configurable: true, value: { protocol: 'https:', host: 'secure.example' } });
    try {
      executionWS.connect('e2', () => {});
      expect(FakeWebSocket.instances[0].url.startsWith('wss://secure.example/ws/executions')).toBe(true);
    } finally {
      Object.defineProperty(window, 'location', { configurable: true, value: original });
    }
  });

  it('uses an absolute VITE_WS_URL as-is', () => {
    vi.stubEnv('VITE_WS_URL', 'wss://ws.example/ws');
    try {
      executionWS.connect('e3', () => {});
      expect(FakeWebSocket.instances[0].url).toBe('wss://ws.example/ws/executions?execution_id=e3');
    } finally {
      vi.unstubAllEnvs();
    }
  });
});

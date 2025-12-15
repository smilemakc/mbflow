# React Hooks

This directory contains custom React hooks for the MBFlow application.

## Available Hooks

#### WebSocket URL

The hook connects to: `/ws/executions?execution_id={id}`

The base WebSocket URL can be configured via `VITE_WS_URL` environment variable.

### useWebSocket

A generic React hook for WebSocket connections with automatic reconnection.

**File:** `hooks/useWebSocket.ts`

#### Features

- Automatic reconnection with exponential backoff
- Ping/pong keepalive mechanism
- Type-safe message handling
- Connection state management

#### Usage

```tsx
import { useWebSocket } from './hooks/useWebSocket';

function MyComponent() {
  const {
    data,
    error,
    isConnected,
    reconnectAttempts,
    send,
    connect,
    disconnect,
    reconnect,
  } = useWebSocket<MyMessageType>('ws://localhost:8000/ws', {
    autoConnect: true,
    maxReconnectAttempts: 5,
    pingInterval: 30000,
  });

  return (
    <div>
      {isConnected && <p>Connected!</p>}
      {data && <pre>{JSON.stringify(data, null, 2)}</pre>}
    </div>
  );
}
```

### useAutoSave

A React hook for auto-saving DAG changes after a period of inactivity.

**File:** `hooks/useAutoSave.ts`

#### Usage

```tsx
import { useAutoSave } from './hooks/useAutoSave';

function DAGEditor() {
  const { isSaving } = useAutoSave();

  return (
    <div>
      {isSaving && <span>Saving...</span>}
      {/* DAG editor content */}
    </div>
  );
}
```


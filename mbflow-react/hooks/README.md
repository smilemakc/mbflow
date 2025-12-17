# React Hooks

This directory contains custom React hooks for the MBFlow application.

## Available Hooks

### Core Refactoring Hooks

#### useNodeConfig

Generic hook for node config state management. Replaces the pattern of local state + useEffect + onChange propagation.

**File:** `hooks/useNodeConfig.ts`

```tsx
import { useNodeConfig } from '@/hooks';

interface Props {
  config: LLMNodeConfig;
  onChange: (config: LLMNodeConfig) => void;
}

export const LLMNodeConfigComponent: React.FC<Props> = ({ config, onChange }) => {
  const [localConfig, updateConfig] = useNodeConfig(config, onChange);

  return (
    <input
      value={localConfig.model}
      onChange={(e) => updateConfig({ model: e.target.value })}
    />
  );
};
```

#### useSafeConfig

Hook for providing default values to config. Ensures config always has all required fields with defaults.

**File:** `hooks/useSafeConfig.ts`

```tsx
import { useSafeConfig } from '@/hooks';

export const GoogleSheetsNodeConfig: React.FC<Props> = ({ config, onChange }) => {
  const safeConfig = useSafeConfig(config, {
    operation: 'read',
    spreadsheet_id: '',
    sheet_name: '',
  });

  return (
    <select value={safeConfig.operation}>
      <option value="read">Read</option>
      <option value="write">Write</option>
    </select>
  );
};
```

#### usePagination

Generic client-side pagination hook. Handles pagination for in-memory arrays.

**File:** `hooks/usePagination.ts`

```tsx
import { usePagination } from '@/hooks';

export const WorkflowsPage: React.FC = () => {
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const pagination = usePagination(workflows, 12);

  return (
    <>
      {pagination.currentItems.map((workflow) => (
        <WorkflowCard key={workflow.id} workflow={workflow} />
      ))}
      <Button onClick={pagination.prevPage} disabled={!pagination.hasPrevPage}>
        Previous
      </Button>
      <Button onClick={pagination.nextPage} disabled={!pagination.hasNextPage}>
        Next
      </Button>
    </>
  );
};
```

#### useTableData

Hook for table data fetching with offset-based pagination. Handles loading, error states, and pagination for server-side data.

**File:** `hooks/useTableData.ts`

```tsx
import { useTableData } from '@/hooks';

export const RentalKeyAdminList: React.FC = () => {
  const table = useTableData({
    fetchFn: async ({ limit, offset, filters }) => {
      const response = await api.list({ limit, offset, ...filters });
      return { items: response.data, total: response.total };
    },
    initialLimit: 20,
  });

  return (
    <>
      {table.loading ? <Loader /> : (
        <table>
          {table.items.map((item) => <tr key={item.id}>...</tr>)}
        </table>
      )}
    </>
  );
};
```

#### useArrayToString

Helper for converting array to string and back. Used for stop_sequences, tags, and other array-like text inputs.

**File:** `hooks/useArrayToString.ts`

```tsx
import { useArrayToString } from '@/hooks';

const [stopSequencesText, parseStopSequences] = useArrayToString(
  config.stop_sequences,
  '\n'
);

const handleChange = (text: string) => {
  const sequences = parseStopSequences(text);
  updateConfig({ stop_sequences: sequences });
};
```

---

### WebSocket & Real-time Hooks

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


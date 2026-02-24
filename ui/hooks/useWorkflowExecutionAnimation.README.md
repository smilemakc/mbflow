# useWorkflowExecutionAnimation Hook

React hook для анимации выполнения workflow на canvas. Портирован из Vue composable
`/mbflow-ui/src/composables/useWorkflowExecutionAnimation.ts`.

## Описание

Hook предоставляет функции для визуальной анимации состояния выполнения workflow:

- Анимация узлов (running, completed, failed)
- Анимация рёбер (active, completed)
- Сохранение input/output/error данных в узлах
- Обработка событий выполнения из WebSocket

## API

### Возвращаемые функции

#### `animateNode(nodeId: string, state: "running" | "completed" | "failed")`

Анимирует узел в соответствии с его состоянием выполнения.

**Параметры:**

- `nodeId` - ID узла для анимации
- `state` - Состояние выполнения:
    - `"running"` - узел выполняется (синяя пульсация)
    - `"completed"` - узел завершён успешно (зелёный)
    - `"failed"` - узел завершился с ошибкой (красный)

**Обновляемые поля в `node.data`:**

- `executionState` - текущее состояние
- `animated` - флаг анимации (true для running, false для completed/failed)

#### `animateEdge(sourceId: string, targetId: string, state: "active" | "completed")`

Анимирует ребро между двумя узлами.

**Параметры:**

- `sourceId` - ID исходного узла
- `targetId` - ID целевого узла
- `state` - Состояние ребра:
    - `"active"` - ребро активно (синее, анимированное)
    - `"completed"` - ребро пройдено (зелёное)

**Обновляемые поля в `edge`:**

- `animated` - флаг анимации
- `style.stroke` - цвет линии (#3b82f6 для active, #10b981 для completed)
- `style.strokeWidth` - толщина линии (2px)

#### `resetAnimations()`

Сбрасывает все анимации на canvas.

**Действия:**

- Очищает `executionState` и `animated` у всех узлов
- Убирает анимацию и стили у всех рёбер

#### `handleExecutionEvent(event: ExecutionEvent)`

Обрабатывает событие выполнения и вызывает соответствующие анимации.

**Параметры:**

- `event` - Событие выполнения из WebSocket

**Обрабатываемые типы событий:**

| event_type            | Действие                                                                                                    |
|-----------------------|-------------------------------------------------------------------------------------------------------------|
| `execution.started`   | Сброс всех анимаций                                                                                         |
| `node.started`        | Анимация узла как running, сохранение input                                                                 |
| `node.completed`      | Анимация узла как completed, сохранение output, анимация входящих рёбер как completed, исходящих как active |
| `node.failed`         | Анимация узла как failed, сохранение error                                                                  |
| `execution.completed` | Остановка всех анимаций                                                                                     |
| `execution.failed`    | Остановка всех анимаций                                                                                     |

**Сохраняемые данные в `node.data`:**

- `executionInput` - входные данные узла (из node.started)
- `executionOutput` - выходные данные узла (из node.completed)
- `executionError` - ошибка выполнения (из node.failed)

## Использование

### Базовое использование с WebSocket

```tsx
import {useWorkflowExecutionAnimation} from './hooks/useWorkflowExecutionAnimation';
import {useExecutionObserver} from './hooks/useExecutionObserver';

function WorkflowCanvas() {
    const {handleExecutionEvent} = useWorkflowExecutionAnimation();

    // Подключение к WebSocket и автоматическая обработка событий
    const {isConnected} = useExecutionObserver({
        executionId: 'exec-123',
        autoConnect: true,
        onEvent: handleExecutionEvent, // Передаём все события в обработчик анимации
    });

    return (
        <div>
            {isConnected ? 'Connected' : 'Disconnected'}
            {/* React Flow canvas */}
        </div>
    );
}
```

### Ручное управление анимацией

```tsx
import {useWorkflowExecutionAnimation} from './hooks/useWorkflowExecutionAnimation';

function ManualControl() {
    const {animateNode, animateEdge, resetAnimations} = useWorkflowExecutionAnimation();

    const startWorkflow = () => {
        resetAnimations();

        // Шаг 1: Запуск первого узла
        animateNode('node-1', 'running');

        setTimeout(() => {
            // Шаг 2: Завершение первого узла
            animateNode('node-1', 'completed');
            animateEdge('node-1', 'node-2', 'active');

            // Шаг 3: Запуск второго узла
            animateNode('node-2', 'running');
        }, 2000);
    };

    return <button onClick={startWorkflow}>Start</button>;
}
```

### Интеграция с React Flow

```tsx
import ReactFlow, {Background, Controls} from 'reactflow';
import {useDagStore} from './store/dagStore';
import {useWorkflowExecutionAnimation} from './hooks/useWorkflowExecutionAnimation';
import {useExecutionObserver} from './hooks/useExecutionObserver';

function AnimatedFlow() {
    const nodes = useDagStore((state) => state.nodes);
    const edges = useDagStore((state) => state.edges);
    const executionId = useDagStore((state) => state.executionId);

    const {handleExecutionEvent} = useWorkflowExecutionAnimation();

    useExecutionObserver({
        executionId: executionId || undefined,
        autoConnect: !!executionId,
        onEvent: handleExecutionEvent,
    });

    return (
        <ReactFlow
            nodes={nodes}
            edges={edges}
            // ... other props
        >
            <Background/>
            <Controls/>
        </ReactFlow>
    );
}
```

### Доступ к данным выполнения в Custom Node

```tsx
import {memo} from 'react';
import {Handle, Position, NodeProps} from 'reactflow';
import type {NodeData} from '@/types';

function CustomNode({data}: NodeProps<NodeData>) {
    return (
        <div className={`
      custom-node
      ${data.executionState || ''}
      ${data.animated ? 'animated' : ''}
    `}>
            <Handle type="target" position={Position.Top}/>

            <div className="node-label">{data.label}</div>

            {/* Отображение состояния */}
            {data.executionState && (
                <div className="execution-state">{data.executionState}</div>
            )}

            {/* Отображение ошибки */}
            {data.executionError && (
                <div className="error-message">{data.executionError}</div>
            )}

            {/* Отображение output */}
            {data.executionOutput && (
                <div className="output">
                    <pre>{JSON.stringify(data.executionOutput, null, 2)}</pre>
                </div>
            )}

            <Handle type="source" position={Position.Bottom}/>
        </div>
    );
}

export default memo(CustomNode);
```

## CSS для анимаций

Рекомендуемые стили для визуализации состояний:

```css
.node {
    border: 2px solid #ccc;
    border-radius: 8px;
    padding: 12px;
    transition: all 0.3s ease;
}

/* Состояние: running */
.node.running {
    border-color: #3b82f6;
    box-shadow: 0 0 12px rgba(59, 130, 246, 0.5);
}

.node.running.animated {
    animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
    0%, 100% {
        transform: scale(1);
        opacity: 1;
    }
    50% {
        transform: scale(1.05);
        opacity: 0.8;
    }
}

/* Состояние: completed */
.node.completed {
    border-color: #10b981;
    background-color: rgba(16, 185, 129, 0.1);
}

/* Состояние: failed */
.node.failed {
    border-color: #ef4444;
    background-color: rgba(239, 68, 68, 0.1);
}

/* Анимация рёбер */
.react-flow__edge-path {
    transition: stroke 0.3s ease;
}

.react-flow__edge.animated .react-flow__edge-path {
    stroke-dasharray: 5;
    animation: dash 0.5s linear infinite;
}

@keyframes dash {
    to {
        stroke-dashoffset: -10;
    }
}
```

## Типы данных

### NodeData (расширенный)

```typescript
interface NodeData extends Record<string, unknown> {
    // ... existing fields
    executionState?: 'running' | 'completed' | 'failed';
    animated?: boolean;
    executionInput?: Record<string, any>;
    executionOutput?: Record<string, any>;
    executionError?: string;
}
```

### ExecutionEvent

```typescript
interface ExecutionEvent {
    type: "event" | "control";
    event?: {
        event_type: string;
        execution_id: string;
        workflow_id: string;
        timestamp: string;
        status: string;
        node_id?: string;
        node_name?: string;
        node_type?: string;
        wave_index?: number;
        node_count?: number;
        duration_ms?: number;
        error?: string;
        input?: Record<string, any>;
        output?: Record<string, any>;
    };
    control?: Record<string, any>;
    timestamp: string;
}
```

## Зависимости

- `useDagStore` - Zustand store для управления nodes/edges
- `ExecutionEvent` - тип события из '@/types/execution'
- React Flow (опционально, для визуализации)

## Отличия от Vue версии

1. **Реактивность**: Используется Zustand вместо Vue refs
2. **Lifecycle**: `useCallback` вместо Vue computed/watch
3. **Store updates**: Прямые вызовы `updateNodeData` и `setState` вместо мутации refs
4. **Dependencies**: Явное указание зависимостей в useCallback

## См. также

- [useExecutionObserver](./useExecutionObserver.ts) - WebSocket observer для событий выполнения
- [useWebSocket](./useWebSocket.ts) - Базовый WebSocket hook
- [Example](./useWorkflowExecutionAnimation.example.tsx) - Примеры использования

# MBFlow UI

Modern Vue 3 + TypeScript frontend for MBFlow workflow orchestration system.

## Tech Stack

- **Vue 3.4+** with Composition API and `<script setup>`
- **TypeScript 5.3+** for type safety
- **Vite 5+** for blazing fast HMR
- **Vuetify 3.5+** for Material Design components
- **Vue Flow 1.33+** for workflow visualization (to be integrated)
- **Pinia 2.1+** for state management
- **Vue Router 4.3+** for routing
- **Axios 1.6+** for HTTP requests

## Project Structure

```
mbflow-ui/
├── src/
│   ├── api/                    # API clients and mock data
│   │   ├── client.ts           # Axios instance
│   │   ├── workflows.api.ts    # Workflows API
│   │   ├── executions.api.ts   # Executions API
│   │   └── mock-data.ts        # Mock data for development
│   │
│   ├── components/             # Vue components (to be added)
│   │   ├── workflow/           # Workflow canvas components
│   │   ├── nodes/              # Node components
│   │   └── edges/              # Edge components
│   │
│   ├── stores/                 # Pinia stores
│   │   ├── workflow.store.ts   # Workflow state
│   │   └── execution.store.ts  # Execution state
│   │
│   ├── types/                  # TypeScript types
│   │   ├── domain.types.ts     # Domain types (mirrors Go)
│   │   ├── workflow.types.ts   # Workflow types
│   │   ├── execution.types.ts  # Execution types
│   │   ├── variable.types.ts   # Variable types
│   │   └── collaboration.types.ts # Collaboration types
│   │
│   ├── views/                  # Page components
│   │   ├── WorkflowListView.vue
│   │   ├── WorkflowEditorView.vue
│   │   ├── ExecutionMonitorView.vue
│   │   └── ExecutionHistoryView.vue
│   │
│   ├── router/                 # Vue Router
│   ├── plugins/                # Vue plugins
│   ├── App.vue                 # Root component
│   └── main.ts                 # Entry point
│
├── index.html
├── vite.config.ts
├── tsconfig.json
└── package.json
```

## Getting Started

### Prerequisites

- Node.js 18+ and npm

### Installation

```bash
# Install dependencies
npm install
```

### Development

```bash
# Start dev server (with mock API)
npm run dev

# The app will be available at http://localhost:3434
```

The development server is configured to:

- Use mock API by default (`VITE_USE_MOCK_API=true`)
- Proxy `/api` requests to `http://localhost:8181` (when using real backend)
- Proxy WebSocket `/ws` to `ws://localhost:8181/ws`

### Building for Production

```bash
# Type check
npm run type-check

# Build for production
npm run build

# Preview production build
npm run preview
```

## Environment Variables

### Development (`.env.development`)

```env
VITE_API_BASE_URL=http://localhost:8181
VITE_WS_URL=ws://localhost:8181/ws
VITE_USE_MOCK_API=true
```

### Production (`.env.production`)

```env
VITE_API_BASE_URL=/api
VITE_WS_URL=ws://localhost:8181/ws
VITE_USE_MOCK_API=false
```

## Features

### Current (Phase 1)

✅ **Project Setup**

- Vue 3 + TypeScript + Vite
- Vuetify 3 UI framework
- Pinia state management
- Vue Router

✅ **Type System**

- Complete TypeScript types mirroring Go domain models
- Type-safe API clients
- Strongly typed stores

✅ **API Integration**

- Mock API for development
- Real API client with Axios
- Workflows CRUD operations
- Executions management

✅ **Basic Views**

- Workflow list with cards
- Workflow editor (placeholder)
- Execution monitor (placeholder)
- Execution history with data table

✅ **State Management**

- Workflow store with undo/redo
- Execution store with event sourcing
- Reactive state updates

### Planned (Future Phases)

🔄 **Phase 2: Workflow Editor**

- Vue Flow integration
- Node palette with drag-and-drop
- Visual workflow canvas
- Node property panels
- Edge configuration

🔄 **Phase 3: Variable System**

- Variable context tracking
- Autocomplete in expressions
- Variable explorer panel
- Scoped variable resolution

🔄 **Phase 4: Testing & Debugging**

- Mock executor for dry-run
- Node tester component
- Step-by-step debugger
- Execution timeline visualization

🔄 **Phase 5: Real-time Features**

- WebSocket integration
- Live execution monitoring
- Event stream visualization
- Real-time status updates

🔄 **Phase 6: Collaboration**

- Multi-user editing
- Cursor tracking
- Resource locking
- Conflict resolution

## Development Guidelines

### Code Style

- Use Composition API with `<script setup>`
- Prefer `const` over `let`
- Use TypeScript strict mode
- Follow Vue 3 best practices

### Component Structure

```vue
<template>
  <!-- Template -->
</template>

<script setup lang="ts">
// Imports
import { ref } from 'vue'

// Props & Emits
interface Props {
  // ...
}
const props = defineProps<Props>()
const emit = defineEmits<{
  // ...
}>()

// State
const state = ref()

// Computed
// Methods
// Lifecycle hooks
</script>

<style scoped>
/* Styles */
</style>
```

### API Usage

```typescript
// Use stores for state management
import { useWorkflowStore } from '@/stores/workflow.store'

const workflowStore = useWorkflowStore()

// Fetch data
await workflowStore.fetchWorkflows()

// Access reactive state
const { workflows, loading } = storeToRefs(workflowStore)
```

## Mock Data

The app includes comprehensive mock data for development:

- 2 sample workflows (simple transform, HTTP API)
- Sample executions with events
- Node type metadata
- Edge type metadata

Toggle mock mode via `VITE_USE_MOCK_API` environment variable.

## Next Steps

1. **Integrate Vue Flow** for visual workflow editing
2. **Implement node components** for all 18 node types
3. **Add property panels** for node configuration
4. **Build variable system** with autocomplete
5. **Create testing tools** (dry-run, debugger)
6. **Add WebSocket support** for real-time updates
7. **Implement collaboration features**

## Contributing

Follow the implementation plan in the main project documentation.

## License

MIT

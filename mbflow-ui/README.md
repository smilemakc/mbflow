# MBFlow UI

Modern Vue 3 frontend for MBFlow workflow orchestration engine.

## Tech Stack

- **Vue 3.5+** with Composition API and `<script setup>`
- **TypeScript 5.3+** for type safety
- **Vite 6.0+** for blazing fast dev server
- **TailwindCSS 3.4+** for styling
- **Radix Vue & Headless UI** for accessible UI components
- **Vue Flow** for DAG workflow visualization
- **Pinia** for state management
- **TanStack Query** for server state management
- **unplugin-vue-router** for file-based auto-routing
- **VeeValidate + Yup** for form validation

## Features

- ✅ **Responsive Layout** - Collapsible sidebar, mobile-friendly
- ✅ **File-based Routing** - Auto-generated routes from `/src/pages`
- ✅ **Type-safe API** - Axios client with TypeScript types
- ✅ **Auto-layout** - ELK and dagre algorithms for DAG positioning
- ✅ **Workflow Editor** - Visual DAG editor with Vue Flow
- ✅ **Workflow Management** - Create, read, update, delete workflows
- ✅ **Custom Node Components** - HTTP, LLM, Transform, Conditional, Merge nodes
- ✅ **Node Configuration** - Dynamic configuration panel for node settings
- ✅ **Drag-and-Drop** - Node palette with drag-and-drop support
- 🚧 **Real-time Updates** - WebSocket integration for execution monitoring
- 🚧 **Form Validation** - VeeValidate with Yup schemas

## Project Structure

```
src/
├── assets/
│   └── styles/           # TailwindCSS styles
├── components/
│   ├── ui/               # Reusable UI components (Button, Input, etc.)
│   ├── layout/           # Layout components (AppShell, Header, Sidebar)
│   └── workflow/         # Workflow editor components
├── composables/          # Vue composables
├── stores/               # Pinia stores
├── api/                  # API client and endpoints
├── types/                # TypeScript type definitions
├── pages/                # File-based routing (auto-routes)
│   ├── index.vue         # Dashboard (/)
│   ├── workflows/        # Workflows (/workflows)
│   ├── executions/       # Executions (/executions)
│   └── triggers/         # Triggers (/triggers)
├── utils/                # Utility functions
├── App.vue               # Root component
└── main.ts               # Application entry point
```

## Development

### Prerequisites

- Node.js 18+
- npm or pnpm

### Installation

```bash
npm install
```

### Development Server

Start the dev server on `http://localhost:3434`:

```bash
npm run dev
```

The dev server includes:
- Hot Module Replacement (HMR)
- API proxy to backend (`/api` → `http://localhost:8181`)
- WebSocket proxy (`/ws` → `ws://localhost:8181`)

### Build

Build for production:

```bash
npm run build
```

Preview production build:

```bash
npm run preview
```

### Type Checking

Run TypeScript type checking:

```bash
npm run type-check
```

### Linting & Formatting

```bash
npm run lint
npm run format
```

### Testing

```bash
# Unit tests
npm run test:unit

# E2E tests
npm run test:e2e

# Coverage report
npm run test:coverage
```

## Configuration

### Environment Variables

Create `.env.development` for development:

```env
VITE_API_URL=/api/v1
VITE_WS_URL=/ws
VITE_APP_TITLE=MBFlow - Workflow Orchestration
```

For production, create `.env.production`:

```env
VITE_API_URL=http://your-backend-url:8181/api/v1
VITE_WS_URL=ws://your-backend-url:8181/ws
```

### Backend Integration

The UI expects the MBFlow backend to be running on `http://localhost:8181`.

Start the backend first:

```bash
cd ../backend
go run cmd/server/main.go
```

## Key Components

### Layout System

- **AppShell** - Main application container with sidebar and header
- **AppHeader** - Top navigation bar with user menu
- **AppSidebar** - Collapsible navigation sidebar

### UI Components

- **Button** - Primary, secondary, danger, and ghost variants
- **Input** - Form input with validation error support
- More components coming soon...

### Pages (Auto-routed)

- `/` - Dashboard with stats
- `/workflows` - Workflows list with CRUD operations
- `/workflows/new` - Create new workflow
- `/workflows/:id` - Visual workflow editor with DAG canvas
- `/executions` - Executions list (coming soon)
- `/triggers` - Triggers management (coming soon)

## Roadmap

### Phase 1: Core UI ✅
- [x] Project setup with Vite + Vue 3 + TypeScript
- [x] TailwindCSS configuration
- [x] Layout system (AppShell, Header, Sidebar)
- [x] Basic UI components
- [x] File-based routing
- [x] API client setup

### Phase 2: Workflow Editor ✅
- [x] Vue Flow canvas integration
- [x] Custom node components (HTTP, LLM, Transform, Conditional, Merge)
- [x] Node configuration panel with dynamic forms
- [x] Auto-layout with ELK/dagre algorithms
- [x] Drag-and-drop node palette
- [x] Edge management and connections
- [x] Workflow toolbar with save/execute/validate actions

### Phase 3: Data Integration ✅ (Partial)
- [x] API endpoints for workflows (CRUD)
- [x] Real API integration
- [x] Workflow store with Pinia
- [x] Auto-layout composable
- [ ] TanStack Query composables (using direct API calls for now)
- [ ] WebSocket for real-time updates
- [ ] Form validation with VeeValidate

### Phase 4: Advanced Features 📋
- [ ] Execution monitoring
- [ ] Trigger management
- [ ] Workflow templates
- [ ] Export/import workflows
- [ ] Dark mode

## Contributing

Detailed plan available at: `.claude/plans/sharded-marinating-dove.md`

## License

Part of MBFlow project.

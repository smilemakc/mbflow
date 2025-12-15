# Pages

React page components for the MBFlow application.

---

## ExecutionsPage

**Location:** `/pages/ExecutionsPage.tsx`
**Route:** `/executions`

### Overview

Comprehensive execution history page for viewing and managing all workflow executions with filtering, pagination, and detailed views.

### Key Features

#### 1. Execution Table
- **ID Column**: Short UUID (8 chars)
- **Workflow Column**: Name + short UUID
- **Status Column**: Color-coded badges with icons
- **Started At**: Formatted date/time
- **Duration**: Human-readable format (ms, s, m)
- **Triggered By**: User/System badge
- **Actions**: View details + Retry (for failed)

#### 2. Advanced Filters
- **Workflow Filter**: Dropdown of all workflows
- **Status Filter**: pending, running, completed, failed, cancelled
- **Date Range**: Date picker for time range
- **Actions**: Apply/Clear filters

#### 3. Pagination
- 20 items per page
- Previous/Next navigation
- Results counter (showing X-Y of Z)

#### 4. Execution Details Modal
- Full execution metadata
- Node execution statuses
- Input/Output data (JSON formatted)
- Error messages (for failed executions)
- Retry action button

#### 5. Status Indicators
| Status | Color | Icon |
|--------|-------|------|
| completed | Green | CheckCircle |
| failed | Red | XCircle |
| running | Blue | Loader2 (animated) |
| pending | Yellow | Clock |
| cancelled | Gray | Pause |

### Usage

```tsx
// In App.tsx router
<Route path="/executions" element={
  <PageWrapper title="Execution History">
    <ExecutionsPage />
  </PageWrapper>
} />
```

### Dependencies
- `react`, `lucide-react`, `executionService`, `workflowService`, `useTranslation`, Tailwind CSS

---

## WorkflowsPage

React component for displaying and managing workflow list.

## Location
`/Users/balashov/PycharmProjects/mbflow/mbflow-react/pages/WorkflowsPage.tsx`

## Features

### 1. Workflow List Display
- Grid layout with responsive cards (1/2/3 columns)
- Card shows:
  - Workflow name and description
  - Status badge with color coding
  - Number of nodes
  - Created and updated dates
  - Action buttons (Edit, Clone, Delete)

### 2. Search and Filtering
- Full-text search by workflow name and description
- Status filter dropdown (all/draft/active/inactive/archived)
- Real-time filtering with results count

### 3. Pagination
- 12 workflows per page
- Page navigation controls
- Current page highlighting

### 4. Status Management
Status badges with distinct colors:
- **Draft**: Slate (default for new workflows)
- **Active**: Green (running workflows)
- **Inactive**: Orange (paused workflows)
- **Archived**: Gray (archived workflows)

### 5. Workflow Actions
- **Create New**: Navigate to builder with empty workflow
- **Edit**: Open workflow in builder
- **Clone**: Duplicate workflow with "(Copy)" suffix
- **Delete**: Delete workflow with confirmation

### 6. Empty States
- No workflows: CTA to create first workflow
- No results: Message to adjust filters
- Loading state with spinner
- Error state with retry button

## Usage

```tsx
import { WorkflowsPage } from './pages/WorkflowsPage';

// In routing:
<Route path="/workflows" element={
  <PageWrapper title="Workflows">
    <WorkflowsPage />
  </PageWrapper>
} />
```

## Dependencies

- `workflowService` - API calls for workflow CRUD operations
- `react-router-dom` - Navigation
- `lucide-react` - Icons
- Tailwind CSS - Styling

## API Integration

Uses `workflowService` methods:
- `getAll()` - Load all workflows
- `create(name, description)` - Create new workflow
- `save(workflow)` - Update workflow (for cloning)
- `delete(id)` - Delete workflow

## Routing

Added to App.tsx:
```tsx
<Route path="/workflows" element={
  <PageWrapper title="Workflows">
    <WorkflowsPage />
  </PageWrapper>
} />
```

Sidebar navigation:
```tsx
<NavItem
  icon={<FolderOpen size={20} />}
  label="Workflows List"
  active={isActive('/workflows')}
  onClick={() => navigate('/workflows')}
/>
```

## Date Formatting

Relative date display:
- "Today" - today
- "Yesterday" - yesterday
- "N days ago" - within a week
- "Mon DD, YYYY" - older dates

## State Management

Local state only:
- `workflows` - All workflows from API
- `filteredWorkflows` - Filtered results
- `searchQuery` - Current search term
- `statusFilter` - Selected status filter
- `currentPage` - Current pagination page
- `isLoading` - Loading state
- `error` - Error message

## Future Enhancements

- Bulk operations (multi-select)
- Sort by name/date/status
- Workflow tags/categories
- Export/import workflows
- Workflow templates preview
- Advanced filters (date range, owner, tags)
- Workflow statistics on card (execution count, success rate)

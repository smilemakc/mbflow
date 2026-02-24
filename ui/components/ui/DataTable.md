# DataTable Component

A reusable, feature-rich table component that eliminates duplicate table code across the application.

## Features

- **Flexible Column Configuration**: Define columns with custom rendering, alignment, and width
- **Loading States**: Built-in loading spinner for async data
- **Error Handling**: Display error messages with ErrorBanner component
- **Empty States**: Customizable empty state with icon, title, description, and action button
- **Pagination**: Built-in pagination controls with offset/limit
- **Row Actions**: Optional actions column with custom buttons
- **Row Interactions**: Support for clickable rows and custom row styling
- **Dark Mode**: Full dark mode support
- **Compact Mode**: Optional compact layout for dense tables
- **Type Safe**: Full TypeScript support with generic types

## Basic Usage

```tsx
import { DataTable, Column } from '@/components/ui';
import { User } from '@/types';

const columns: Column<User>[] = [
  {
    key: 'username',
    header: 'Username',
    render: (user) => <span className="font-medium">{user.username}</span>,
  },
  {
    key: 'email',
    header: 'Email',
  },
  {
    key: 'status',
    header: 'Status',
    align: 'center',
  },
];

<DataTable
  data={users}
  columns={columns}
  keyExtractor={(user) => user.id}
  loading={loading}
  error={error}
  emptyIcon={User}
  emptyTitle="No users found"
  emptyDescription="Add your first user to get started"
/>
```

## Column Configuration

```tsx
interface Column<T> {
  key: string;              // Unique column identifier
  header: string | React.ReactNode;  // Column header
  width?: string;           // CSS width (e.g., "150px", "20%")
  align?: 'left' | 'center' | 'right';  // Text alignment
  sortable?: boolean;       // Future: Enable sorting
  render?: (item: T, index: number) => React.ReactNode;  // Custom cell renderer
}
```

## Props

### Data Props
- `data: T[]` - Array of data items
- `columns: Column<T>[]` - Column configuration
- `keyExtractor: (item: T) => string` - Extract unique key from item

### State Props
- `loading?: boolean` - Show loading state
- `error?: string | null` - Display error message

### Empty State Props
- `emptyIcon?: React.ElementType` - Icon component (from lucide-react)
- `emptyTitle?: string` - Empty state title
- `emptyDescription?: string` - Empty state description
- `emptyAction?: { label, onClick, icon }` - Call-to-action button

### Row Features
- `onRowClick?: (item: T) => void` - Row click handler
- `rowClassName?: (item: T) => string` - Dynamic row styling

### Actions Column
- `actions?: (item: T) => React.ReactNode` - Render action buttons
- `actionsHeader?: string` - Actions column header (default: "Actions")

### Pagination
- `pagination?: { offset, limit, total, onOffsetChange }` - Pagination config

### Styling
- `className?: string` - Additional CSS classes
- `compact?: boolean` - Use compact padding

## Examples

### With Pagination

```tsx
<DataTable
  data={executions}
  columns={columns}
  keyExtractor={(item) => item.id}
  pagination={{
    offset: 0,
    limit: 20,
    total: 100,
    onOffsetChange: (newOffset) => setOffset(newOffset),
  }}
/>
```

### With Actions

```tsx
<DataTable
  data={items}
  columns={columns}
  keyExtractor={(item) => item.id}
  actions={(item) => (
    <>
      <Button
        variant="ghost"
        size="sm"
        icon={<Edit2 size={14} />}
        onClick={() => handleEdit(item)}
      />
      <Button
        variant="ghost"
        size="sm"
        icon={<Trash2 size={14} />}
        onClick={() => handleDelete(item)}
      />
    </>
  )}
/>
```

### With Clickable Rows

```tsx
<DataTable
  data={workflows}
  columns={columns}
  keyExtractor={(item) => item.id}
  onRowClick={(workflow) => navigate(`/workflows/${workflow.id}`)}
  rowClassName={(workflow) =>
    workflow.status === 'archived' ? 'opacity-50' : ''
  }
/>
```

### With Empty State Action

```tsx
<DataTable
  data={triggers}
  columns={columns}
  keyExtractor={(item) => item.id}
  emptyIcon={Calendar}
  emptyTitle="No triggers found"
  emptyDescription="Create your first trigger to automate workflows"
  emptyAction={{
    label: 'Create Trigger',
    onClick: () => setShowModal(true),
    icon: <Plus size={16} />,
  }}
/>
```

## Code Elimination

This component can replace ~800 lines of duplicate table code across:

### 1. RentalKeyAdminList.tsx (~200 lines)
- Table structure (lines 372-427)
- Pagination logic (lines 156-172, 430-452)
- Loading state (lines 401-406)
- Empty state (lines 407-412)

### 2. TriggersPage.tsx (~220 lines)
- Table structure (lines 254-378)
- Loading state (lines 227-230)
- Empty state (lines 231-252)
- Filter results display (lines 221-223)

### 3. ExecutionsPage.tsx (~200 lines)
- Table structure (lines 278-401)
- Loading state (lines 280-283)
- Empty state (lines 284-289)
- Pagination (lines 374-398)

### 4. UsersPage.tsx (~180 lines)
- Table structure (lines 120-264)
- Loading state (lines 121-126)
- Pagination (lines 266-290)

## Migration Guide

### Before (RentalKeyAdminList)
```tsx
<div className="bg-white dark:bg-slate-900 border rounded-lg">
  <table className="w-full">
    <thead className="bg-slate-50 dark:bg-slate-800">
      <tr>
        <th className="px-4 py-3 text-left text-xs font-medium...">Name</th>
        {/* More headers */}
      </tr>
    </thead>
    <tbody className="divide-y">
      {loading ? (
        <tr><td colSpan={7}><Spinner /></td></tr>
      ) : rentalKeys.length === 0 ? (
        <tr><td colSpan={7}>No data</td></tr>
      ) : (
        rentalKeys.map(key => <tr>...</tr>)
      )}
    </tbody>
  </table>
  {/* Custom pagination */}
</div>
```

### After
```tsx
<DataTable
  data={rentalKeys}
  columns={columns}
  keyExtractor={(key) => key.id}
  loading={loading}
  emptyIcon={Key}
  emptyTitle="No rental keys found"
  emptyDescription="Create a rental key to start using the API"
  pagination={{ offset, limit, total, onOffsetChange: setOffset }}
  actions={(key) => <RentalKeyActions key={key} />}
/>
```

## Benefits

1. **Consistency**: All tables use the same structure and styling
2. **Maintainability**: Bug fixes and features in one place
3. **Type Safety**: Generic types catch errors at compile time
4. **Less Boilerplate**: ~80% reduction in table code
5. **Accessibility**: Built-in keyboard navigation and ARIA attributes
6. **Performance**: Optimized rendering with proper key extraction
7. **Dark Mode**: Automatic dark mode support
8. **Responsive**: Horizontal scroll on small screens

## Styling Tokens

The component uses these design tokens:

- **Background**: `bg-white dark:bg-slate-900`
- **Border**: `border-slate-200 dark:border-slate-800`
- **Header**: `bg-slate-50 dark:bg-slate-800`
- **Text**: `text-slate-900 dark:text-white`
- **Muted**: `text-slate-500 dark:text-slate-400`
- **Hover**: `hover:bg-slate-50 dark:hover:bg-slate-800/50`
- **Divider**: `divide-slate-200 dark:divide-slate-700`

## Future Enhancements

- [ ] Column sorting with `sortable` prop
- [ ] Column resizing with drag handles
- [ ] Column visibility toggle
- [ ] Row selection with checkboxes
- [ ] Bulk actions on selected rows
- [ ] Virtual scrolling for large datasets
- [ ] Export to CSV/Excel
- [ ] Sticky header on scroll
- [ ] Column filters
- [ ] Mobile-responsive card view

## Related Components

- `EmptyState` - Used for empty data display
- `LoadingState` - Used for loading spinner
- `ErrorBanner` - Used for error display
- `Button` - Used for pagination and actions
- `Pagination` - Standalone pagination component

/**
 * DataTable Usage Examples
 *
 * This file demonstrates various use cases of the DataTable component.
 * It can be used as a reference when implementing tables in the application.
 */

import React, { useState } from 'react';
import { Edit2, Trash2, Play, Key, User, Calendar } from 'lucide-react';
import { DataTable, Column, Button } from '@/components/ui';

// Example 1: Simple table with user data
interface User {
  id: string;
  username: string;
  email: string;
  is_active: boolean;
  created_at: string;
}

export const SimpleUserTable: React.FC = () => {
  const [users, setUsers] = useState<User[]>([
    { id: '1', username: 'john_doe', email: 'john@example.com', is_active: true, created_at: '2024-01-15' },
    { id: '2', username: 'jane_smith', email: 'jane@example.com', is_active: false, created_at: '2024-02-20' },
  ]);

  const columns: Column<User>[] = [
    {
      key: 'username',
      header: 'Username',
      render: (user) => (
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-full bg-blue-500 flex items-center justify-center text-white text-xs font-medium">
            {user.username.slice(0, 2).toUpperCase()}
          </div>
          <span className="font-medium text-slate-900 dark:text-white">{user.username}</span>
        </div>
      ),
    },
    {
      key: 'email',
      header: 'Email',
      render: (user) => (
        <span className="text-sm text-slate-600 dark:text-slate-300">{user.email}</span>
      ),
    },
    {
      key: 'is_active',
      header: 'Status',
      align: 'center',
      render: (user) => (
        <span
          className={`inline-flex px-2 py-0.5 text-xs font-medium rounded-full ${
            user.is_active
              ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
              : 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-400'
          }`}
        >
          {user.is_active ? 'Active' : 'Inactive'}
        </span>
      ),
    },
    {
      key: 'created_at',
      header: 'Created',
      render: (user) => (
        <span className="text-sm text-slate-500 dark:text-slate-400">
          {new Date(user.created_at).toLocaleDateString()}
        </span>
      ),
    },
  ];

  return (
    <DataTable
      data={users}
      columns={columns}
      keyExtractor={(user) => user.id}
      emptyIcon={User}
      emptyTitle="No users found"
      emptyDescription="Add your first user to get started"
      actions={(user) => (
        <>
          <Button variant="ghost" size="sm" icon={<Edit2 size={14} />} title="Edit" />
          <Button variant="ghost" size="sm" icon={<Trash2 size={14} />} title="Delete" />
        </>
      )}
    />
  );
};

// Example 2: Table with pagination and loading state
interface Execution {
  id: string;
  workflow_name: string;
  status: 'completed' | 'failed' | 'running';
  started_at: string;
  duration_ms: number;
}

export const ExecutionsTableWithPagination: React.FC = () => {
  const [executions] = useState<Execution[]>([
    { id: '1', workflow_name: 'Data Pipeline', status: 'completed', started_at: '2024-03-15T10:30:00Z', duration_ms: 5432 },
    { id: '2', workflow_name: 'Email Sender', status: 'running', started_at: '2024-03-15T11:00:00Z', duration_ms: 0 },
  ]);
  const [loading] = useState(false);
  const [offset, setOffset] = useState(0);
  const [total] = useState(50);

  const columns: Column<Execution>[] = [
    {
      key: 'id',
      header: 'ID',
      width: '100px',
      render: (exec) => (
        <span className="font-mono text-xs text-slate-500 dark:text-slate-400">
          {exec.id.substring(0, 8)}
        </span>
      ),
    },
    {
      key: 'workflow_name',
      header: 'Workflow',
      render: (exec) => (
        <span className="font-medium text-slate-900 dark:text-white">{exec.workflow_name}</span>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      align: 'center',
      render: (exec) => {
        const colors = {
          completed: 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400',
          failed: 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-400',
          running: 'bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-400',
        };
        return (
          <span className={`inline-flex px-2 py-0.5 text-xs font-medium rounded-full ${colors[exec.status]}`}>
            {exec.status}
          </span>
        );
      },
    },
    {
      key: 'started_at',
      header: 'Started At',
      render: (exec) => (
        <span className="text-sm text-slate-600 dark:text-slate-300">
          {new Date(exec.started_at).toLocaleString()}
        </span>
      ),
    },
    {
      key: 'duration_ms',
      header: 'Duration',
      align: 'right',
      render: (exec) => (
        <span className="font-mono text-xs text-slate-500 dark:text-slate-400">
          {exec.duration_ms > 0 ? `${(exec.duration_ms / 1000).toFixed(2)}s` : '-'}
        </span>
      ),
    },
  ];

  return (
    <DataTable
      data={executions}
      columns={columns}
      keyExtractor={(exec) => exec.id}
      loading={loading}
      emptyIcon={Calendar}
      emptyTitle="No executions found"
      emptyDescription="Run a workflow to see executions here"
      pagination={{
        offset,
        limit: 20,
        total,
        onOffsetChange: setOffset,
      }}
      actions={(exec) => (
        <>
          <Button variant="ghost" size="sm" icon={<Play size={14} />} title="Retry" />
        </>
      )}
    />
  );
};

// Example 3: Table with clickable rows and error handling
interface RentalKey {
  id: string;
  name: string;
  provider: string;
  status: 'active' | 'suspended';
  total_requests: number;
  total_cost: number;
}

export const RentalKeysTable: React.FC = () => {
  const [keys] = useState<RentalKey[]>([
    { id: '1', name: 'Production Key', provider: 'OpenAI', status: 'active', total_requests: 1500, total_cost: 45.67 },
    { id: '2', name: 'Test Key', provider: 'Anthropic', status: 'suspended', total_requests: 250, total_cost: 12.34 },
  ]);
  const [error] = useState<string | null>(null);

  const handleRowClick = (key: RentalKey) => {
    console.log('Clicked key:', key.id);
  };

  const columns: Column<RentalKey>[] = [
    {
      key: 'name',
      header: 'Name',
      render: (key) => (
        <div>
          <div className="font-medium text-slate-900 dark:text-white">{key.name}</div>
          <div className="text-xs text-slate-500 dark:text-slate-400 font-mono">ID: {key.id.slice(0, 8)}</div>
        </div>
      ),
    },
    {
      key: 'provider',
      header: 'Provider',
      render: (key) => (
        <span className="text-sm text-slate-900 dark:text-white">{key.provider}</span>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      align: 'center',
      render: (key) => (
        <span
          className={`inline-flex px-2 py-0.5 text-xs font-medium rounded-full ${
            key.status === 'active'
              ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
              : 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-400'
          }`}
        >
          {key.status}
        </span>
      ),
    },
    {
      key: 'total_requests',
      header: 'Usage',
      align: 'right',
      render: (key) => (
        <div className="text-right">
          <div className="text-sm text-slate-900 dark:text-white">{key.total_requests.toLocaleString()} req</div>
          <div className="text-xs text-slate-500 dark:text-slate-400">${key.total_cost.toFixed(2)}</div>
        </div>
      ),
    },
  ];

  return (
    <DataTable
      data={keys}
      columns={columns}
      keyExtractor={(key) => key.id}
      error={error}
      emptyIcon={Key}
      emptyTitle="No rental keys found"
      emptyDescription="Create a rental key to start using the API"
      emptyAction={{
        label: 'Create Key',
        onClick: () => console.log('Create key'),
        icon: <Key size={16} />,
      }}
      onRowClick={handleRowClick}
      rowClassName={(key) => (key.status === 'suspended' ? 'opacity-60' : '')}
      actions={(key) => (
        <>
          <Button variant="ghost" size="sm" icon={<Edit2 size={14} />} title="Edit" />
          <Button variant="ghost" size="sm" icon={<Trash2 size={14} />} title="Delete" />
        </>
      )}
    />
  );
};

// Example 4: Compact table
export const CompactTable: React.FC = () => {
  const data = [
    { id: '1', name: 'Item 1', value: 100 },
    { id: '2', name: 'Item 2', value: 200 },
  ];

  const columns: Column<typeof data[0]>[] = [
    { key: 'name', header: 'Name' },
    { key: 'value', header: 'Value', align: 'right' },
  ];

  return (
    <DataTable
      data={data}
      columns={columns}
      keyExtractor={(item) => item.id}
      compact
      emptyIcon={Calendar}
      emptyTitle="No items"
      emptyDescription="Add items to get started"
    />
  );
};

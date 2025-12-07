import React from 'react';
import {
  CheckCircle,
  XCircle,
  Clock,
  Loader2,
  Pause,
  AlertCircle
} from 'lucide-react';
import { ExecutionStatus } from '@/types/execution';

export const getStatusIcon = (status: ExecutionStatus, size: number = 16): React.ReactNode => {
  const iconProps = { size };

  switch (status) {
    case 'completed':
      return React.createElement(CheckCircle, { ...iconProps, className: 'text-green-600 dark:text-green-400' });
    case 'failed':
      return React.createElement(XCircle, { ...iconProps, className: 'text-red-600 dark:text-red-400' });
    case 'running':
      return React.createElement(Loader2, { ...iconProps, className: 'text-blue-600 dark:text-blue-400 animate-spin' });
    case 'pending':
      return React.createElement(Clock, { ...iconProps, className: 'text-yellow-600 dark:text-yellow-400' });
    case 'cancelled':
      return React.createElement(Pause, { ...iconProps, className: 'text-gray-600 dark:text-gray-400' });
    default:
      return React.createElement(AlertCircle, { ...iconProps, className: 'text-gray-600 dark:text-gray-400' });
  }
};

export const getStatusBadgeClass = (status: ExecutionStatus): string => {
  switch (status) {
    case 'completed':
      return 'bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-400 dark:border-green-900/30';
    case 'failed':
      return 'bg-red-50 text-red-700 border-red-200 dark:bg-red-900/20 dark:text-red-400 dark:border-red-900/30';
    case 'running':
      return 'bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-900/20 dark:text-blue-400 dark:border-blue-900/30';
    case 'pending':
      return 'bg-yellow-50 text-yellow-700 border-yellow-200 dark:bg-yellow-900/20 dark:text-yellow-400 dark:border-yellow-900/30';
    case 'cancelled':
      return 'bg-gray-50 text-gray-700 border-gray-200 dark:bg-gray-900/20 dark:text-gray-400 dark:border-gray-900/30';
    default:
      return 'bg-gray-50 text-gray-700 border-gray-200 dark:bg-gray-900/20 dark:text-gray-400 dark:border-gray-900/30';
  }
};

export const formatDuration = (startedAt: string, completedAt?: string): string => {
  if (!completedAt) return '-';
  const start = new Date(startedAt).getTime();
  const end = new Date(completedAt).getTime();
  const durationMs = end - start;

  if (durationMs < 1000) return `${durationMs}ms`;
  if (durationMs < 60000) return `${(durationMs / 1000).toFixed(2)}s`;
  return `${(durationMs / 60000).toFixed(2)}m`;
};

export const formatDate = (dateStr: string): string => {
  const date = new Date(dateStr);
  return new Intl.DateTimeFormat('en-US', {
    weekday: 'short',
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }).format(date);
};

export interface ExecutionStats {
  total: number;
  completed: number;
  failed: number;
  running: number;
  pending: number;
}

export const calculateStats = (nodeExecutions: { status: ExecutionStatus }[] | undefined): ExecutionStats => {
  if (!nodeExecutions) {
    return { total: 0, completed: 0, failed: 0, running: 0, pending: 0 };
  }

  return nodeExecutions.reduce(
    (acc, ne) => {
      acc.total++;
      if (ne.status === 'completed') acc.completed++;
      if (ne.status === 'failed') acc.failed++;
      if (ne.status === 'running') acc.running++;
      if (ne.status === 'pending') acc.pending++;
      return acc;
    },
    { total: 0, completed: 0, failed: 0, running: 0, pending: 0 }
  );
};

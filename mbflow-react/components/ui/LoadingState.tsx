import React from 'react';
import { Loader2 } from 'lucide-react';

interface LoadingStateProps {
  message?: string;
  size?: number;
}

export const LoadingState: React.FC<LoadingStateProps> = ({ message, size = 32 }) => (
  <div className="flex flex-col items-center justify-center py-20">
    <Loader2 size={size} className="animate-spin text-blue-600 dark:text-blue-400" />
    {message && (
      <p className="mt-4 text-sm text-slate-600 dark:text-slate-400">{message}</p>
    )}
  </div>
);

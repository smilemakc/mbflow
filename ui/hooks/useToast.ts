import { useToastStore, Toast, ToastType } from '../lib/toast';

export type { Toast, ToastType };

export interface ToastOptions {
  type?: ToastType;
  title: string;
  message?: string;
  duration?: number;
}

export interface UseToast {
  toasts: Toast[];
  showToast: (options: ToastOptions) => string;
  success: (title: string, message?: string) => string;
  error: (title: string, message?: string) => string;
  warning: (title: string, message?: string) => string;
  info: (title: string, message?: string) => string;
  dismissToast: (id: string) => void;
  clearAll: () => void;
}

export function useToast(): UseToast {
  const { toasts, addToast, removeToast, clearAll } = useToastStore();

  const showToast = (options: ToastOptions): string => {
    const duration = options.duration ?? 5000;
    const type = options.type ?? 'info';

    return addToast({
      type,
      title: options.title,
      message: options.message,
      duration,
    });
  };

  const success = (title: string, message?: string): string => {
    return addToast({
      type: 'success',
      title,
      message,
      duration: 5000,
    });
  };

  const error = (title: string, message?: string): string => {
    return addToast({
      type: 'error',
      title,
      message,
      duration: 7000,
    });
  };

  const warning = (title: string, message?: string): string => {
    return addToast({
      type: 'warning',
      title,
      message,
      duration: 5000,
    });
  };

  const info = (title: string, message?: string): string => {
    return addToast({
      type: 'info',
      title,
      message,
      duration: 5000,
    });
  };

  return {
    toasts,
    showToast,
    success,
    error,
    warning,
    info,
    dismissToast: removeToast,
    clearAll,
  };
}

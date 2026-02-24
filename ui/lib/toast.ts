import { create } from 'zustand';

export type ToastType = 'success' | 'error' | 'warning' | 'info';

export interface Toast {
  id: string;
  type: ToastType;
  title: string;
  message?: string;
  duration: number;
}

interface ToastStore {
  toasts: Toast[];
  addToast: (toast: Omit<Toast, 'id'>) => string;
  removeToast: (id: string) => void;
  clearAll: () => void;
}

let nextId = 0;

export const useToastStore = create<ToastStore>((set) => ({
  toasts: [],

  addToast: (toast) => {
    const id = `toast-${++nextId}`;
    const newToast: Toast = { id, ...toast };

    set((state) => ({
      toasts: [...state.toasts, newToast],
    }));

    if (toast.duration > 0) {
      setTimeout(() => {
        set((state) => ({
          toasts: state.toasts.filter((t) => t.id !== id),
        }));
      }, toast.duration);
    }

    return id;
  },

  removeToast: (id) => {
    set((state) => ({
      toasts: state.toasts.filter((t) => t.id !== id),
    }));
  },

  clearAll: () => {
    set({ toasts: [] });
  },
}));

// Standalone toast functions that can be called from anywhere (including stores)
export const toast = {
  success: (title: string, message?: string) => {
    return useToastStore.getState().addToast({
      type: 'success',
      title,
      message,
      duration: 5000,
    });
  },

  error: (title: string, message?: string) => {
    return useToastStore.getState().addToast({
      type: 'error',
      title,
      message,
      duration: 7000,
    });
  },

  warning: (title: string, message?: string) => {
    return useToastStore.getState().addToast({
      type: 'warning',
      title,
      message,
      duration: 5000,
    });
  },

  info: (title: string, message?: string) => {
    return useToastStore.getState().addToast({
      type: 'info',
      title,
      message,
      duration: 5000,
    });
  },

  dismiss: (id: string) => {
    useToastStore.getState().removeToast(id);
  },

  clearAll: () => {
    useToastStore.getState().clearAll();
  },
};

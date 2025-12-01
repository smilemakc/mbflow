import { ref } from "vue";
import type { Toast, ToastType, ToastOptions } from "@/types/toast";

const toasts = ref<Toast[]>([]);

let nextId = 0;

export function useToast() {
  function addToast(type: ToastType, options: ToastOptions): string {
    const id = `toast-${++nextId}`;
    const duration = options.duration ?? 5000;

    const toast: Toast = {
      id,
      type,
      ...options,
      duration,
    };

    toasts.value.push(toast);

    if (duration > 0) {
      setTimeout(() => {
        removeToast(id);
      }, duration);
    }

    return id;
  }

  function removeToast(id: string) {
    const index = toasts.value.findIndex((t) => t.id === id);
    if (index !== -1) {
      toasts.value.splice(index, 1);
    }
  }

  function success(options: ToastOptions | string) {
    const opts = typeof options === "string" ? { title: options } : options;
    return addToast("success", opts);
  }

  function error(options: ToastOptions | string) {
    const opts = typeof options === "string" ? { title: options } : options;
    return addToast("error", opts);
  }

  function warning(options: ToastOptions | string) {
    const opts = typeof options === "string" ? { title: options } : options;
    return addToast("warning", opts);
  }

  function info(options: ToastOptions | string) {
    const opts = typeof options === "string" ? { title: options } : options;
    return addToast("info", opts);
  }

  function clear() {
    toasts.value = [];
  }

  return {
    toasts,
    success,
    error,
    warning,
    info,
    removeToast,
    clear,
  };
}

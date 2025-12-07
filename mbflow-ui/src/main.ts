import { createApp } from "vue";
import { createPinia } from "pinia";
import { createRouter, createWebHistory } from "vue-router";
import { VueQueryPlugin, QueryClient } from "@tanstack/vue-query";
import Vue3Toastify, { type ToastContainerOptions } from "vue3-toastify";
import "vue3-toastify/dist/index.css";
import App from "./App.vue";

// Import global styles
import "./assets/styles/main.css";
import { routes } from "vue-router/auto-routes";

const app = createApp(App);

// Create router with auto-generated routes from unplugin-vue-router
const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
});

// Create query client for TanStack Query
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 5 * 60 * 1000, // 5 minutes
    },
  },
});

// Install plugins
app.use(createPinia());
app.use(router);
app.use(VueQueryPlugin, { queryClient });
app.use(Vue3Toastify, {
  autoClose: 3000,
  position: "top-right",
  theme: "auto",
} as ToastContainerOptions);

// Mount app
app.mount("#app");

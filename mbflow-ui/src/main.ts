import { createApp } from "vue";
import { createPinia } from "pinia";
import { createRouter, createWebHistory } from "vue-router";
import { VueQueryPlugin, QueryClient } from "@tanstack/vue-query";
import App from "./App.vue";

// Import global styles
import "./assets/styles/main.css";

// Import pages manually
import IndexPage from "./pages/index.vue";
import WorkflowsIndex from "./pages/workflows/index.vue";
import WorkflowsNew from "./pages/workflows/new.vue";
import WorkflowsDetail from "./pages/workflows/[id].vue";
import ExecutionsIndex from "./pages/executions/index.vue";
import ExecutionsDetail from "./pages/executions/[id].vue";
import TriggersIndex from "./pages/triggers/index.vue";

const app = createApp(App);

// Create router with manually defined routes
const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: "/", component: IndexPage },
    { path: "/workflows", component: WorkflowsIndex },
    { path: "/workflows/new", component: WorkflowsNew },
    { path: "/workflows/:id", component: WorkflowsDetail },
    { path: "/executions", component: ExecutionsIndex },
    { path: "/executions/:id", component: ExecutionsDetail },
    { path: "/triggers", component: TriggersIndex },
  ],
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

// Mount app
app.mount("#app");

import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import VueRouter from "unplugin-vue-router/vite";
import { fileURLToPath, URL } from "node:url";

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    // Auto-routing must come before Vue plugin
    VueRouter({
      routesFolder: "src/pages",
      dts: "src/typed-router.d.ts",
      extensions: [".vue"],
      exclude: ["**/components/**"],
    }),
    vue(),
  ],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  server: {
    port: 3434,
    proxy: {
      "/api": {
        target: "http://localhost:8181",
        changeOrigin: true,
      },
      "/ws": {
        target: "ws://localhost:8181",
        ws: true,
      },
    },
  },
  build: {
    sourcemap: true,
    rollupOptions: {
      output: {
        manualChunks: {
          "vue-vendor": ["vue", "vue-router", "pinia"],
          "vue-flow": [
            "@vue-flow/core",
            "@vue-flow/background",
            "@vue-flow/controls",
            "@vue-flow/minimap",
          ],
          "tanstack-query": ["@tanstack/vue-query"],
        },
      },
    },
  },
});

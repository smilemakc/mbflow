<script setup lang="ts">
import AppHeader from "./AppHeader.vue";
import AppSidebar from "./AppSidebar.vue";
import { useUIStore } from "@/stores/ui";

const uiStore = useUIStore();
</script>

<template>
  <div class="h-screen w-full overflow-x-auto overflow-y-hidden bg-gray-50">
    <div class="flex h-full min-w-[1024px]">
      <!-- Sidebar -->
      <AppSidebar v-model:open="uiStore.sidebarOpen" class="shrink-0" />

      <!-- Main content -->
      <div class="flex flex-1 flex-col overflow-hidden">
        <!-- Header -->
        <AppHeader @toggle-sidebar="uiStore.toggleSidebar" />

        <!-- Content area -->
        <main class="flex-1 overflow-y-auto overflow-x-auto p-6">
          <slot />
        </main>
      </div>
    </div>
  </div>
</template>

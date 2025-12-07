<script setup lang="ts">
import { useRoute } from "vue-router";
import { Icon } from "@iconify/vue";
import { cn } from "@/utils/cn";

interface Props {
  open: boolean;
}

defineProps<Props>();
defineEmits<{
  "update:open": [value: boolean];
}>();

const route = useRoute();

const menuItems = [
  { title: "Dashboard", icon: "heroicons:home", to: "/" },
  { title: "Workflows", icon: "heroicons:squares-2x2", to: "/workflows" },
  {
    title: "Templates",
    icon: "heroicons:document-duplicate",
    to: "/templates",
  },
  { title: "Executions", icon: "heroicons:play-circle", to: "/executions" },
  { title: "Triggers", icon: "heroicons:bell-alert", to: "/triggers" },
  { title: "Settings", icon: "heroicons:cog-6-tooth", to: "/settings" },
];

const isActive = (path: string) => {
  if (path === "/") {
    return route.path === "/";
  }
  return route.path.startsWith(path);
};
</script>

<template>
  <aside
    :class="
      cn(
        'flex flex-col bg-white border-r border-gray-200 transition-all duration-300',
        open ? 'w-64' : 'w-20',
      )
    "
  >
    <div class="flex-1 overflow-y-auto py-4">
      <nav class="space-y-1 px-3">
        <router-link
          v-for="item in menuItems"
          :key="item.to"
          :to="item.to"
          :class="
            cn(
              'flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium transition-colors',
              isActive(item.to)
                ? 'bg-blue-50 text-blue-700'
                : 'text-gray-700 hover:bg-gray-50 hover:text-gray-900',
            )
          "
        >
          <Icon :icon="item.icon" class="size-5 shrink-0" />
          <span v-if="open" class="truncate">{{ item.title }}</span>
        </router-link>
      </nav>
    </div>
  </aside>
</template>

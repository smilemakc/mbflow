<script setup lang="ts">
import { Icon } from "@iconify/vue";
import { Menu, MenuButton, MenuItems, MenuItem } from "@headlessui/vue";

const emit = defineEmits<{
  "toggle-sidebar": [];
}>();
</script>

<template>
  <header class="border-b border-gray-200 bg-white px-6 py-4">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-4">
        <button
          @click="emit('toggle-sidebar')"
          class="text-gray-500 hover:text-gray-700 focus:outline-none"
        >
          <Icon icon="heroicons:bars-3" class="size-6" />
        </button>

        <h1 class="text-xl font-bold text-gray-900">MBFlow</h1>
      </div>

      <div class="flex items-center gap-4">
        <!-- Theme toggle -->
        <button class="text-gray-500 hover:text-gray-700">
          <Icon icon="heroicons:moon" class="size-5" />
        </button>

        <!-- Notifications -->
        <button class="text-gray-500 hover:text-gray-700">
          <Icon icon="heroicons:bell" class="size-5" />
        </button>

        <!-- User menu -->
        <Menu as="div" class="relative">
          <MenuButton
            class="flex items-center text-gray-700 hover:text-gray-900"
          >
            <Icon icon="heroicons:user-circle" class="size-8" />
          </MenuButton>

          <MenuItems
            class="absolute right-0 z-10 mt-2 w-48 rounded-md bg-white shadow-lg ring-1 ring-black/5 focus:outline-none"
          >
            <MenuItem v-slot="{ active }">
              <a
                href="/settings"
                :class="[
                  active ? 'bg-gray-100' : '',
                  'block px-4 py-2 text-sm text-gray-700',
                ]"
              >
                Settings
              </a>
            </MenuItem>
            <MenuItem v-slot="{ active }">
              <button
                :class="[
                  active ? 'bg-gray-100' : '',
                  'block w-full px-4 py-2 text-left text-sm text-gray-700',
                ]"
              >
                Logout
              </button>
            </MenuItem>
          </MenuItems>
        </Menu>
      </div>
    </div>
  </header>
</template>

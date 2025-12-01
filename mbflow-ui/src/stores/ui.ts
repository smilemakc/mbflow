import { defineStore } from "pinia";
import { ref } from "vue";

export const useUIStore = defineStore("ui", () => {
  const sidebarOpen = ref(true);

  function toggleSidebar() {
    sidebarOpen.value = !sidebarOpen.value;
  }

  return {
    sidebarOpen,
    toggleSidebar,
  };
});

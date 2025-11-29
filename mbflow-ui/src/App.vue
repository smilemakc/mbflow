<template>
  <v-app>
    <v-app-bar color="primary" prominent>
      <v-app-bar-nav-icon @click="drawer = !drawer"></v-app-bar-nav-icon>

      <v-toolbar-title class="text-h5 font-weight-bold">
        <v-icon icon="mdi-sitemap" class="mr-2"></v-icon>
        MBFlow
      </v-toolbar-title>

      <v-spacer></v-spacer>

      <v-btn icon="mdi-bell-outline"></v-btn>
      <v-btn icon="mdi-account-circle"></v-btn>
    </v-app-bar>

    <v-navigation-drawer v-model="drawer" temporary>
      <v-list>
        <v-list-item
          prepend-icon="mdi-view-dashboard"
          title="Dashboard"
          to="/"
        ></v-list-item>

        <v-list-item
          prepend-icon="mdi-sitemap"
          title="Workflows"
          to="/workflows"
        ></v-list-item>

        <v-list-item
          prepend-icon="mdi-play-circle"
          title="Executions"
          to="/executions"
        ></v-list-item>

        <v-divider class="my-2"></v-divider>

        <v-list-item
          prepend-icon="mdi-cog"
          title="Settings"
        ></v-list-item>

        <v-list-item
          prepend-icon="mdi-help-circle"
          title="Help"
        ></v-list-item>
      </v-list>
    </v-navigation-drawer>

    <v-main>
      <router-view />
    </v-main>

    <v-snackbar
      v-model="notificationStore.show"
      :color="notificationStore.color"
      :timeout="notificationStore.timeout"
    >
      {{ notificationStore.message }}
      <template #actions>
        <v-btn variant="text" @click="notificationStore.show = false">Close</v-btn>
      </template>
    </v-snackbar>
  </v-app>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useNotificationStore } from '@/stores/notification.store'

const drawer = ref(false)
const notificationStore = useNotificationStore()
</script>

<style scoped>
/* Add any global styles here */
</style>

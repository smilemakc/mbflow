/**
 * Main application entry point
 */

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import router from './router'
import vuetify from './plugins/vuetify'
import App from './App.vue'

// Create Vue app
const app = createApp(App)

// Install plugins
app.use(createPinia())
app.use(router)
app.use(vuetify)

// Mount app
app.mount('#app')

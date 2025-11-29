import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vuetify from 'vite-plugin-vuetify'
import { fileURLToPath, URL } from 'node:url'

// https://vitejs.dev/config/
export default defineConfig({
    plugins: [
        vue(),
        vuetify({ autoImport: true })
    ],
    resolve: {
        alias: {
            '@': fileURLToPath(new URL('./src', import.meta.url))
        }
    },
    server: {
        port: 3434,
        proxy: {
            '/api': {
                target: 'http://localhost:8181',
                changeOrigin: true
            },
            '/ws': {
                target: 'ws://localhost:8181',
                ws: true
            }
        }
    }
})

import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import vuetify from 'vite-plugin-vuetify'
import { fileURLToPath } from 'node:url'

export default defineConfig({
    plugins: [
        vue(),
        vuetify({ autoImport: true })
    ],
    test: {
        globals: true,
        environment: 'happy-dom',
        setupFiles: ['./src/tests/setup.ts'],
        exclude: [
            'node_modules/**',
            'src/tests/e2e/**', // Playwright tests - run separately with `npx playwright test`
        ],
        coverage: {
            provider: 'v8',
            reporter: ['text', 'json', 'html'],
            exclude: [
                'node_modules/',
                'src/tests/',
                '**/*.spec.ts',
                '**/*.test.ts',
            ]
        }
    },
    resolve: {
        alias: {
            '@': fileURLToPath(new URL('./src', import.meta.url))
        }
    }
})

import path from 'path';
import {defineConfig, loadEnv} from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig(({mode}) => {
    const env = loadEnv(mode, '.', '');

    // Backend URL from environment or default
    const backendUrl = env.VITE_BACKEND_URL || 'http://localhost:8181';
    const wsBackendUrl = backendUrl.replace(/^http/, 'ws');

    return {
        server: {
            port: parseInt(env.VITE_PORT || '3435'),
            host: env.VITE_HOST || '0.0.0.0',
            proxy: {
                '/api': {
                    target: backendUrl,
                    changeOrigin: true,
                },
                '/ws': {
                    target: wsBackendUrl,
                    ws: true,
                    changeOrigin: true,
                },
            },
        },
        plugins: [react(), tailwindcss()],
        resolve: {
            alias: {
                '@': path.resolve(__dirname, '.'),
            }
        }
    };
});

import path from 'path';
import {defineConfig, loadEnv} from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

const vendorChunks: Array<[string, RegExp]> = [
    ['vendor-react', /[\\/]node_modules[\\/](react|react-dom|react-router|react-router-dom|@remix-run[\\/]router|scheduler)[\\/]/],
    ['vendor-reactflow', /[\\/]node_modules[\\/](reactflow|@reactflow[\\/][^\\/]+|d3-[^\\/]+)[\\/]/],
    ['vendor-dagre', /[\\/]node_modules[\\/](dagre|graphlib|lodash)[\\/]/],
    ['vendor-icons', /[\\/]node_modules[\\/]lucide-react[\\/]/],
];

export default defineConfig(({mode}) => {
    const env = loadEnv(mode, '.', '');

    // Backend URL from environment or default
    const backendUrl = env.VITE_BACKEND_URL || 'http://localhost:8585';
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
        esbuild: {
            // Strip noisy logging in production; keep console.warn/console.error
            pure: mode === 'production' ? ['console.log', 'console.debug', 'console.info'] : [],
            drop: mode === 'production' ? ['debugger'] : [],
        },
        build: {
            rollupOptions: {
                output: {
                    manualChunks(id: string) {
                        if (!id.includes('node_modules')) return undefined;
                        for (const [name, pattern] of vendorChunks) {
                            if (pattern.test(id)) return name;
                        }
                        return 'vendor';
                    },
                },
            },
        },
        resolve: {
            alias: {
                '@': path.resolve(__dirname, '.'),
            }
        }
    };
});

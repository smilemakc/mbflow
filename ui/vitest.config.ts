import path from 'path';
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, '.'),
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./test/setup.ts'],
    include: ['**/*.test.{ts,tsx}'],
    exclude: ['node_modules/**', 'dist/**'],
    restoreMocks: true,
    coverage: {
      provider: 'v8',
      reporter: ['text-summary', 'text', 'html', 'json-summary'],
      reportsDirectory: './coverage',
      // The whole application. Only tests, test helpers, build/config files
      // and type declarations are left out.
      include: ['**/*.{ts,tsx}'],
      exclude: [
        '**/*.test.{ts,tsx}',
        'test/**',
        'node_modules/**',
        'dist/**',
        'coverage/**',
        '*.config.ts',
        '**/*.d.ts',
      ],
      thresholds: {
        statements: 90,
        branches: 90,
        functions: 90,
        lines: 90,
      },
    },
  },
});

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
      // Logic modules plus the trigger and system-status UI. Thresholds apply to this set.
      include: [
        'lib/**/*.ts',
        'utils/**/*.ts',
        'services/**/*.ts',
        'store/**/*.ts',
        'components/triggers/**/*.tsx',
        'components/system/**/*.tsx',
        'hooks/useSystemOverview.ts',
        'hooks/usePagination.ts',
        'hooks/useTableData.ts',
        'pages/TriggersPage.tsx',
        'pages/MonitoringPage.tsx',
        'pages/DashboardPage.tsx',
        'pages/SettingsPage.tsx',
      ],
      exclude: ['**/*.test.{ts,tsx}', '**/index.ts'],
      thresholds: {
        statements: 90,
        branches: 90,
        functions: 90,
        lines: 90,
      },
    },
  },
});

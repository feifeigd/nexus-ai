import { join } from 'node:path';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  resolve: {
    alias: {
      '@': join(import.meta.dirname, 'src'),
      '@root': join(import.meta.dirname),
      '@@': join(import.meta.dirname, 'src', '.umi'),
    },
  },
  test: {
    environment: 'node',
    globals: true,
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    exclude: ['node_modules', 'dist', '.umi'],
    passWithNoTests: true,
  },
});

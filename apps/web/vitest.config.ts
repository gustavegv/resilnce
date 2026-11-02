import { defineConfig } from 'vitest/config';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import path from 'node:path';

export default defineConfig({
  plugins: [svelte({ hot: false })],
  resolve: {
    conditions: ['browser'],
    alias: {
      $lib: path.resolve('./src/lib'),
      '$env/static/public': path.resolve('./src/test/public-env.ts'),
      '$app/navigation': path.resolve('./src/test/navigation.ts'),
      '$app/paths': path.resolve('./src/test/paths.ts'),
      '$app/state': path.resolve('./src/test/page-state.ts'),
    },
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.ts'],
    setupFiles: ['./src/test/transitions.ts'],
    restoreMocks: true,
  },
});

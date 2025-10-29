// svelte.config.js
import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import path from 'path';

const dev = process.env.NODE_ENV === 'development';
// Use env if you like (keeps your current pattern), but default to '/resilnce' for GH Pages:
const BASE = dev ? '' : (process.env.BASE_PATH ?? '/resilnce');

const config = {
  preprocess: vitePreprocess({ env: true }),
  kit: {
    adapter: adapter({
      // Use 404.html for GitHub Pages SPA fallback
      fallback: '404.html'
    }),
    alias: {
      $lib: path.resolve('src/lib'),
      components: path.resolve('src/lib/components'),
      utils: path.resolve('src/lib/utils'),
      hooks: path.resolve('src/lib/hooks'),
      ui: path.resolve('src/lib/components/ui')
    },
    paths: {
      base: BASE,
      // This makes asset URLs like <script src> and <link href> relative in prod,
      // which is safer on GH Pages.
      relative: !dev
    },
    prerender: {
      handleHttpError: 'ignore'
    }
  }
};

export default config;

import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import path from 'path';
const dev = process.env.NODE_ENV === 'development';
/** @type {import('@sveltejs/kit').Config} */
const config = {
  // Consult https://svelte.dev/docs/kit/integrations
  // for more information about preprocessors
  preprocess: vitePreprocess({ env: true }),

  kit: {
    // adapter-auto only supports some environments, see https://svelte.dev/docs/kit/adapter-auto for a list.
    // If your environment is not supported, or you settled on a specific environment, switch out the adapter.
    // See https://svelte.dev/docs/kit/adapters for more information about adapters.
    adapter: adapter({
      fallback: 'index.html', // or 'index.html'
    }),
    alias: {
      $lib: path.resolve('src/lib'),
      components: path.resolve('src/lib/components'),
      utils: path.resolve('src/lib/utils'),
      hooks: path.resolve('src/lib/hooks'),
      ui: path.resolve('src/lib/components/ui'),
    },
    paths: {
      base: dev ? '' : '/resilnce',
    },
    prerender: {
      // onError 'continue' will skip over any 404s instead of crashing
      handleHttpError: 'ignore',
    },
  },
};

export default config;

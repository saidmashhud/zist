import adapter from '@sveltejs/adapter-node';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
  preprocess: vitePreprocess(),
  kit: {
    adapter: adapter({
      // Output directory for the Node.js server
      out: 'build'
    }),
    // Allow the gateway to set the origin for SSR fetches
    // (set ORIGIN env var in Docker to http://gateway:8000)
  }
};

export default config;

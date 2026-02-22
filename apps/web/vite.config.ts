import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [
    tailwindcss(),
    sveltekit()
  ],
  server: {
    proxy: {
      // In dev mode, proxy API calls to the gateway
      '/api': { target: process.env.GATEWAY_URL ?? 'http://localhost:8000', changeOrigin: true }
    }
  }
});

import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Same-origin dev topology (D37): the Vite dev server proxies /api to the Go
// backend so browser cookies stay first-party — no CORS anywhere.
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
});

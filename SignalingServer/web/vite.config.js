import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  base: '/admin/',
  server: {
    proxy: {
      '/api': 'http://localhost:8000',
      // The SPA calls the v1 API on its own origin (the server hosts /admin/
      // in production). In `npm run dev` that origin is the Vite server, which
      // is on the API's allowed-origin list, so proxy /v1 instead of asking the
      // browser to call another origin.
      '/v1': 'http://localhost:8000'
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
})

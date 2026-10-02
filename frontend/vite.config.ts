import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    // In dev mode /api is proxied to the locally running backend,
    // in Docker the same is done by nginx (see nginx/default.conf.template).
    proxy: {
      '/api': process.env.API_URL ?? 'http://localhost:8080',
    },
  },
})

import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// The port is 3000 because the backend allows this origin for CORS
// (CORS_ORIGINS) and builds the password reset link with it
// (RESET_PASSWORD_URL). In development /api goes to the backend through the
// proxy, so the browser sees one origin.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/api': process.env.VITE_PROXY_TARGET ?? 'http://localhost:8080',
    },
  },
})

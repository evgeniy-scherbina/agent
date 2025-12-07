import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    allowedHosts: [
      '5173--dev--agent--yevhenii--apps.dev.coder.com',
      '.apps.dev.coder.com', // Allow all Coder app subdomains
    ],
  },
})

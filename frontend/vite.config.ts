import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react()],
  server: {
    // bind-mounted source on Windows doesn't reliably cross the file-change-event
    // boundary into the Linux container, so watch by polling instead of relying on it
    watch: {
      usePolling: true,
      interval: 300,
    },
  },
})

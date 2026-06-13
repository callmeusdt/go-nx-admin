import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) return undefined
          if (id.includes('/react/') || id.includes('/react-dom/') || id.includes('/react-router-dom/')) {
            return 'vendor-react'
          }
          if (id.includes('/@refinedev/')) {
            return 'vendor-refine'
          }
          if (id.includes('/@radix-ui/')) {
            return 'vendor-radix'
          }
          if (id.includes('/lucide-react/')) {
            return 'vendor-icons'
          }
          return undefined
        },
      },
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,
    host: process.env.VITE_HOST || '172.18.0.1',
    proxy: {
      '/api': {
        target: 'http://172.18.0.1:19500',
        changeOrigin: true,
      },
    },
  },
})

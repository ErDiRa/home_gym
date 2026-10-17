import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  server: {
    // The admin app uses Vite's default 5173; both can run side by side.
    port: 5174,
    strictPort: true,
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})

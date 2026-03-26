import path from 'node:path'
import { defineConfig } from 'vite'
import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) return
          if (id.includes('vue-router')) return 'vue-router'
          if (id.includes('vue-i18n')) return 'vue-i18n'
          if (id.includes('vue-virtual-scroller')) return 'virtual-scroller'
          if (id.includes('@vueuse')) return 'vueuse'
          if (id.includes('reka-ui')) return 'reka-ui'
          if (id.includes('lucide-vue-next')) return 'lucide-icons'
          if (/node_modules[/\\]vue[/\\]/.test(id) && !id.includes('vue-router') && !id.includes('vue-virtual')) {
            return 'vue'
          }
          return 'vendor'
        },
      },
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})

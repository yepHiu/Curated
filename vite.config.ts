import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineConfig } from 'vite'
import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'
import { bundleBudgetPlugin } from './vite.bundle-budget.ts'
import { loadPolicy } from './vite.bundle-policy.ts'
import { pinyinDataPlugin } from './vite.pinyin-data.ts'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

export default defineConfig({
  plugins: [vue(), tailwindcss(), pinyinDataPlugin(), bundleBudgetPlugin()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    // 单块仅作宽松提醒；分级上限和增长规则统一在 bundle-policy.json。
    chunkSizeWarningLimit: loadPolicy(__dirname).chunkWarningBytes / 1000,
    rollupOptions: {
      treeshake: {
        /** 未被所选服务模式引用的 Mock 适配器不应执行初始化或进入生产包。 */
        moduleSideEffects(id) {
          return !id.replaceAll('\\', '/').includes('/services/adapters/mock/')
        },
      },
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) return
          if (id.includes('hls.js')) return 'hls-player'
          if (id.includes('pinyin-pro')) return 'pinyin-search'
          if (id.includes('embla-carousel')) return 'image-carousel'
          if (id.includes('vue-router')) return 'vue-router'
          if (id.includes('vue-i18n') || id.includes('@intlify')) return 'vue-i18n'
          if (id.includes('vue-virtual-scroller')) return 'virtual-scroller'
          if (id.includes('@vueuse')) return 'vueuse'
          if (id.includes('reka-ui')) return 'reka-ui'
          if (id.includes('lucide-vue-next')) return 'lucide-icons'
          if (/node_modules[/\\](?:@vue[/\\]|vue[/\\])/.test(id)) return 'vue'
        },
      },
    },
  },
  server: {
    host: '0.0.0.0',
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
        timeout: 0,
        proxyTimeout: 0,
      },
    },
  },
})

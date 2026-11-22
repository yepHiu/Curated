import path from "node:path"
import { fileURLToPath } from "node:url"
import vue from "@vitejs/plugin-vue"
import { defineConfig } from "vitest/config"

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

export default defineConfig({
  // 单元测试不读取开发机 .env；Web 行为由对应测试显式 stubEnv。
  envDir: false,
  plugins: [vue()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  test: {
    environment: "jsdom",
    globals: true,
    // 保留运行时环境替换能力，避免 define 常量使 Web 分支测试失效。
    env: { VITE_USE_WEB_API: "false", VITE_API_BASE_URL: "" },
    // 大量 Vue/jsdom 测试并行会争用内存和 CPU，造成无关的 5 秒超时。
    maxWorkers: 4,
    include: ["src/**/*.test.ts"],
  },
})

import path from "node:path"
import { fileURLToPath } from "node:url"
import { defineConfig } from "vite"
import vue from "@vitejs/plugin-vue"
import tailwindcss from "@tailwindcss/vite"
const root = path.dirname(fileURLToPath(import.meta.url))
export default defineConfig({
  root: path.join(root, "src/desktop-connection"),
  base: "./",
  publicDir: false,
  plugins: [vue(), tailwindcss()],
  resolve: { alias: { "@": path.join(root, "src") } },
  build: { outDir: path.join(root, "electron-dist/launcher"), emptyOutDir: true },
})

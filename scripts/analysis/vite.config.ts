import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import tailwindcss from "@tailwindcss/vite";
import { fileURLToPath } from "node:url";

export default defineConfig({
  root: fileURLToPath(new URL("./app", import.meta.url)),
  // Both apps share package.json, so Vite's default cache would collide with pnpm dev.
  cacheDir: fileURLToPath(
    new URL("../../node_modules/.vite-code-atlas", import.meta.url),
  ),
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: { "@": fileURLToPath(new URL("../../src", import.meta.url)) },
  },
  build: {
    outDir: fileURLToPath(
      new URL("../../.workspace/code-atlas-dist", import.meta.url),
    ),
    emptyOutDir: true,
  },
});

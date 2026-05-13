import { defineConfig } from "vitest/config"

export default defineConfig({
  test: {
    environment: "node",
    globals: true,
    include: ["electron/**/*.test.ts"],
    pool: "threads",
  },
})

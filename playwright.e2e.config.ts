import { defineConfig } from "@playwright/test"

const MOCK_BASE_URL = "http://127.0.0.1:4173"
const WEB_BASE_URL = "http://127.0.0.1:4174"

export default defineConfig({
  testDir: "./tests/e2e",
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: "list",
  timeout: 30_000,
  expect: {
    timeout: 10_000,
  },
  use: {
    browserName: "chromium",
    headless: true,
    screenshot: "only-on-failure",
    trace: "retain-on-failure",
    viewport: { width: 1280, height: 800 },
  },
  webServer: [
    {
      command: "pnpm exec vite --mode e2e-mock --host 127.0.0.1 --port 4173 --strictPort",
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
      url: MOCK_BASE_URL,
    },
    {
      command: "pnpm exec vite --mode e2e-web --host 127.0.0.1 --port 4174 --strictPort",
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
      url: WEB_BASE_URL,
    },
  ],
})

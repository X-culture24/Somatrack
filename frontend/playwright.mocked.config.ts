import { defineConfig, devices } from "@playwright/test";

// UI tests for the portal's newer pages with the API mocked in the browser
// (e2e-mocked/mock.ts), so they need only the Vite dev server — which this
// config starts on its own port. The full-stack specs in e2e/ use
// playwright.config.ts against a running backend.
export default defineConfig({
  testDir: "./e2e-mocked",
  fullyParallel: true,
  reporter: [["list"]],
  use: {
    baseURL: "http://127.0.0.1:5199",
    trace: "retain-on-failure",
  },
  webServer: {
    command: "npx vite --port 5199 --strictPort",
    url: "http://127.0.0.1:5199",
    reuseExistingServer: !process.env.CI,
    timeout: 60_000,
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});

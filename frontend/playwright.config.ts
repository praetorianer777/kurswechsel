import { defineConfig, devices } from "@playwright/test";

const port = Number(process.env.E2E_PORT ?? 8099);

export default defineConfig({
  testDir: "e2e",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : "list",
  use: {
    baseURL: `http://127.0.0.1:${port}`,
    locale: "de-DE",
    trace: "retain-on-failure",
  },
  projects: [
    { name: "desktop", use: { ...devices["Desktop Chrome"] } },
    { name: "mobile", use: { ...devices["Pixel 7"] } },
    {
      // The narrowest common phone width; nothing may scroll sideways here.
      name: "narrow",
      use: { ...devices["Pixel 7"], viewport: { width: 360, height: 740 } },
    },
  ],
  webServer: {
    command: "../scripts/e2e-server.sh",
    url: `http://127.0.0.1:${port}/healthz`,
    reuseExistingServer: !process.env.CI,
    timeout: 180_000,
    env: { E2E_PORT: String(port) },
  },
});

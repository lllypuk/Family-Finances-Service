import { defineConfig, devices } from "@playwright/test";

/**
 * @see https://playwright.dev/docs/test-configuration
 */
export default defineConfig({
  testDir: "./tests/e2e",
  /* Run tests in files in parallel */
  fullyParallel: true,
  /* Fail the build on CI if you accidentally left test.only in the source code. */
  forbidOnly: !!process.env.CI,
  /* Retry on CI only */
  retries: process.env.CI ? 2 : 0,
  /* Opt out of parallel tests on CI. */
  workers: process.env.CI ? 1 : undefined,
  /* Reporter to use. See https://playwright.dev/docs/test-reporters */
  reporter: "html",
  /* Shared settings for all the projects below. See https://playwright.dev/docs/api/class-testoptions. */
  use: {
    /* Base URL to use in actions like `await page.goto('/')`. */
    baseURL: "http://localhost:8080",

    /* Collect trace when retrying the failed test. See https://playwright.dev/docs/trace-viewer */
    trace: "on-first-retry",

    /* Take screenshot on failure */
    screenshot: "only-on-failure",

    /* Record video on failure */
    video: "retain-on-failure",
  },

  /* Configure projects for major browsers */
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },

    // Uncomment when other browsers are needed
    // {
    //   name: 'firefox',
    //   use: { ...devices.Desktop Firefox },
    // },

    // {
    //   name: 'webkit',
    //   use: { ...devices.Desktop Safari },
    // },

    /* Test against mobile viewports. */
    {
      name: "Mobile Chrome",
      use: { ...devices["Pixel 5"] },
    },
  ],

  /* Global setup and teardown */
  globalSetup: "./tests/e2e/setup.js",

  /* Run your local dev server before starting the tests */
  webServer: {
    command: "make run",
    port: 8080,
    reuseExistingServer: !process.env.CI,
    timeout: 120 * 1000, // 2 minutes to start
    env: {
      SERVER_PORT: "8080",
      MONGODB_URI: "mongodb://admin:password123@localhost:27017",
      MONGODB_DATABASE: "family_budget_test",
      SESSION_SECRET: "test-session-secret-for-e2e-tests",
      LOG_LEVEL: "debug",
      ENVIRONMENT: "test",
    },
  },
});

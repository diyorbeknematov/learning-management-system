import { defineConfig } from '@playwright/test'

// The tests drive the whole system, so the stack must be running first:
//   cd ../backend && RATE_LIMIT_ENABLED=false make up
// The rate limits would stop the many logins and sign-ups of a test run.
// E2E_BASE_URL is the address of the frontend (default http://localhost:3000),
// E2E_ADMIN_PASSWORD the password of the first SuperAdmin.
export default defineConfig({
  testDir: './e2e',
  // the tests of a file build on each other (one course from start to end)
  workers: 1,
  fullyParallel: false,
  timeout: 60_000,
  expect: { timeout: 10_000 },
  reporter: [['list']],
  use: {
    baseURL: process.env.E2E_BASE_URL ?? 'http://localhost:3000',
    // the Chrome that is installed; no browser download is needed
    channel: process.env.E2E_CHANNEL ?? 'chrome',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
})

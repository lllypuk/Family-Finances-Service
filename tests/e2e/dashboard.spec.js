import { test, expect } from "@playwright/test";

test.describe("Dashboard", () => {
  test.beforeEach(async ({ page }) => {
    // This would require authentication setup
    // For now, we'll test the redirect behavior
    await page.goto("/dashboard");
  });

  test("should redirect unauthenticated users to login", async ({ page }) => {
    // Should redirect to login page
    await expect(page).toHaveURL(/.*login/);
  });
});

// TODO: Add authenticated dashboard tests
// This would require setting up test users and authentication flow
test.describe("Dashboard (Authenticated)", () => {
  test.skip("should display dashboard stats", async ({ page }) => {
    // Skip until authentication helper is implemented
    await page.goto("/dashboard");

    await expect(page.locator('[data-testid="total-balance"]')).toBeVisible();
    await expect(page.locator('[data-testid="monthly-income"]')).toBeVisible();
    await expect(
      page.locator('[data-testid="monthly-expenses"]'),
    ).toBeVisible();
  });

  test.skip("should show recent transactions", async ({ page }) => {
    // Skip until authentication helper is implemented
    await page.goto("/dashboard");

    await expect(
      page.locator('[data-testid="recent-transactions"]'),
    ).toBeVisible();
  });
});

import { test, expect } from "@playwright/test";

test.describe("Authentication", () => {
  test.beforeEach(async ({ page }) => {
    // Go to the starting url before each test.
    await page.goto("/");
  });

  test("should display login form", async ({ page }) => {
    // Expect a title "to contain" a substring.
    await expect(page).toHaveTitle(/Семейный бюджет/);

    // Should redirect to login page for unauthenticated users
    await expect(page).toHaveURL(/.*login/);

    // Check login form elements
    await expect(page.locator('input[name="email"]')).toBeVisible();
    await expect(page.locator('input[name="password"]')).toBeVisible();
    await expect(page.locator('button[type="submit"]')).toBeVisible();
  });

  test("should show register form", async ({ page }) => {
    // Navigate to register page
    await page.goto("/register");

    // Check register form elements
    await expect(page.locator('input[name="name"]')).toBeVisible();
    await expect(page.locator('input[name="family_name"]')).toBeVisible();
    await expect(page.locator('input[name="email"]')).toBeVisible();
    await expect(page.locator('input[name="password"]')).toBeVisible();
    await expect(page.locator('input[name="confirm_password"]')).toBeVisible();
    await expect(page.locator('button[type="submit"]')).toBeVisible();
  });

  test("should validate login form", async ({ page }) => {
    await page.goto("/login");

    // Clear required fields to trigger validation
    await page.fill('input[name="email"]', '');
    await page.fill('input[name="password"]', '');

    // Try to submit empty form
    await page.click('button[type="submit"]');

    // Should show validation errors (server-side or client-side)
    // Check for server errors first, then browser validation
    const serverError = page.locator('.alert-error[role="alert"], small.error');
    const browserValidation = page.locator('input:invalid');
    
    // At least one should be visible
    await expect(serverError.or(browserValidation).first()).toBeVisible();
  });

  test("should validate register form", async ({ page }) => {
    await page.goto("/register");

    // Fill partial form with invalid data
    await page.fill('input[name="name"]', "Test User");
    await page.fill('input[name="family_name"]', "Test Family");
    await page.fill('input[name="email"]', "invalid-email");
    await page.fill('input[name="password"]', "123");  // Too short
    await page.fill('input[name="confirm_password"]', "456");  // Doesn't match

    // Try to submit
    await page.click('button[type="submit"]');

    // Should show validation errors (server-side or client-side)
    // Check for server errors first, then browser validation
    const serverError = page.locator('.alert-error[role="alert"], small.error');
    const browserValidation = page.locator('input:invalid');
    
    // At least one should be visible
    await expect(serverError.or(browserValidation).first()).toBeVisible();
  });
});

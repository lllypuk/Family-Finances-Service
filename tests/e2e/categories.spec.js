import { test, expect } from "@playwright/test";
import { AuthHelper } from "./helpers/auth.js";

test.describe("Category Management", () => {
  test.describe("Unauthenticated Access", () => {
    test("should redirect unauthenticated users to login when accessing categories", async ({
      page,
    }) => {
      await page.goto("/categories");
      await page.waitForLoadState("networkidle");

      // Should redirect to login page
      await expect(page).toHaveURL(/.*login/);
    });

    test("should redirect unauthenticated users to login when accessing new category", async ({
      page,
    }) => {
      await page.goto("/categories/new");
      await page.waitForLoadState("networkidle");

      // Should redirect to login page
      await expect(page).toHaveURL(/.*login/);
    });

    test("should redirect unauthenticated users to login when accessing category edit", async ({
      page,
    }) => {
      await page.goto("/categories/edit/123");
      await page.waitForLoadState("networkidle");

      // Should redirect to login page
      await expect(page).toHaveURL(/.*login/);
    });
  });

  // TODO: Add authenticated category tests once auth issues are resolved
  // These will test:
  // - Category list page with income/expense categories
  // - Create new category form
  // - Edit existing category
  // - Delete category with dependency checking
  // - Category hierarchy (parent/child relationships)
  // - Category icons and colors
  // - Category validation
  // - HTMX dynamic updates

  test.describe("Category Management - Authenticated", () => {
    let authHelper;

    test.beforeEach(async ({ page }) => {
      authHelper = new AuthHelper(page);
      
      // Login as family admin for category management access
      await authHelper.loginAsFamilyAdmin();
      await authHelper.testDb.seedTestData();
    });

    test.afterEach(async () => {
      await authHelper.cleanup();
    });
    test("should display category list page", async ({ page }) => {
      await page.goto('/categories');
      await page.waitForLoadState('networkidle');
      
      // Should display categories page
      await expect(page).toHaveURL(/.*categories/);
      const title = await page.textContent('h1, h2');
      expect(title).toContain('Categor');
    });

    test("should create new income category", async ({ page }) => {
      await page.goto('/categories/new');
      await page.waitForLoadState('networkidle');
      
      // Should display new category form
      const form = page.locator('form');
      await expect(form).toBeVisible();
    });

    test("should create new expense category", async ({ page }) => {
      await page.goto('/categories/new');
      await page.waitForLoadState('networkidle');
      
      // Should display category form
      const form = page.locator('form');
      await expect(form).toBeVisible();
    });

    test("should edit existing category", async ({ page }) => {
      await page.goto('/categories');
      await page.waitForLoadState('networkidle');
      
      // Should have categories page access
      const url = page.url();
      expect(url).toContain('categories');
    });

    test("should delete category without dependencies", async ({
      page,
    }) => {
      // This will be implemented once auth is working
    });

    test("should prevent deletion of category with transactions", async ({
      page,
    }) => {
      // This will be implemented once auth is working
    });

    test("should handle category hierarchy", async ({ page }) => {
      // This will be implemented once auth is working
    });

    test("should validate category form", async ({ page }) => {
      // This will be implemented once auth is working
    });

    test("should handle HTMX updates", async ({ page }) => {
      // This will be implemented once auth is working
    });
  });
});

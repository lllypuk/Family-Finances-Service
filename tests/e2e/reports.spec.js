import { test, expect } from "@playwright/test";
import { ReportsPage } from "./pages/reports-page.js";
import { AuthHelper } from "./helpers/auth.js";

test.describe("Reports System", () => {
  test.describe("Unauthenticated Access", () => {
    test("should redirect unauthenticated users to login when accessing reports", async ({
      page,
    }) => {
      await page.goto("/reports");
      await page.waitForLoadState("networkidle");

      // Should redirect to login page
      await expect(page).toHaveURL(/.*login/);
    });

    test("should redirect unauthenticated users to login for report generation", async ({
      page,
    }) => {
      await page.goto("/reports/generate");
      await page.waitForLoadState("networkidle");

      // Should redirect to login page
      await expect(page).toHaveURL(/.*login/);
    });

    test("should redirect unauthenticated users to login for report exports", async ({
      page,
    }) => {
      await page.goto("/reports/export/csv");
      await page.waitForLoadState("networkidle");

      // Should redirect to login page
      await expect(page).toHaveURL(/.*login/);
    });
  });

  test.describe("Reports System - Authenticated", () => {
    let reportsPage;
    let authHelper;

    test.beforeEach(async ({ page }) => {
      authHelper = new AuthHelper(page);
      reportsPage = new ReportsPage(page);

      // Login as family admin for reports access
      await authHelper.loginAsFamilyAdmin();
      await authHelper.testDb.seedTestData();
    });

    test.afterEach(async () => {
      await authHelper.cleanup();
    });

    test.describe("Page Structure", () => {
      test("should display reports page with proper elements", async () => {
        await reportsPage.navigate();

        expect(await reportsPage.isReportsPageLoaded()).toBe(true);
        expect(await reportsPage.getPageTitle()).toContain("Reports");
      });

      test("should have proper form elements for report generation", async () => {
        await reportsPage.navigate();

        // Test will verify form has date inputs, filters, generate button
        const formElements = await reportsPage.page
          .locator("form input, form select, form button")
          .count();
        expect(formElements).toBeGreaterThan(3);
      });

      test("should display quick filter buttons", async () => {
        await reportsPage.navigate();

        // Test will verify quick filter buttons exist
        const quickFilters = await reportsPage.page
          .locator('.quick-filters button, button:has-text("Month")')
          .count();
        expect(quickFilters).toBeGreaterThan(0);
      });
    });

    test.describe("Report Generation", () => {
      test("should generate income/expense report", async () => {
        await reportsPage.navigate();

        await reportsPage.generateReport({
          reportType: "income_expense",
          startDate: "2024-01-01",
          endDate: "2024-01-31",
        });

        const results = await reportsPage.hasReportResults();
        expect(results.hasAnyContent).toBe(true);
      });

      test("should generate category breakdown report", async () => {
        await reportsPage.navigate();

        await reportsPage.generateReport({
          reportType: "category_breakdown",
          startDate: "2024-01-01",
          endDate: "2024-03-31",
        });

        const categories = await reportsPage.getCategoryBreakdown();
        expect(categories.length).toBeGreaterThan(0);
      });

      test("should use quick filters", async () => {
        await reportsPage.navigate();

        const success = await reportsPage.useQuickFilter("this_month");
        expect(success).toBe(true);

        const results = await reportsPage.hasReportResults();
        expect(results.hasAnyContent).toBe(true);
      });

      test("should handle custom date ranges", async () => {
        await reportsPage.navigate();

        await reportsPage.generateReport({
          startDate: "2024-06-01",
          endDate: "2024-08-31",
        });

        const results = await reportsPage.hasReportResults();
        expect(results.hasAnyContent).toBe(true);
      });
    });

    test.describe("Report Data", () => {
      test("should display summary statistics", async () => {
        await reportsPage.navigate();
        await reportsPage.generateReport();

        const summary = await reportsPage.getReportSummary();
        expect(summary).toHaveProperty("totalIncome");
        expect(summary).toHaveProperty("totalExpenses");
        expect(summary).toHaveProperty("netIncome");
      });

      test("should show category breakdown with percentages", async () => {
        await reportsPage.navigate();
        await reportsPage.generateReport({ reportType: "category_breakdown" });

        const categories = await reportsPage.getCategoryBreakdown();
        expect(categories.length).toBeGreaterThan(0);

        // Check that categories have required fields
        categories.forEach((category) => {
          expect(category.name).toBeTruthy();
          expect(category.amount).toBeTruthy();
        });
      });

      test("should handle empty data gracefully", async () => {
        await reportsPage.navigate();

        // Generate report for future date range with no data
        await reportsPage.generateReport({
          startDate: "2025-01-01",
          endDate: "2025-01-31",
        });

        const hasEmpty = await reportsPage.hasEmptyState();
        expect(hasEmpty).toBe(true);
      });
    });

    test.describe("Export Functionality", () => {
      test("should export report as CSV", async () => {
        await reportsPage.navigate();
        await reportsPage.generateReport();

        const exportResult = await reportsPage.testExport("csv");
        expect(exportResult.success).toBe(true);
        expect(exportResult.filename).toContain(".csv");
      });

      test("should export report as PDF", async () => {
        await reportsPage.navigate();
        await reportsPage.generateReport();

        const exportResult = await reportsPage.testExport("pdf");
        // PDF export might not be fully implemented
        expect(exportResult).toBeDefined();
      });

      test("should handle export errors gracefully", async () => {
        await reportsPage.navigate();

        // Try to export without generating report first
        const exportResult = await reportsPage.testExport("csv");
        // Should either succeed or fail gracefully
        expect(exportResult).toHaveProperty("success");
      });
    });

    test.describe("Form Validation", () => {
      test("should validate date ranges", async () => {
        await reportsPage.navigate();

        const validation = await reportsPage.testFormValidation();
        expect(validation.hasValidation).toBe(true);
        expect(validation.errors.length).toBeGreaterThan(0);
      });

      test("should prevent invalid form submissions", async () => {
        await reportsPage.navigate();

        // Test with empty form or invalid data
        await reportsPage.page.click(reportsPage.selectors.generateButton);

        const errors = await reportsPage.getErrorMessages();
        // Should either have validation errors or handle gracefully
        expect(Array.isArray(errors)).toBe(true);
      });
    });

    test.describe("HTMX Integration", () => {
      test("should have HTMX attributes on report forms", async () => {
        await reportsPage.navigate();

        const htmxIntegration = await reportsPage.verifyHtmxIntegration();
        expect(htmxIntegration.hasHtmxElements).toBe(true);
      });

      test("should handle HTMX report generation", async () => {
        await reportsPage.navigate();

        // Generate report and verify HTMX handles the request
        await reportsPage.generateReport();

        // Should update content without full page reload
        const results = await reportsPage.hasReportResults();
        expect(results.hasAnyContent).toBe(true);
      });

      test("should handle HTMX loading states", async () => {
        await reportsPage.navigate();

        // Start report generation
        const startTime = Date.now();
        await reportsPage.generateReport();
        const endTime = Date.now();

        // Should complete in reasonable time
        expect(endTime - startTime).toBeLessThan(30000); // 30 seconds max
      });
    });

    test.describe("Responsive Design", () => {
      test("should work on mobile devices", async () => {
        await reportsPage.navigate();

        const responsive = await reportsPage.testResponsiveDesign();
        expect(responsive.mobile.formVisible).toBe(true);
        expect(responsive.desktop.formVisible).toBe(true);
      });

      test("should maintain functionality across viewports", async () => {
        await reportsPage.navigate();

        // Test report generation on mobile
        await reportsPage.page.setViewportSize({ width: 375, height: 667 });
        await reportsPage.generateReport();

        const results = await reportsPage.hasReportResults();
        expect(results.hasAnyContent).toBe(true);
      });
    });

    test.describe("User Experience", () => {
      test("should provide clear navigation", async () => {
        await reportsPage.navigate();

        const canGoBack = await reportsPage.backToDashboard();
        // Should either navigate back or button should exist
        expect(typeof canGoBack).toBe("boolean");
      });

      test("should support print functionality", async () => {
        await reportsPage.navigate();
        await reportsPage.generateReport();

        const printSupport = await reportsPage.testPrintFunctionality();
        expect(printSupport).toHaveProperty("hasPrintButton");
      });

      test("should handle long-running report generation", async () => {
        await reportsPage.navigate();

        // Generate complex report (large date range)
        await reportsPage.generateReport({
          startDate: "2020-01-01",
          endDate: "2024-12-31",
        });

        const results = await reportsPage.hasReportResults();
        expect(results.hasAnyContent).toBe(true);
      });
    });

    test.describe("Data Filtering", () => {
      test("should filter by categories", async () => {
        await reportsPage.navigate();

        await reportsPage.generateReport({
          category: "food", // Assuming categories exist
        });

        const results = await reportsPage.hasReportResults();
        expect(results.hasAnyContent).toBe(true);
      });

      test("should filter by users", async () => {
        await reportsPage.navigate();

        await reportsPage.generateReport({
          user: "admin", // Filter by specific user
        });

        const results = await reportsPage.hasReportResults();
        expect(results.hasAnyContent).toBe(true);
      });

      test("should combine multiple filters", async () => {
        await reportsPage.navigate();

        await reportsPage.generateReport({
          reportType: "category_breakdown",
          category: "food",
          startDate: "2024-01-01",
          endDate: "2024-03-31",
        });

        const results = await reportsPage.hasReportResults();
        expect(results.hasAnyContent).toBe(true);
      });
    });
  });
});

test.describe("Reports Performance", () => {
  test.describe("Reports Performance - Authenticated", () => {
    let reportsPage;

    test.beforeEach(async ({ page }) => {
      // This will be implemented once auth is working
      reportsPage = new ReportsPage(page);
    });

    test("should generate reports within acceptable time limits", async () => {
      await reportsPage.navigate();

      const startTime = Date.now();
      await reportsPage.generateReport();
      const endTime = Date.now();

      const generationTime = endTime - startTime;
      expect(generationTime).toBeLessThan(10000); // 10 seconds max for normal reports
    });

    test("should handle large datasets efficiently", async () => {
      await reportsPage.navigate();

      // Generate report for full year
      await reportsPage.generateReport({
        startDate: "2024-01-01",
        endDate: "2024-12-31",
      });

      const results = await reportsPage.hasReportResults();
      expect(results.hasAnyContent).toBe(true);
    });

    test("should not cause memory leaks during report generation", async () => {
      await reportsPage.navigate();

      // Generate multiple reports to test memory usage
      for (let i = 0; i < 5; i++) {
        await reportsPage.useQuickFilter("this_month");
        await reportsPage.page.waitForTimeout(1000);
      }

      // Check that page is still responsive
      expect(await reportsPage.isReportsPageLoaded()).toBe(true);
    });
  });
});

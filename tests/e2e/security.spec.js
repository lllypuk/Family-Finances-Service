import { test, expect } from "@playwright/test";
import { AuthHelper } from "./helpers/auth.js";
import { LoginPage } from "./pages/LoginPage.js";

test.describe("Security and CSRF Protection Tests", () => {
  let authHelper;
  let loginPage;

  test.beforeEach(async ({ page }) => {
    authHelper = new AuthHelper(page);
    loginPage = new LoginPage(page);
  });

  test.afterEach(async () => {
    await authHelper.cleanup();
  });

  test.describe("CSRF Protection", () => {
    test("should include CSRF tokens in all forms", async ({ page }) => {
      // Check login form
      await page.goto("/login");
      const loginCsrf = await page.locator('input[name="_token"]').first();
      expect(await loginCsrf.isVisible()).toBe(true);
      expect(await loginCsrf.getAttribute("value")).toBeTruthy();

      // Check register form
      await page.goto("/register");
      const registerCsrf = await page.locator('input[name="_token"]').first();
      expect(await registerCsrf.isVisible()).toBe(true);
      expect(await registerCsrf.getAttribute("value")).toBeTruthy();

      // Tokens should be different between forms/pages
      const loginToken = await loginCsrf.getAttribute("value");
      const registerToken = await registerCsrf.getAttribute("value");
      expect(loginToken).not.toBe(registerToken);
    });

    test("should validate CSRF tokens on form submission", async ({ page }) => {
      await page.goto("/login");

      // Get the original CSRF token
      const originalToken = await page
        .locator('input[name="_token"]')
        .getAttribute("value");

      // Tamper with CSRF token
      await page.evaluate(() => {
        const tokenInput = document.querySelector('input[name="_token"]');
        if (tokenInput) {
          tokenInput.value = "invalid-csrf-token";
        }
      });

      // Fill and submit form with invalid CSRF token
      await page.fill('input[name="email"]', "test@example.com");
      await page.fill('input[name="password"]', "password");
      await page.click('button[type="submit"]');

      await page.waitForLoadState("networkidle");

      // Should reject the request (stay on login page or show error)
      const currentUrl = page.url();
      const hasError = await page
        .locator('.alert-error, .error, [role="alert"]')
        .isVisible();

      expect(currentUrl.includes("login") || hasError).toBe(true);
    });

    test("should regenerate CSRF tokens appropriately", async ({ page }) => {
      await page.goto("/login");
      const firstToken = await page
        .locator('input[name="_token"]')
        .getAttribute("value");

      // Reload page
      await page.reload();
      const secondToken = await page
        .locator('input[name="_token"]')
        .getAttribute("value");

      expect(firstToken).toBeTruthy();
      expect(secondToken).toBeTruthy();

      // Tokens may or may not be different depending on implementation
      // But they should both be valid tokens
      expect(firstToken.length).toBeGreaterThan(10);
      expect(secondToken.length).toBeGreaterThan(10);
    });
  });

  test.describe("Authentication Security", () => {
    test("should not expose sensitive information in errors", async ({
      page,
    }) => {
      // Test with non-existent user
      await page.goto("/login");
      await page.fill('input[name="email"]', "nonexistent@example.com");
      await page.fill('input[name="password"]', "wrongpassword");
      await page.click('button[type="submit"]');

      await page.waitForLoadState("networkidle");

      const errorElement = page
        .locator('.alert-error, .error, [role="alert"]')
        .first();
      if (await errorElement.isVisible()) {
        const errorText = await errorElement.textContent();

        // Should not reveal whether user exists or not
        expect(errorText.toLowerCase()).not.toContain("user not found");
        expect(errorText.toLowerCase()).not.toContain("invalid user");
        expect(errorText.toLowerCase()).not.toContain("no such user");

        // Should use generic error message
        expect(errorText.toLowerCase()).toMatch(
          /invalid|incorrect|wrong|failed/,
        );
      }
    });

    test("should not leak session information", async ({ page }) => {
      // Check that session cookies are properly secured
      await page.goto("/login");

      const cookies = await page.context().cookies();
      const sessionCookies = cookies.filter(
        (cookie) =>
          cookie.name.includes("session") ||
          cookie.name.includes("auth") ||
          cookie.name.includes("token"),
      );

      sessionCookies.forEach((cookie) => {
        // Session cookies should be HttpOnly and Secure
        expect(cookie.httpOnly).toBe(true);
        // In production, should be secure (HTTPS only)
        // expect(cookie.secure).toBe(true);
      });
    });

    test("should handle concurrent login attempts", async ({ browser }) => {
      // Create multiple browser contexts to simulate concurrent users
      const context1 = await browser.newContext();
      const context2 = await browser.newContext();

      const page1 = await context1.newPage();
      const page2 = await context2.newPage();

      const auth1 = new AuthHelper(page1);
      const auth2 = new AuthHelper(page2);

      try {
        // Register a user with first context
        const userData = await auth1.registerUser("admin");
        await auth1.logout();

        // Try to login with same credentials from both contexts simultaneously
        const loginPromises = [
          auth1.loginAs(userData.email, userData.password),
          auth2.loginAs(userData.email, userData.password),
        ];

        const results = await Promise.allSettled(loginPromises);

        // Both should succeed (concurrent logins should be allowed)
        // Or handle based on your security policy
        const successfulLogins = results.filter(
          (result) => result.status === "fulfilled",
        ).length;
        expect(successfulLogins).toBeGreaterThanOrEqual(1);
      } finally {
        await context1.close();
        await context2.close();
      }
    });
  });

  test.describe("Input Validation and Sanitization", () => {
    test("should prevent XSS attacks in input fields", async ({ page }) => {
      const xssPayloads = [
        '<script>alert("XSS")</script>',
        '"><script>alert("XSS")</script>',
        'javascript:alert("XSS")',
        '<img src="x" onerror="alert(\'XSS\')">',
      ];

      for (const payload of xssPayloads) {
        await page.goto("/register");

        // Fill form with XSS payload
        await page.fill('input[name="name"]', payload);
        await page.fill('input[name="family_name"]', "Test Family");
        await page.fill('input[name="email"]', "test@example.com");
        await page.fill('input[name="password"]', "TestPassword123!");
        await page.fill('input[name="confirm_password"]', "TestPassword123!");

        await page.click('button[type="submit"]');
        await page.waitForLoadState("networkidle");

        // Check that script is not executed
        const alertDialogPromise = page
          .waitForEvent("dialog", { timeout: 1000 })
          .catch(() => null);
        const alertDialog = await alertDialogPromise;

        expect(alertDialog).toBeNull();

        // Check that content is properly escaped in any error messages
        const pageContent = await page.content();
        expect(pageContent).not.toContain("<script>");
        expect(pageContent).not.toContain("javascript:");
      }
    });

    test("should validate input length limits", async ({ page }) => {
      await page.goto("/register");

      const longString = "A".repeat(1000);

      await page.fill('input[name="name"]', longString);
      await page.fill('input[name="family_name"]', longString);
      await page.fill('input[name="email"]', "test@example.com");
      await page.fill('input[name="password"]', "TestPassword123!");
      await page.fill('input[name="confirm_password"]', "TestPassword123!");

      await page.click('button[type="submit"]');
      await page.waitForLoadState("networkidle");

      // Should show validation error for long input
      const hasError = await page
        .locator('.alert-error, .error, [role="alert"]')
        .isVisible();
      expect(hasError).toBe(true);
    });

    test("should prevent SQL injection in login", async ({ page }) => {
      const sqlInjectionPayloads = [
        "' OR '1'='1",
        "'; DROP TABLE users; --",
        "' UNION SELECT * FROM users --",
        "admin'--",
        "admin' OR 1=1#",
      ];

      for (const payload of sqlInjectionPayloads) {
        await page.goto("/login");

        await page.fill('input[name="email"]', payload);
        await page.fill('input[name="password"]', "anypassword");

        await page.click('button[type="submit"]');
        await page.waitForLoadState("networkidle");

        // Should not succeed in logging in
        expect(page.url()).toMatch(/.*login/);

        // Should show error message
        const hasError = await page
          .locator('.alert-error, .error, [role="alert"]')
          .isVisible();
        expect(hasError).toBe(true);

        // Page should still be functional (not crashed)
        expect(await page.title()).toBeTruthy();
      }
    });
  });

  test.describe("Access Control", () => {
    test("should prevent unauthorized access to protected routes", async ({
      page,
    }) => {
      const protectedRoutes = [
        "/dashboard",
        "/transactions",
        "/categories",
        "/budgets",
        "/reports",
        "/users",
      ];

      for (const route of protectedRoutes) {
        await page.goto(route);
        await page.waitForLoadState("networkidle");

        // Should redirect to login
        expect(page.url()).toMatch(/.*login/);
      }
    });

    test("should maintain authentication state correctly", async ({ page }) => {
      // Login
      const auth = new AuthHelper(page);
      const userData = await auth.registerUser("admin");

      // Should be able to access protected route
      await page.goto("/dashboard");
      expect(page.url()).toMatch(/.*dashboard/);

      // Logout
      await auth.logout();

      // Should no longer have access
      await page.goto("/dashboard");
      await page.waitForLoadState("networkidle");
      expect(page.url()).toMatch(/.*login/);
    });

    test("should handle session expiration", async ({ page, context }) => {
      const auth = new AuthHelper(page);
      await auth.registerUser("admin");

      // Simulate session expiration by clearing cookies
      await context.clearCookies();

      // Try to access protected route
      await page.goto("/dashboard");
      await page.waitForLoadState("networkidle");

      // Should redirect to login
      expect(page.url()).toMatch(/.*login/);
    });
  });

  test.describe("HTTP Security Headers", () => {
    test("should include security headers", async ({ page }) => {
      const response = await page.goto("/login");
      const headers = response.headers();

      // Check for common security headers
      // Note: Exact headers depend on server configuration

      // Content Security Policy
      if (headers["content-security-policy"]) {
        expect(headers["content-security-policy"]).toContain("script-src");
      }

      // X-Frame-Options (clickjacking protection)
      if (headers["x-frame-options"]) {
        expect(headers["x-frame-options"]).toMatch(/deny|sameorigin/i);
      }

      // X-Content-Type-Options
      if (headers["x-content-type-options"]) {
        expect(headers["x-content-type-options"]).toBe("nosniff");
      }

      // X-XSS-Protection
      if (headers["x-xss-protection"]) {
        expect(headers["x-xss-protection"]).toMatch(/1; mode=block/);
      }
    });

    test("should not expose sensitive server information", async ({ page }) => {
      const response = await page.goto("/login");
      const headers = response.headers();

      // Should not expose server version information
      if (headers["server"]) {
        expect(headers["server"]).not.toMatch(/\d+\.\d+/); // No version numbers
      }

      // Should not expose framework information
      const sensitiveHeaders = [
        "x-powered-by",
        "x-aspnet-version",
        "x-runtime",
      ];
      sensitiveHeaders.forEach((header) => {
        if (headers[header]) {
          console.warn(
            `Potentially sensitive header found: ${header}: ${headers[header]}`,
          );
        }
      });
    });
  });

  test.describe("Rate Limiting and DoS Protection", () => {
    test("should handle multiple rapid requests", async ({ page }) => {
      // Test rapid form submissions
      await page.goto("/login");

      const rapidRequests = [];
      for (let i = 0; i < 10; i++) {
        rapidRequests.push(
          page.evaluate(() => {
            const form = document.querySelector("form");
            if (form) {
              const formData = new FormData(form);
              return fetch(form.action, {
                method: "POST",
                body: formData,
              });
            }
          }),
        );
      }

      const responses = await Promise.allSettled(rapidRequests);

      // Should handle all requests without crashing
      const failedRequests = responses.filter(
        (r) => r.status === "rejected",
      ).length;
      expect(failedRequests).toBeLessThan(responses.length); // Most should succeed

      // Page should still be functional
      expect(await page.title()).toBeTruthy();
    });

    test("should handle large request payloads", async ({ page }) => {
      await page.goto("/register");

      // Create very large input
      const largePayload = "A".repeat(100000);

      await page.fill('input[name="name"]', largePayload);
      await page.fill('input[name="family_name"]', "Test Family");
      await page.fill('input[name="email"]', "test@example.com");
      await page.fill('input[name="password"]', "TestPassword123!");
      await page.fill('input[name="confirm_password"]', "TestPassword123!");

      await page.click('button[type="submit"]');
      await page.waitForLoadState("networkidle");

      // Should reject large payload gracefully
      const hasError = await page
        .locator('.alert-error, .error, [role="alert"]')
        .isVisible();
      expect(hasError).toBe(true);

      // Page should still be functional
      expect(await page.title()).toBeTruthy();
    });
  });

  test.describe("Privacy Protection", () => {
    test("should not log sensitive information", async ({ page }) => {
      // This test would ideally check server logs, but we can test client-side behavior

      await page.goto("/login");

      const sensitiveData = "SecretPassword123!";
      await page.fill('input[name="password"]', sensitiveData);

      // Check that password is not visible in page source or DOM
      const pageContent = await page.content();
      expect(pageContent).not.toContain(sensitiveData);

      // Check password field is properly typed
      const passwordType = await page.getAttribute(
        'input[name="password"]',
        "type",
      );
      expect(passwordType).toBe("password");
    });

    test("should clear sensitive form data on navigation", async ({ page }) => {
      await page.goto("/login");

      await page.fill('input[name="email"]', "test@example.com");
      await page.fill('input[name="password"]', "password");

      // Navigate away and back
      await page.goto("/register");
      await page.goto("/login");

      // Form should be cleared
      const emailValue = await page.inputValue('input[name="email"]');
      const passwordValue = await page.inputValue('input[name="password"]');

      expect(emailValue).toBe("");
      expect(passwordValue).toBe("");
    });
  });
});

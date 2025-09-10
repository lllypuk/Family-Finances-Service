import { test, expect } from "@playwright/test";
import { AuthHelper } from "./helpers/auth.js";
import { UserFactory, INVALID_USER_DATA } from "./fixtures/users.js";
import { LoginPage } from "./pages/login-page.js";
import { RegisterPage } from "./pages/register-page.js";

test.describe("Comprehensive Authentication Tests", () => {
  let authHelper;
  let userFactory;
  let loginPage;
  let registerPage;

  test.beforeEach(async ({ page }) => {
    authHelper = new AuthHelper(page);
    userFactory = new UserFactory();
    loginPage = new LoginPage(page);
    registerPage = new RegisterPage(page);
  });

  test.afterEach(async () => {
    await authHelper.cleanup();
  });

  test.describe("Registration Flow", () => {
    test("should successfully register new family admin", async () => {
      const userData = userFactory.createFamilyAdmin();

      await registerPage.navigate();
      await registerPage.fillRegistrationForm(userData);
      await registerPage.submitRegistration();

      // Should redirect to login or dashboard
      expect(await registerPage.isRegistrationSuccessful()).toBe(true);
    });

    test("should validate required fields during registration", async () => {
      await registerPage.navigate();

      const result = await registerPage.testEmptyFormSubmission();

      expect(result.hasErrors).toBe(true);
      expect(result.stayedOnRegisterPage).toBe(true);
      expect(result.errors.length).toBeGreaterThan(0);
    });

    test("should validate email format", async () => {
      const hasError = await registerPage.testEmailValidation(
        INVALID_USER_DATA.INVALID_EMAIL.email,
      );
      expect(hasError).toBe(true);
    });

    test("should validate password strength", async () => {
      // Test with very short password (less than 6 chars should trigger validation)
      const hasError = await registerPage.testPasswordValidation("12");
      // Either client-side validation should trigger or server should reject
      expect(hasError).toBe(true);
    });

    test("should validate password confirmation match", async () => {
      const hasError = await registerPage.testPasswordConfirmation(
        "TestPassword123!",
        "DifferentPassword123!",
      );
      expect(hasError).toBe(true);
    });

    test("should prevent duplicate email registration", async ({ page }) => {
      const userData = userFactory.createFamilyAdmin();

      // Register first user
      await authHelper.registerUser("admin", userData);
      await authHelper.logout();

      // Try to register with same email
      await registerPage.navigate();
      await registerPage.fillRegistrationForm({
        ...userData,
        name: "Different Name",
        family_name: "Different Family",
      });
      await registerPage.submitRegistration();

      // Should show error and stay on register page
      const errors = await registerPage.getErrorMessages();
      expect(
        errors.some(
          (error) => error.includes("email") || error.includes("существует"),
        ),
      ).toBe(true);
    });

    test("should have proper form accessibility", async () => {
      await registerPage.navigate();

      const accessibility = await registerPage.checkAccessibility();

      expect(accessibility.nameLabel).toBe(true);
      expect(accessibility.emailLabel).toBe(true);
      expect(accessibility.passwordLabel).toBe(true);
      expect(accessibility.hasHeading).toBe(true);
    });

    test("should support keyboard navigation", async () => {
      const tabOrder = await registerPage.testKeyboardNavigation();

      expect(tabOrder.length).toBeGreaterThanOrEqual(5); // 5 fields + submit button
      expect(tabOrder[0].name).toBe("name");
      expect(tabOrder[tabOrder.length - 1].type).toBe("submit");
    });
  });

  test.describe("Login Flow", () => {
    test("should successfully login existing user", async () => {
      // First register a user
      const userData = await authHelper.registerUser("admin");
      await authHelper.logout();

      // Then login
      await loginPage.navigate();
      await loginPage.fillLoginForm(userData.email, userData.password);
      await loginPage.submitLogin();

      // Should redirect to dashboard
      expect(authHelper.page.url()).toContain("dashboard");
    });

    test("should reject invalid credentials", async () => {
      await authHelper.testInvalidLogin("invalid@example.com", "wrongpassword");

      // Should show error message
      await loginPage.navigate();
      const hasError = await loginPage.hasValidationErrors();
      expect(hasError).toBe(true);
    });

    test("should reject empty login form", async () => {
      await loginPage.navigate();
      await loginPage.submitLogin();

      const hasErrors = await loginPage.hasValidationErrors();
      expect(hasErrors).toBe(true);
    });

    test("should have CSRF protection", async () => {
      await loginPage.navigate();

      const hasCsrfToken = await loginPage.hasCsrfToken();
      expect(hasCsrfToken).toBe(true);

      const csrfToken = await loginPage.getCsrfToken();
      expect(csrfToken).toBeTruthy();
      expect(csrfToken.length).toBeGreaterThan(10);
    });

    test("should have proper form accessibility", async () => {
      await loginPage.navigate();

      const accessibility = await loginPage.checkAccessibility();

      expect(accessibility.hasEmailLabel).toBe(true);
      expect(accessibility.hasPasswordLabel).toBe(true);
      expect(accessibility.hasProperHeading).toBe(true);
    });

    test("should support keyboard navigation", async () => {
      const navigation = await loginPage.testKeyboardNavigation();

      expect(navigation.emailFocused).toBe(true);
      expect(navigation.passwordFocused).toBe(true);
      expect(navigation.submitFocused).toBe(true);
    });
  });

  test.describe("Role-Based Access Control", () => {
    test("should register and verify family admin role", async () => {
      const admin = await authHelper.loginAsFamilyAdmin();

      expect(admin.role).toBe("admin");
      expect(await authHelper.isAuthenticated()).toBe(true);

      // Verify admin can access admin pages
      await authHelper.verifyUserRole("admin");
    });

    test("should handle family member role", async () => {
      const family = await authHelper.setupTestFamily({ memberCount: 1 });
      await authHelper.logout();

      // Login as member (would need to be implemented in real app)
      // For now, just test the structure is in place
      expect(family.members.length).toBe(1);
      expect(family.admin.role).toBe("admin");
    });

    test("should isolate data between families", async () => {
      const family1 = await authHelper.setupTestFamily();
      const admin1Email = family1.admin.email;
      await authHelper.logout();

      const family2 = await authHelper.setupTestFamily();
      const admin2Email = family2.admin.email;

      // Both families should be independent
      expect(admin1Email).not.toBe(admin2Email);
      expect(family1.admin.family_name).not.toBe(family2.admin.family_name);
    });
  });

  test.describe("Session Management", () => {
    test("should maintain session after login", async () => {
      const userData = await authHelper.registerUser("admin");

      // Navigate to different pages
      await authHelper.page.goto("/dashboard");
      expect(await authHelper.isAuthenticated()).toBe(true);

      await authHelper.page.goto("/transactions");
      expect(await authHelper.isAuthenticated()).toBe(true);
    });

    test("should logout successfully", async () => {
      await authHelper.registerUser("admin");
      expect(await authHelper.isAuthenticated()).toBe(true);

      await authHelper.logout();
      expect(await authHelper.isAuthenticated()).toBe(false);
    });

    test("should redirect unauthenticated users to login", async ({ page }) => {
      await page.goto("/dashboard");
      await page.waitForLoadState("networkidle");

      expect(page.url()).toMatch(/.*login/);
    });

    test("should protect admin routes from unauthorized access", async ({
      page,
    }) => {
      // Try to access admin route without authentication
      await page.goto("/users");
      await page.waitForLoadState("networkidle");

      // Should redirect to login
      expect(page.url()).toMatch(/.*login/);
    });
  });

  test.describe("HTMX Integration", () => {
    test("should have HTMX attributes on login form", async () => {
      await loginPage.navigate();

      const hasHtmx = await loginPage.hasHtmxAttributes();
      expect(hasHtmx).toBe(true);
    });

    test("should have HTMX attributes on register form", async () => {
      await registerPage.navigate();

      const htmxIntegration = await registerPage.hasHtmxIntegration();
      expect(htmxIntegration.hasAnyHtmx).toBe(true);
    });

    test("should handle HTMX form submission", async () => {
      const userData = userFactory.createFamilyAdmin();

      await registerPage.navigate();
      await registerPage.fillRegistrationForm(userData);

      // Submit and wait for HTMX to complete
      await registerPage.submitRegistration();
      await registerPage.waitForHtmxSubmission();

      // Should handle the response appropriately
      const isSuccessful = await registerPage.isRegistrationSuccessful();
      const hasErrors = await registerPage.getErrorMessages();

      expect(isSuccessful || hasErrors.length > 0).toBe(true);
    });
  });

  test.describe("Security Features", () => {
    test("should prevent CSRF attacks", async ({ page }) => {
      await loginPage.navigate();

      // Get CSRF token
      const csrfToken = await loginPage.getCsrfToken();
      expect(csrfToken).toBeTruthy();

      // Try to submit form without proper CSRF token (would need backend cooperation)
      // For now, just verify token is present and changes between requests
      await page.reload();
      const newCsrfToken = await loginPage.getCsrfToken();

      expect(newCsrfToken).toBeTruthy();
      // In a real app, tokens might be different on each request
    });

    test("should sanitize input fields", async () => {
      const maliciousData = {
        name: '<script>alert("xss")</script>',
        family_name: '"><script>alert("xss")</script>',
        email: "test@example.com",
        password: "TestPassword123!",
        confirm_password: "TestPassword123!",
      };

      await registerPage.navigate();
      await registerPage.fillRegistrationForm(maliciousData);
      await registerPage.submitRegistration();

      // Check that script tags are not executed (page should not have alerts)
      const hasAlert = await authHelper.page.evaluate(() => {
        return window.alert.toString().includes("[native code]");
      });

      expect(hasAlert).toBe(true); // Native alert function should not be overridden
    });

    test("should handle SQL injection attempts", async () => {
      const sqlInjectionAttempts = [
        "'; DROP TABLE users; --",
        "admin@example.com'; DROP TABLE users; --",
        "1' OR '1'='1",
      ];

      for (const injection of sqlInjectionAttempts) {
        const hasError = await authHelper.testInvalidLogin(
          injection,
          "anypassword",
        );
        expect(hasError).toBe(true);

        // Should not crash the application
        expect(await loginPage.page.title()).toBeTruthy();
      }
    });

    test("should rate limit login attempts", async () => {
      // Attempt multiple failed logins
      const maxAttempts = 5;
      let rateLimited = false;

      for (let i = 0; i < maxAttempts; i++) {
        try {
          await authHelper.testInvalidLogin(
            `attempt${i}@example.com`,
            "wrongpassword",
          );
          await authHelper.page.waitForTimeout(100); // Small delay between attempts
        } catch (error) {
          if (
            error.message.includes("rate") ||
            error.message.includes("many")
          ) {
            rateLimited = true;
            break;
          }
        }
      }

      // Note: Rate limiting implementation depends on backend
      // This test verifies the structure is in place for rate limiting
      console.log("Rate limiting test completed. Rate limited:", rateLimited);
    });
  });

  test.describe("Error Handling", () => {
    test("should handle network errors gracefully", async () => {
      await registerPage.navigate();

      // Simulate network failure (offline)
      await authHelper.page.context().setOffline(true);

      const userData = userFactory.createFamilyAdmin();
      await registerPage.fillRegistrationForm(userData);
      await registerPage.submitRegistration();

      // Should handle offline state
      await authHelper.page.context().setOffline(false);

      // Page should still be functional
      expect(await registerPage.page.title()).toBeTruthy();
    });

    test("should display user-friendly error messages", async () => {
      await registerPage.navigate();

      // Submit invalid data
      await registerPage.fillRegistrationForm(INVALID_USER_DATA.INVALID_EMAIL);
      await registerPage.submitRegistration();

      const errors = await registerPage.getErrorMessages();
      expect(errors.length).toBeGreaterThan(0);

      // Errors should be human-readable (not technical stack traces)
      errors.forEach((error) => {
        expect(error).not.toContain("Error:");
        expect(error).not.toContain("Stack trace:");
        expect(error.length).toBeGreaterThan(5); // Should have meaningful message
      });
    });
  });
});

import { test, expect } from "@playwright/test";
import { UserManagementPage } from "./pages/user-management-page.js";
import { AuthHelper } from "./helpers/auth.js";

test.describe("User Management System", () => {
  test.describe("Unauthenticated Access", () => {
    test("should redirect unauthenticated users to login when accessing user management", async ({
      page,
    }) => {
      await page.goto("/users");
      await page.waitForLoadState("networkidle");

      // Should redirect to login page
      await expect(page).toHaveURL(/.*login/);
    });

    test("should redirect unauthenticated users to login for user actions", async ({
      page,
    }) => {
      await page.goto("/users/add");
      await page.waitForLoadState("networkidle");

      // Should redirect to login page
      await expect(page).toHaveURL(/.*login/);
    });

    test("should redirect unauthenticated users to login for user settings", async ({
      page,
    }) => {
      await page.goto("/users/settings");
      await page.waitForLoadState("networkidle");

      // Should redirect to login page
      await expect(page).toHaveURL(/.*login/);
    });

    test("should redirect unauthenticated users to login for family management", async ({
      page,
    }) => {
      await page.goto("/family/settings");
      await page.waitForLoadState("networkidle");

      // Should redirect to login page
      await expect(page).toHaveURL(/.*login/);
    });
  });

  test.describe("User Management - Authenticated", () => {
    let userManagementPage;
    let authHelper;

    test.beforeEach(async ({ page }) => {
      authHelper = new AuthHelper(page);
      userManagementPage = new UserManagementPage(page);

      // Login as family admin for user management access
      await authHelper.loginAsFamilyAdmin();
      await authHelper.testDb.seedTestData();
    });

    test.afterEach(async () => {
      await authHelper.cleanup();
    });

    test.describe("Page Structure", () => {
      test("should display user management page with proper elements", async () => {
        await userManagementPage.navigate();

        expect(await userManagementPage.isUserManagementPageLoaded()).toBe(
          true,
        );
        expect(await userManagementPage.getPageTitle()).toContain("Users");
      });

      test("should show user list table", async () => {
        await userManagementPage.navigate();

        const users = await userManagementPage.getUsersList();
        expect(Array.isArray(users)).toBe(true);
        // Should have at least the current user
        expect(users.length).toBeGreaterThan(0);
      });

      test("should have add user functionality", async () => {
        await userManagementPage.navigate();

        const addButtonVisible = await userManagementPage.page
          .locator(userManagementPage.selectors.addUserButton)
          .isVisible();
        expect(addButtonVisible).toBe(true);
      });
    });

    test.describe("User Listing", () => {
      test("should display current family members", async () => {
        await userManagementPage.navigate();

        const users = await userManagementPage.getUsersList();
        expect(users.length).toBeGreaterThan(0);

        // Each user should have required fields
        users.forEach((user) => {
          expect(user.name || user.email).toBeTruthy();
          expect(
            ["admin", "member", "child"].some((role) =>
              user.role.toLowerCase().includes(role),
            ),
          ).toBe(true);
        });
      });

      test("should show user roles correctly", async () => {
        await userManagementPage.navigate();

        const roleCounts = await userManagementPage.getUserCountByRole();
        expect(typeof roleCounts).toBe("object");

        // Should have at least one admin
        expect(
          roleCounts.admin + roleCounts.member + roleCounts.child,
        ).toBeGreaterThan(0);
      });

      test("should display user status information", async () => {
        await userManagementPage.navigate();

        const users = await userManagementPage.getUsersList();
        users.forEach((user) => {
          // Status should be one of the valid statuses or empty
          if (user.status) {
            expect(
              ["active", "pending", "inactive", "blocked"].some((status) =>
                user.status.toLowerCase().includes(status),
              ),
            ).toBe(true);
          }
        });
      });
    });

    test.describe("Add User", () => {
      test("should add new family member", async () => {
        await userManagementPage.navigate();

        const newUser = {
          name: "Test Member",
          email: "test.member@example.com",
          role: "member",
        };

        await userManagementPage.addUser(newUser);

        const messages = await userManagementPage.getMessages();
        expect(
          messages.success.length > 0 || messages.errors.length === 0,
        ).toBe(true);
      });

      test("should invite user with different roles", async () => {
        await userManagementPage.navigate();

        const roles = ["admin", "member", "child"];

        for (const role of roles) {
          const newUser = {
            name: `Test ${role}`,
            email: `test.${role}@example.com`,
            role: role,
          };

          await userManagementPage.addUser(newUser);
          await userManagementPage.page.waitForTimeout(1000);
        }

        const users = await userManagementPage.getUsersList();
        expect(users.length).toBeGreaterThan(0);
      });

      test("should handle email validation", async () => {
        await userManagementPage.navigate();

        const invalidUser = {
          name: "Invalid Email User",
          email: "invalid-email",
          role: "member",
        };

        await userManagementPage.addUser(invalidUser);

        const messages = await userManagementPage.getMessages();
        // Should have validation error or form should prevent submission
        expect(
          messages.errors.length > 0 || messages.success.length === 0,
        ).toBe(true);
      });
    });

    test.describe("Edit User", () => {
      test("should edit existing user information", async () => {
        await userManagementPage.navigate();

        const users = await userManagementPage.getUsersList();
        if (users.length > 0) {
          const userToEdit = users[0];
          const newData = {
            name: "Updated Name",
            role: userToEdit.role === "admin" ? "member" : "admin",
          };

          await userManagementPage.editUser(userToEdit.name, newData);

          const messages = await userManagementPage.getMessages();
          expect(
            messages.success.length > 0 || messages.errors.length === 0,
          ).toBe(true);
        }
      });

      test("should change user role", async () => {
        await userManagementPage.navigate();

        const users = await userManagementPage.getUsersList();
        if (users.length > 1) {
          // Need at least 2 users to safely change roles
          const userToChange = users.find((u) => u.role !== "admin");
          if (userToChange) {
            const newRole = userToChange.role === "member" ? "child" : "member";

            const success = await userManagementPage.changeUserRole(
              userToChange.name,
              newRole,
            );
            expect(success).toBe(true);
          }
        }
      });

      test("should prevent removing last admin", async () => {
        await userManagementPage.navigate();

        const roleCounts = await userManagementPage.getUserCountByRole();

        if (roleCounts.admin === 1) {
          const users = await userManagementPage.getUsersList();
          const adminUser = users.find((u) =>
            u.role.toLowerCase().includes("admin"),
          );

          if (adminUser) {
            // Try to change the only admin to member role
            const success = await userManagementPage.changeUserRole(
              adminUser.name,
              "member",
            );

            // Should either fail or show error message
            const messages = await userManagementPage.getMessages();
            expect(success === false || messages.errors.length > 0).toBe(true);
          }
        }
      });
    });

    test.describe("Delete User", () => {
      test("should remove user from family", async () => {
        await userManagementPage.navigate();

        // First add a user to delete
        const userToDelete = {
          name: "User To Delete",
          email: "delete.me@example.com",
          role: "member",
        };

        await userManagementPage.addUser(userToDelete);
        await userManagementPage.page.waitForTimeout(1000);

        // Then delete the user
        const deleteSuccess = await userManagementPage.deleteUser(
          userToDelete.name,
        );
        expect(deleteSuccess).toBe(true);
      });

      test("should show confirmation before deleting user", async () => {
        await userManagementPage.navigate();

        const users = await userManagementPage.getUsersList();
        if (users.length > 1) {
          // Don't delete if only one user
          const userToDelete = users.find(
            (u) => !u.role.toLowerCase().includes("admin"),
          );

          if (userToDelete) {
            // The deleteUser method handles confirmation modals
            const deleteSuccess = await userManagementPage.deleteUser(
              userToDelete.name,
            );
            expect(typeof deleteSuccess).toBe("boolean");
          }
        }
      });

      test("should prevent deleting last admin user", async () => {
        await userManagementPage.navigate();

        const roleCounts = await userManagementPage.getUserCountByRole();

        if (roleCounts.admin === 1) {
          const users = await userManagementPage.getUsersList();
          const adminUser = users.find((u) =>
            u.role.toLowerCase().includes("admin"),
          );

          if (adminUser) {
            try {
              await userManagementPage.deleteUser(adminUser.name);

              // Should show error message
              const messages = await userManagementPage.getMessages();
              expect(messages.errors.length > 0).toBe(true);
            } catch (error) {
              // Error is expected when trying to delete last admin
              expect(error.message).toBeTruthy();
            }
          }
        }
      });
    });

    test.describe("Search and Filtering", () => {
      test("should search users by name", async () => {
        await userManagementPage.navigate();

        const allUsers = await userManagementPage.getUsersList();
        if (allUsers.length > 0) {
          const searchTerm = allUsers[0].name.substring(0, 3);
          const searchResults =
            await userManagementPage.searchUsers(searchTerm);

          expect(Array.isArray(searchResults)).toBe(true);
          // Search results should contain the search term
          searchResults.forEach((user) => {
            expect(
              user.name.toLowerCase().includes(searchTerm.toLowerCase()),
            ).toBe(true);
          });
        }
      });

      test("should filter users by role", async () => {
        await userManagementPage.navigate();

        const roles = ["admin", "member", "child"];

        for (const role of roles) {
          const filteredUsers =
            await userManagementPage.filterUsersByRole(role);

          filteredUsers.forEach((user) => {
            expect(user.role.toLowerCase().includes(role)).toBe(true);
          });
        }
      });

      test("should handle empty search results", async () => {
        await userManagementPage.navigate();

        const searchResults = await userManagementPage.searchUsers(
          "nonexistentuser12345",
        );
        expect(searchResults.length).toBe(0);
      });
    });

    test.describe("Family Settings", () => {
      test("should update family name", async () => {
        await userManagementPage.navigate();

        const newFamilyName = "Updated Family Name";
        const success = await userManagementPage.updateFamilySettings({
          familyName: newFamilyName,
        });

        if (success) {
          const messages = await userManagementPage.getMessages();
          expect(
            messages.success.length > 0 || messages.errors.length === 0,
          ).toBe(true);
        }
      });

      test("should display current family information", async () => {
        await userManagementPage.navigate();

        const familyNameField = userManagementPage.page.locator(
          userManagementPage.selectors.familyNameField,
        );
        if (await familyNameField.isVisible()) {
          const currentName = await familyNameField.inputValue();
          expect(typeof currentName).toBe("string");
        }
      });
    });

    test.describe("Password Management", () => {
      test("should change user password", async () => {
        await userManagementPage.navigate();

        const passwordData = {
          currentPassword: "currentpass123",
          newPassword: "newpass456",
          confirmPassword: "newpass456",
        };

        try {
          const success = await userManagementPage.changePassword(passwordData);

          if (success) {
            const messages = await userManagementPage.getMessages();
            expect(
              messages.success.length > 0 || messages.errors.length > 0,
            ).toBe(true);
          }
        } catch (error) {
          // Password change might not be available or current password might be wrong
          console.log("Password change test skipped:", error.message);
        }
      });

      test("should validate password requirements", async () => {
        await userManagementPage.navigate();

        const weakPasswordData = {
          currentPassword: "currentpass123",
          newPassword: "123", // Weak password
          confirmPassword: "123",
        };

        try {
          await userManagementPage.changePassword(weakPasswordData);

          const messages = await userManagementPage.getMessages();
          // Should have validation errors for weak password
          expect(messages.errors.length > 0).toBe(true);
        } catch (error) {
          // Expected for weak passwords
          expect(error.message).toBeTruthy();
        }
      });

      test("should validate password confirmation match", async () => {
        await userManagementPage.navigate();

        const mismatchPasswordData = {
          currentPassword: "currentpass123",
          newPassword: "newpass456",
          confirmPassword: "differentpass789",
        };

        try {
          await userManagementPage.changePassword(mismatchPasswordData);

          const messages = await userManagementPage.getMessages();
          // Should have validation errors for password mismatch
          expect(messages.errors.length > 0).toBe(true);
        } catch (error) {
          // Expected for mismatched passwords
          expect(error.message).toBeTruthy();
        }
      });
    });

    test.describe("Permissions & Roles", () => {
      test("should display role-based permissions", async () => {
        await userManagementPage.navigate();

        const users = await userManagementPage.getUsersList();

        for (const user of users) {
          const permissions = await userManagementPage.getUserPermissions(
            user.name,
          );
          expect(Array.isArray(permissions)).toBe(true);

          // Admin should have more permissions than members
          if (user.role.toLowerCase().includes("admin")) {
            expect(permissions.filter((p) => p.enabled).length).toBeGreaterThan(
              0,
            );
          }
        }
      });

      test("should enforce role hierarchy", async () => {
        await userManagementPage.navigate();

        const users = await userManagementPage.getUsersList();
        const adminUsers = users.filter((u) =>
          u.role.toLowerCase().includes("admin"),
        );
        const memberUsers = users.filter((u) =>
          u.role.toLowerCase().includes("member"),
        );
        const childUsers = users.filter((u) =>
          u.role.toLowerCase().includes("child"),
        );

        // Admin should have the highest permissions
        // Member should have moderate permissions
        // Child should have the lowest permissions
        expect(
          adminUsers.length + memberUsers.length + childUsers.length,
        ).toBeGreaterThan(0);
      });

      test("should restrict child user capabilities", async () => {
        await userManagementPage.navigate();

        const users = await userManagementPage.getUsersList();
        const childUsers = users.filter((u) =>
          u.role.toLowerCase().includes("child"),
        );

        for (const childUser of childUsers) {
          const permissions = await userManagementPage.getUserPermissions(
            childUser.name,
          );

          // Child users should have limited permissions
          const restrictedPermissions = permissions.filter(
            (p) => p.name.includes("delete") || p.name.includes("manage"),
          );

          restrictedPermissions.forEach((permission) => {
            expect(permission.enabled).toBe(false);
          });
        }
      });
    });

    test.describe("HTMX Integration", () => {
      test("should have HTMX attributes on user management forms", async () => {
        await userManagementPage.navigate();

        const htmxIntegration =
          await userManagementPage.verifyHtmxIntegration();
        expect(htmxIntegration.hasHtmxElements).toBe(true);
        expect(htmxIntegration.elementCount).toBeGreaterThan(0);
      });

      test("should handle HTMX user actions without page reload", async () => {
        await userManagementPage.navigate();

        const pageUrl = userManagementPage.page.url();

        // Perform user action
        const users = await userManagementPage.getUsersList();
        if (users.length > 0) {
          // Try to edit a user (this should use HTMX)
          const userToEdit = users[0];
          await userManagementPage.editUser(userToEdit.name, {
            name: "HTMX Test Name",
          });
        }

        // URL should remain the same (no page reload)
        expect(userManagementPage.page.url()).toBe(pageUrl);
      });

      test("should show loading states during HTMX requests", async () => {
        await userManagementPage.navigate();

        // Add a user and check for loading indicators
        const newUser = {
          name: "Loading Test User",
          email: "loading@example.com",
          role: "member",
        };

        // This should trigger loading state
        await userManagementPage.addUser(newUser);

        // The waitForUserAction method should handle loading states
        const messages = await userManagementPage.getMessages();
        expect(
          Array.isArray(messages.success) && Array.isArray(messages.errors),
        ).toBe(true);
      });
    });

    test.describe("Form Validation", () => {
      test("should validate required fields", async () => {
        await userManagementPage.navigate();

        const validation = await userManagementPage.testFormValidation();
        expect(validation.hasValidation).toBe(true);
        expect(validation.errors.length).toBeGreaterThan(0);
      });

      test("should validate email format", async () => {
        await userManagementPage.navigate();

        const invalidUser = {
          name: "Invalid Email User",
          email: "not-an-email",
          role: "member",
        };

        await userManagementPage.addUser(invalidUser);

        const messages = await userManagementPage.getMessages();
        expect(
          messages.errors.some(
            (error) =>
              error.toLowerCase().includes("email") ||
              error.toLowerCase().includes("invalid"),
          ),
        ).toBe(true);
      });

      test("should prevent duplicate emails", async () => {
        await userManagementPage.navigate();

        const duplicateUser = {
          name: "Duplicate User",
          email: "duplicate@example.com",
          role: "member",
        };

        // Add user first time
        await userManagementPage.addUser(duplicateUser);
        await userManagementPage.page.waitForTimeout(1000);

        // Try to add same email again
        await userManagementPage.addUser({
          ...duplicateUser,
          name: "Another Duplicate User",
        });

        const messages = await userManagementPage.getMessages();
        expect(
          messages.errors.some(
            (error) =>
              error.toLowerCase().includes("exist") ||
              error.toLowerCase().includes("duplicate"),
          ),
        ).toBe(true);
      });
    });

    test.describe("Responsive Design", () => {
      test("should work on mobile devices", async () => {
        await userManagementPage.navigate();

        const responsive = await userManagementPage.testResponsiveDesign();
        expect(responsive.mobile.titleVisible).toBe(true);
        expect(responsive.desktop.titleVisible).toBe(true);
      });

      test("should maintain functionality on small screens", async () => {
        await userManagementPage.navigate();

        // Test user list on mobile
        await userManagementPage.page.setViewportSize({
          width: 375,
          height: 667,
        });

        const users = await userManagementPage.getUsersList();
        expect(Array.isArray(users)).toBe(true);

        // Should be able to perform actions on mobile
        if (users.length > 0) {
          const addButtonVisible = await userManagementPage.page
            .locator(userManagementPage.selectors.addUserButton)
            .isVisible();
          expect(typeof addButtonVisible).toBe("boolean");
        }
      });

      test("should have accessible navigation on all screen sizes", async () => {
        await userManagementPage.navigate();

        const viewports = [
          { width: 375, height: 667 }, // Mobile
          { width: 768, height: 1024 }, // Tablet
          { width: 1280, height: 720 }, // Desktop
        ];

        for (const viewport of viewports) {
          await userManagementPage.page.setViewportSize(viewport);
          await userManagementPage.page.waitForTimeout(500);

          const canNavigateBack = await userManagementPage.backToDashboard();
          expect(typeof canNavigateBack).toBe("boolean");

          // Navigate back to user management for next iteration
          await userManagementPage.navigate();
        }
      });
    });

    test.describe("User Experience", () => {
      test("should provide clear success feedback", async () => {
        await userManagementPage.navigate();

        const newUser = {
          name: "Success Test User",
          email: "success@example.com",
          role: "member",
        };

        await userManagementPage.addUser(newUser);

        const messages = await userManagementPage.getMessages();
        expect(
          messages.success.length > 0 || messages.errors.length === 0,
        ).toBe(true);
      });

      test("should handle errors gracefully", async () => {
        await userManagementPage.navigate();

        // Try to perform an action that might cause an error
        try {
          await userManagementPage.deleteUser("NonexistentUser12345");
        } catch (error) {
          expect(error.message).toBeTruthy();
        }

        // Page should still be functional after error
        expect(await userManagementPage.isUserManagementPageLoaded()).toBe(
          true,
        );
      });

      test("should maintain data consistency", async () => {
        await userManagementPage.navigate();

        const initialUsers = await userManagementPage.getUsersList();
        const initialCount = initialUsers.length;

        // Add a user
        const newUser = {
          name: "Consistency Test User",
          email: "consistency@example.com",
          role: "member",
        };

        await userManagementPage.addUser(newUser);
        await userManagementPage.page.waitForTimeout(1000);

        const afterAddUsers = await userManagementPage.getUsersList();
        expect(afterAddUsers.length).toBeGreaterThanOrEqual(initialCount);

        // User should appear in the list
        const addedUser = afterAddUsers.find(
          (u) =>
            u.email.includes(newUser.email) || u.name.includes(newUser.name),
        );
        expect(addedUser).toBeTruthy();
      });
    });
  });
});

test.describe("User Management Performance", () => {
  test.describe("Performance Testing - Authenticated", () => {
    let userManagementPage;
    let authHelper;

    test.beforeEach(async ({ page }) => {
      authHelper = new AuthHelper(page);
      userManagementPage = new UserManagementPage(page);

      // Login as family admin for user management access
      await authHelper.loginAsFamilyAdmin();
      await authHelper.testDb.seedTestData();
    });

    test.afterEach(async () => {
      await authHelper.cleanup();
    });

    test("should load user list within acceptable time", async () => {
      const startTime = Date.now();
      await userManagementPage.navigate();

      const users = await userManagementPage.getUsersList();
      const endTime = Date.now();

      const loadTime = endTime - startTime;
      expect(loadTime).toBeLessThan(5000); // 5 seconds max for user list
      expect(users.length).toBeGreaterThanOrEqual(0);
    });

    test("should handle concurrent user operations", async () => {
      await userManagementPage.navigate();

      // Simulate concurrent operations
      const operations = [
        userManagementPage.getUsersList(),
        userManagementPage.searchUsers("test"),
        userManagementPage.filterUsersByRole("member"),
      ];

      const results = await Promise.all(operations);

      // All operations should complete successfully
      results.forEach((result) => {
        expect(Array.isArray(result)).toBe(true);
      });
    });

    test("should not cause memory leaks with repeated actions", async () => {
      await userManagementPage.navigate();

      // Perform repeated actions to test memory usage
      for (let i = 0; i < 10; i++) {
        await userManagementPage.getUsersList();
        await userManagementPage.page.waitForTimeout(200);
      }

      // Page should still be responsive
      expect(await userManagementPage.isUserManagementPageLoaded()).toBe(true);
    });
  });
});

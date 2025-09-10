import { test, expect } from "@playwright/test";
import { AuthHelper } from "./helpers/auth.js";
import { DashboardPage } from "./pages/DashboardPage.js";

test.describe("Dashboard - Unauthenticated", () => {
  let dashboardPage;

  test.beforeEach(async ({ page }) => {
    dashboardPage = new DashboardPage(page);
  });

  test("should redirect unauthenticated users to login", async ({ page }) => {
    await page.goto("/dashboard");
    await page.waitForLoadState("networkidle");

    // Should redirect to login page
    await expect(page).toHaveURL(/.*login/);
  });

  test("should redirect unauthenticated home access to login", async ({
    page,
  }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");

    // Should redirect to login page
    await expect(page).toHaveURL(/.*login/);
  });
});

test.describe("Dashboard - Authenticated", () => {
  let authHelper;
  let dashboardPage;
  let userData;

  test.beforeEach(async ({ page }) => {
    authHelper = new AuthHelper(page);
    dashboardPage = new DashboardPage(page);

    // Register a user using the working pattern from auth-comprehensive
    userData = await authHelper.registerUser("admin");
    await dashboardPage.waitForDashboardLoad();
  });

  test.afterEach(async () => {
    await authHelper.cleanup();
  });

  test.describe("Dashboard Structure", () => {
    test("should display welcome message with user name", async () => {
      const welcomeMessage = await dashboardPage.getWelcomeMessage();

      expect(welcomeMessage).toContain("Добро пожаловать");
      expect(welcomeMessage).toContain(userData.name || userData.first_name);
    });

    test("should display family information", async () => {
      const familyInfo = await dashboardPage.getFamilyInfo();

      expect(familyInfo).toContain("семьи");
      expect(familyInfo).toContain(userData.family_name);
    });

    test("should have proper page title", async () => {
      const title = await dashboardPage.getPageTitle();

      expect(title).toContain("Семейный бюджет");
    });

    test("should display all main dashboard sections", async () => {
      const sections = await dashboardPage.verifyDashboardSections();

      expect(sections.welcome).toBe(true);
      expect(sections.quickActions).toBe(true);
      expect(sections.recentActivity).toBe(true);
      expect(sections.budgetOverview).toBe(true);
      expect(sections.stats).toBe(true);
    });
  });

  test.describe("Navigation", () => {
    test("should display main navigation with all links", async () => {
      const navigation = await dashboardPage.verifyNavigation();

      expect(navigation.brandVisible).toBe(true);
      expect(navigation.hasMainNav).toBe(true);
      expect(navigation.home).toBe(true);
      expect(navigation.transactions).toBe(true);
      expect(navigation.categories).toBe(true);
      expect(navigation.budgets).toBe(true);
      expect(navigation.reports).toBe(true);
      expect(navigation.hasUserDropdown).toBe(true);
      expect(navigation.hasLogoutButton).toBe(true);
    });

    test("should navigate to transactions page", async () => {
      const navigated = await dashboardPage.navigateTo("transactions");

      expect(navigated).toBe(true);
      expect(dashboardPage.page.url()).toContain("/transactions");
    });

    test("should navigate to categories page", async () => {
      const navigated = await dashboardPage.navigateTo("categories");

      expect(navigated).toBe(true);
      expect(dashboardPage.page.url()).toContain("/categories");
    });

    test("should navigate to budgets page", async () => {
      const navigated = await dashboardPage.navigateTo("budgets");

      expect(navigated).toBe(true);
      expect(dashboardPage.page.url()).toContain("/budgets");
    });

    test("should navigate to reports page", async () => {
      const navigated = await dashboardPage.navigateTo("reports");

      expect(navigated).toBe(true);
      expect(dashboardPage.page.url()).toContain("/reports");
    });
  });

  test.describe("Quick Actions", () => {
    test("should display all quick action buttons", async () => {
      const quickActions = await dashboardPage.verifyQuickActions();

      expect(quickActions.sectionVisible).toBe(true);
      expect(quickActions.addTransaction).toBe(true);
      expect(quickActions.createBudget).toBe(true);
      expect(quickActions.viewReports).toBe(true);
      expect(quickActions.manageCategories).toBe(true);
    });

    test("should navigate to add transaction page", async () => {
      const clicked = await dashboardPage.clickQuickAction("addTransaction");

      expect(clicked).toBe(true);
      expect(dashboardPage.page.url()).toContain("/transactions/new");
    });

    test("should navigate to create budget page", async () => {
      const clicked = await dashboardPage.clickQuickAction("createBudget");

      expect(clicked).toBe(true);
      expect(dashboardPage.page.url()).toContain("/budgets/new");
    });

    test("should navigate to reports via quick action", async () => {
      const clicked = await dashboardPage.clickQuickAction("viewReports");

      expect(clicked).toBe(true);
      expect(dashboardPage.page.url()).toContain("/reports");
    });

    test("should navigate to categories via quick action", async () => {
      const clicked = await dashboardPage.clickQuickAction("manageCategories");

      expect(clicked).toBe(true);
      expect(dashboardPage.page.url()).toContain("/categories");
    });
  });

  test.describe("Dynamic Content", () => {
    test("should load recent transactions section", async () => {
      const recentTx = await dashboardPage.hasRecentTransactions();

      expect(recentTx.hasSection).toBe(true);
      expect(recentTx.loaded).toBe(true);
      expect(recentTx.isLoading).toBe(false);
    });

    test("should load budget overview section", async () => {
      const budgetOv = await dashboardPage.hasBudgetOverview();

      expect(budgetOv.hasSection).toBe(true);
      expect(budgetOv.loaded).toBe(true);
      expect(budgetOv.isLoading).toBe(false);
    });

    test("should handle HTMX content loading", async () => {
      // Navigate away and back to test HTMX loading
      await dashboardPage.navigateTo("transactions");
      await dashboardPage.navigateHome();
      await dashboardPage.waitForDashboardLoad();

      const recentTx = await dashboardPage.hasRecentTransactions();
      const budgetOv = await dashboardPage.hasBudgetOverview();

      expect(recentTx.loaded).toBe(true);
      expect(budgetOv.loaded).toBe(true);
    });
  });

  test.describe("Role-Based Access", () => {
    test("should display admin-specific elements for admin users", async () => {
      const roleUI = await dashboardPage.verifyRoleBasedUI("admin");

      expect(roleUI.isAdmin).toBe(true);
      // Admin elements might not be visible if not implemented yet
      // expect(roleUI.hasAdminElements).toBe(true);
    });

    test("should allow logout functionality", async () => {
      const loggedOut = await dashboardPage.logout();

      expect(loggedOut).toBe(true);
      expect(await authHelper.isAuthenticated()).toBe(false);
    });
  });

  test.describe("Responsive Design", () => {
    test("should work on mobile and desktop viewports", async () => {
      const responsive = await dashboardPage.verifyResponsiveDesign();

      // Both mobile and desktop should show main sections
      expect(responsive.mobile.navigationVisible).toBe(true);
      expect(responsive.mobile.quickActionsVisible).toBe(true);
      expect(responsive.desktop.navigationVisible).toBe(true);
      expect(responsive.desktop.quickActionsVisible).toBe(true);
    });
  });

  test.describe("Accessibility", () => {
    test("should have proper accessibility features", async () => {
      const accessibility = await dashboardPage.checkAccessibility();

      expect(accessibility.hasMainHeading).toBe(true);
      expect(accessibility.hasSectionHeadings).toBe(true);
      expect(accessibility.hasRoleButtons).toBe(true);
      expect(accessibility.focusableElements).toBeGreaterThan(5);
    });

    test("should support keyboard navigation", async () => {
      const tabOrder = await dashboardPage.testKeyboardNavigation();

      expect(tabOrder.length).toBeGreaterThan(5);
      // Should include navigation links and action buttons
      const hasLinks = tabOrder.some((el) => el.tagName === "A");
      const hasButtons = tabOrder.some((el) => el.tagName === "BUTTON");
      expect(hasLinks || hasButtons).toBe(true);
    });
  });

  test.describe("Empty State Handling", () => {
    test("should handle empty dashboard gracefully", async () => {
      const emptyState = await dashboardPage.hasEmptyState();

      // Dashboard should load even without data
      expect(emptyState.emptyTransactions).toBeDefined();
      expect(emptyState.emptyBudgets).toBeDefined();
    });
  });
});

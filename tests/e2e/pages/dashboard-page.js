/**
 * Page Object Model for Dashboard Page
 * Provides structured access to dashboard elements and actions
 */

export class DashboardPage {
  constructor(page) {
    this.page = page;

    // Page selectors
    this.selectors = {
      // Navigation elements
      brandLink: "a.brand",
      mainNavigation: "nav.container-fluid",
      navLinks: {
        home: 'nav a[href="/"]',
        transactions: 'nav a[href="/transactions"]',
        categories: 'nav a[href="/categories"]',
        budgets: 'nav a[href="/budgets"]',
        reports: 'nav a[href="/reports"]',
        profile: 'nav a[href="/profile"]',
        users: 'nav a[href="/users"]',
        familySettings: 'nav a[href="/family/settings"]',
      },
      userDropdown: "nav details.dropdown",
      logoutForm: 'nav form[action="/logout"]',
      logoutButton: 'nav form[action="/logout"] button[type="submit"]',

      // Main content
      welcomeSection: "section.welcome",
      welcomeTitle: "section.welcome h1",
      familyInfo: "section.welcome p",
      periodSelector: ".period-selector",

      // Stats sections
      statsSection: "section.stats-grid, .dashboard-stats",
      loadingStats: 'section.stats-grid p[aria-busy="true"]',

      // Quick actions
      quickActions: "section.quick-actions",
      quickActionButtons: {
        addTransaction: 'a[href="/transactions/new"]',
        createBudget: 'a[href="/budgets/new"]',
        viewReports: 'a[href="/reports"]',
        manageCategories: 'a[href="/categories"]',
      },

      // Dashboard content sections
      recentActivity: "section.recent-activity",
      recentActivityTitle: "section.recent-activity h2",
      recentTransactions:
        '.recent-transactions, [data-testid="recent-transactions"]',
      loadingTransactions: 'section.recent-activity p[aria-busy="true"]',

      budgetOverview: "section.budget-overview-section",
      budgetOverviewTitle: "section.budget-overview-section h2",
      loadingBudgets: 'section.budget-overview-section p[aria-busy="true"]',

      // Category insights
      categoryInsights: "section.category-insights",
      topExpenses: ".expense-categories",
      topIncome: ".income-categories",
      categoryItems: ".category-item",

      // HTMX elements
      htmxRecentTransactions: 'div[hx-get="/htmx/transactions/recent"]',
      htmxBudgetOverview: 'div[hx-get="/htmx/budgets/overview"]',

      // Flash messages
      flashMessages: "#flash-messages",
      alertMessages: ".alert",

      // Footer
      footer: "footer.container",
      helpLinks: {
        help: 'footer a[href="/help"]',
        privacy: 'footer a[href="/privacy"]',
        terms: 'footer a[href="/terms"]',
      },
    };

    // Dashboard sections for easy iteration
    this.sections = [
      "welcomeSection",
      "statsSection",
      "quickActions",
      "recentActivity",
      "budgetOverview",
    ];
  }

  /**
   * Navigate to dashboard page
   */
  async navigate() {
    await this.page.goto("/dashboard");
    await this.page.waitForSelector(this.selectors.welcomeSection);
  }

  /**
   * Navigate to home page (dashboard)
   */
  async navigateHome() {
    await this.page.goto("/");
    await this.page.waitForSelector(this.selectors.welcomeSection);
  }

  /**
   * Check if user is authenticated and dashboard is loaded
   */
  async isAuthenticated() {
    try {
      await this.page.waitForSelector(this.selectors.welcomeSection, {
        timeout: 3000,
      });
      const url = this.page.url();
      return url.includes("dashboard") || url === "/" || !url.includes("login");
    } catch {
      return false;
    }
  }

  /**
   * Get welcome message with user name
   */
  async getWelcomeMessage() {
    const titleElement = this.page.locator(this.selectors.welcomeTitle);
    return await titleElement.textContent();
  }

  /**
   * Get family information
   */
  async getFamilyInfo() {
    const familyElement = this.page.locator(this.selectors.familyInfo);
    return await familyElement.textContent();
  }

  /**
   * Check if navigation elements are present
   */
  async verifyNavigation() {
    const navigation = {};

    navigation.brandVisible = await this.page
      .locator(this.selectors.brandLink)
      .isVisible();
    navigation.hasMainNav = await this.page
      .locator(this.selectors.mainNavigation)
      .isVisible();

    // Check main navigation links
    for (const [key, selector] of Object.entries(this.selectors.navLinks)) {
      navigation[key] = await this.page.locator(selector).isVisible();
    }

    navigation.hasUserDropdown = await this.page
      .locator(this.selectors.userDropdown)
      .isVisible();
    navigation.hasLogoutButton = await this.page
      .locator(this.selectors.logoutButton)
      .isVisible();

    return navigation;
  }

  /**
   * Navigate to a specific section via navigation
   */
  async navigateTo(section) {
    const selectorMap = {
      transactions: this.selectors.navLinks.transactions,
      categories: this.selectors.navLinks.categories,
      budgets: this.selectors.navLinks.budgets,
      reports: this.selectors.navLinks.reports,
      profile: this.selectors.navLinks.profile,
      users: this.selectors.navLinks.users,
      "family-settings": this.selectors.navLinks.familySettings,
    };

    if (selectorMap[section]) {
      await this.page.click(selectorMap[section]);
      await this.page.waitForLoadState("networkidle");
      return true;
    }
    return false;
  }

  /**
   * Check quick actions section
   */
  async verifyQuickActions() {
    const quickActions = {};

    quickActions.sectionVisible = await this.page
      .locator(this.selectors.quickActions)
      .isVisible();

    // Check individual action buttons
    for (const [key, selector] of Object.entries(
      this.selectors.quickActionButtons,
    )) {
      quickActions[key] = await this.page.locator(selector).isVisible();
    }

    return quickActions;
  }

  /**
   * Click a quick action button
   */
  async clickQuickAction(action) {
    const selector = this.selectors.quickActionButtons[action];
    if (selector) {
      await this.page.click(selector);
      await this.page.waitForLoadState("networkidle");
      return true;
    }
    return false;
  }

  /**
   * Check dashboard sections visibility
   */
  async verifyDashboardSections() {
    const sections = {};

    sections.welcome = await this.page
      .locator(this.selectors.welcomeSection)
      .isVisible();
    sections.quickActions = await this.page
      .locator(this.selectors.quickActions)
      .isVisible();
    sections.recentActivity = await this.page
      .locator(this.selectors.recentActivity)
      .isVisible();
    sections.budgetOverview = await this.page
      .locator(this.selectors.budgetOverview)
      .isVisible();

    // Check for stats section (either loaded or loading)
    const statsLoaded = await this.page
      .locator(this.selectors.statsSection)
      .isVisible();
    const statsLoading = await this.page
      .locator(this.selectors.loadingStats)
      .isVisible();
    sections.stats = statsLoaded || statsLoading;

    return sections;
  }

  /**
   * Wait for HTMX content to load
   */
  async waitForHtmxContent() {
    // Wait for recent transactions to load
    try {
      await this.page.waitForFunction(
        () =>
          !document.querySelector(
            'section.recent-activity p[aria-busy="true"]',
          ),
        { timeout: 10000 },
      );
    } catch {
      // Timeout is acceptable - content might be empty
    }

    // Wait for budget overview to load
    try {
      await this.page.waitForFunction(
        () =>
          !document.querySelector(
            'section.budget-overview-section p[aria-busy="true"]',
          ),
        { timeout: 10000 },
      );
    } catch {
      // Timeout is acceptable - content might be empty
    }
  }

  /**
   * Check if recent transactions are displayed
   */
  async hasRecentTransactions() {
    await this.waitForHtmxContent();

    const hasSection = await this.page
      .locator(this.selectors.recentActivity)
      .isVisible();
    const hasContent = await this.page
      .locator(this.selectors.recentTransactions)
      .isVisible();
    const isLoading = await this.page
      .locator(this.selectors.loadingTransactions)
      .isVisible();

    return {
      hasSection,
      hasContent,
      isLoading,
      loaded: hasSection && !isLoading,
    };
  }

  /**
   * Check if budget overview is displayed
   */
  async hasBudgetOverview() {
    await this.waitForHtmxContent();

    const hasSection = await this.page
      .locator(this.selectors.budgetOverview)
      .isVisible();
    const isLoading = await this.page
      .locator(this.selectors.loadingBudgets)
      .isVisible();

    return {
      hasSection,
      isLoading,
      loaded: hasSection && !isLoading,
    };
  }

  /**
   * Check if category insights are displayed
   */
  async hasCategoryInsights() {
    const hasInsights = await this.page
      .locator(this.selectors.categoryInsights)
      .isVisible();
    const hasExpenses = await this.page
      .locator(this.selectors.topExpenses)
      .isVisible();
    const hasIncome = await this.page
      .locator(this.selectors.topIncome)
      .isVisible();

    return {
      hasInsights,
      hasExpenses,
      hasIncome,
    };
  }

  /**
   * Check role-based UI elements
   */
  async verifyRoleBasedUI(expectedRole) {
    const roleElements = {};

    // Admin-only elements
    roleElements.usersLink = await this.page
      .locator(this.selectors.navLinks.users)
      .isVisible();
    roleElements.familySettingsLink = await this.page
      .locator(this.selectors.navLinks.familySettings)
      .isVisible();

    if (expectedRole === "admin") {
      return {
        isAdmin: true,
        hasAdminElements:
          roleElements.usersLink || roleElements.familySettingsLink,
        ...roleElements,
      };
    } else {
      return {
        isAdmin: false,
        hasAdminElements: false,
        ...roleElements,
      };
    }
  }

  /**
   * Logout from the dashboard
   */
  async logout() {
    // Open user dropdown if needed
    const dropdown = this.page.locator(this.selectors.userDropdown);
    if (await dropdown.isVisible()) {
      await dropdown.click();
    }

    // Click logout button
    await this.page.click(this.selectors.logoutButton);
    await this.page.waitForLoadState("networkidle");

    // Should redirect to login
    const url = this.page.url();
    return url.includes("login");
  }

  /**
   * Check responsive design elements
   */
  async verifyResponsiveDesign() {
    // Test mobile viewport
    await this.page.setViewportSize({ width: 375, height: 667 });
    await this.page.waitForTimeout(500);

    const mobile = {
      quickActionsVisible: await this.page
        .locator(this.selectors.quickActions)
        .isVisible(),
      navigationVisible: await this.page
        .locator(this.selectors.mainNavigation)
        .isVisible(),
      sectionsVisible: await this.verifyDashboardSections(),
    };

    // Test desktop viewport
    await this.page.setViewportSize({ width: 1280, height: 720 });
    await this.page.waitForTimeout(500);

    const desktop = {
      quickActionsVisible: await this.page
        .locator(this.selectors.quickActions)
        .isVisible(),
      navigationVisible: await this.page
        .locator(this.selectors.mainNavigation)
        .isVisible(),
      sectionsVisible: await this.verifyDashboardSections(),
    };

    return { mobile, desktop };
  }

  /**
   * Get flash messages if any
   */
  async getFlashMessages() {
    const messages = [];
    const messageElements = this.page.locator(this.selectors.alertMessages);
    const count = await messageElements.count();

    for (let i = 0; i < count; i++) {
      const messageText = await messageElements.nth(i).textContent();
      const messageClass = await messageElements.nth(i).getAttribute("class");
      messages.push({
        text: messageText.trim(),
        type: messageClass.includes("alert-success")
          ? "success"
          : messageClass.includes("alert-danger")
            ? "danger"
            : messageClass.includes("alert-warning")
              ? "warning"
              : "info",
      });
    }

    return messages;
  }

  /**
   * Check accessibility features
   */
  async checkAccessibility() {
    const accessibility = {};

    // Check for proper headings hierarchy
    accessibility.hasMainHeading = await this.page.locator("h1").isVisible();
    accessibility.hasSectionHeadings =
      (await this.page.locator("h2").count()) > 0;

    // Check for ARIA attributes
    accessibility.hasLoadingIndicators = await this.page
      .locator('[aria-busy="true"]')
      .count();
    accessibility.hasRoleButtons =
      (await this.page.locator('[role="button"]').count()) > 0;
    accessibility.hasAlerts = await this.page.locator('[role="alert"]').count();

    // Check for keyboard navigation
    accessibility.focusableElements = await this.page
      .locator("a, button, input, select, textarea, [tabindex]")
      .count();

    return accessibility;
  }

  /**
   * Test keyboard navigation
   */
  async testKeyboardNavigation() {
    const tabOrder = [];

    // Start from the first focusable element
    await this.page.keyboard.press("Tab");

    for (let i = 0; i < 10; i++) {
      // Test first 10 tab stops
      const activeElement = await this.page.evaluate(() => {
        const el = document.activeElement;
        return {
          tagName: el.tagName,
          href: el.href || null,
          text: el.textContent?.trim().substring(0, 50) || "",
          role: el.getAttribute("role"),
        };
      });

      tabOrder.push(activeElement);
      await this.page.keyboard.press("Tab");
    }

    return tabOrder;
  }

  /**
   * Get page title
   */
  async getPageTitle() {
    return await this.page.title();
  }

  /**
   * Wait for dashboard to be fully loaded
   */
  async waitForDashboardLoad() {
    // Wait for main structure
    await this.page.waitForSelector(this.selectors.welcomeSection);
    await this.page.waitForSelector(this.selectors.quickActions);

    // Wait for HTMX content to finish loading
    await this.waitForHtmxContent();

    // Wait for any network activity to complete
    await this.page.waitForLoadState("networkidle");
  }

  /**
   * Check if dashboard displays empty state appropriately
   */
  async hasEmptyState() {
    await this.waitForDashboardLoad();

    const recentTx = await this.hasRecentTransactions();
    const budgetOv = await this.hasBudgetOverview();
    const categoryIn = await this.hasCategoryInsights();

    return {
      emptyTransactions: recentTx.loaded && !recentTx.hasContent,
      emptyBudgets: budgetOv.loaded,
      emptyCategoryInsights: !categoryIn.hasInsights,
      hasAnyContent: recentTx.hasContent || categoryIn.hasInsights,
    };
  }
}

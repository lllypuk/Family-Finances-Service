/**
 * Page Object Model for Login Page
 * Provides structured access to login page elements and actions
 */

export class LoginPage {
  constructor(page) {
    this.page = page;

    // Page selectors
    this.selectors = {
      emailInput: 'input[name="email"]',
      passwordInput: 'input[name="password"]',
      loginButton: 'button[type="submit"]',
      registerLink: 'a[href*="register"]',
      forgotPasswordLink: 'a[href*="forgot"]',
      csrfToken: 'input[name="_token"]',
      errorAlert: '.alert-error[role="alert"], small.error',
      successAlert: ".alert-success, .success",
      loginForm: 'form[action*="login"], form[hx-post*="login"]',
    };
  }

  /**
   * Navigate to login page
   */
  async navigate() {
    await this.page.goto("/login");
    await this.page.waitForSelector(this.selectors.emailInput);
  }

  /**
   * Fill login form
   */
  async fillLoginForm(email, password) {
    await this.page.fill(this.selectors.emailInput, email);
    await this.page.fill(this.selectors.passwordInput, password);
  }

  /**
   * Submit login form
   */
  async submitLogin() {
    await this.page.click(this.selectors.loginButton);
    await this.page.waitForLoadState("networkidle");
  }

  /**
   * Complete login process
   */
  async login(email, password) {
    await this.navigate();
    await this.fillLoginForm(email, password);
    await this.submitLogin();
  }

  /**
   * Check if CSRF token is present
   */
  async hasCsrfToken() {
    const csrfToken = this.page.locator(this.selectors.csrfToken);
    return await csrfToken.isVisible();
  }

  /**
   * Get CSRF token value
   */
  async getCsrfToken() {
    const csrfToken = this.page.locator(this.selectors.csrfToken);
    return await csrfToken.getAttribute("value");
  }

  /**
   * Check for error messages
   */
  async getErrorMessage() {
    const errorElement = this.page.locator(this.selectors.errorAlert).first();

    if (await errorElement.isVisible()) {
      return await errorElement.textContent();
    }

    return null;
  }

  /**
   * Check if form validation errors are displayed
   */
  async hasValidationErrors() {
    const errorElement = this.page.locator(this.selectors.errorAlert);
    return await errorElement.isVisible();
  }

  /**
   * Navigate to register page
   */
  async goToRegister() {
    await this.page.click(this.selectors.registerLink);
    await this.page.waitForURL(/.*register/);
  }

  /**
   * Check if email field has validation state
   */
  async isEmailFieldInvalid() {
    const emailInput = this.page.locator(this.selectors.emailInput);

    // Check for various invalid states
    const hasInvalidAttribute =
      (await emailInput.getAttribute("aria-invalid")) === "true";
    const hasErrorClass = await emailInput.evaluate(
      (el) =>
        el.classList.contains("error") || el.classList.contains("invalid"),
    );

    return hasInvalidAttribute || hasErrorClass;
  }

  /**
   * Check if password field has validation state
   */
  async isPasswordFieldInvalid() {
    const passwordInput = this.page.locator(this.selectors.passwordInput);

    const hasInvalidAttribute =
      (await passwordInput.getAttribute("aria-invalid")) === "true";
    const hasErrorClass = await passwordInput.evaluate(
      (el) =>
        el.classList.contains("error") || el.classList.contains("invalid"),
    );

    return hasInvalidAttribute || hasErrorClass;
  }

  /**
   * Wait for HTMX request to complete
   */
  async waitForHtmxRequest() {
    // Wait for HTMX to settle
    await this.page.evaluate(() => {
      return new Promise((resolve) => {
        if (window.htmx) {
          htmx.on("htmx:afterSettle", resolve);
        } else {
          resolve();
        }
      });
    });
  }

  /**
   * Check if form has HTMX attributes
   */
  async hasHtmxAttributes() {
    const form = this.page.locator(this.selectors.loginForm);

    const hasHxPost = (await form.getAttribute("hx-post")) !== null;
    const hasHxTarget = (await form.getAttribute("hx-target")) !== null;

    return hasHxPost || hasHxTarget;
  }

  /**
   * Verify page elements are present
   */
  async verifyPageElements() {
    const checks = {
      emailInput: await this.page
        .locator(this.selectors.emailInput)
        .isVisible(),
      passwordInput: await this.page
        .locator(this.selectors.passwordInput)
        .isVisible(),
      loginButton: await this.page
        .locator(this.selectors.loginButton)
        .isVisible(),
      loginForm: await this.page.locator(this.selectors.loginForm).isVisible(),
      csrfToken: await this.hasCsrfToken(),
    };

    return checks;
  }

  /**
   * Get page title
   */
  async getPageTitle() {
    return await this.page.title();
  }

  /**
   * Check if page is accessible (basic accessibility check)
   */
  async checkAccessibility() {
    // Check for proper labels
    const emailLabel = this.page.locator('label[for="email"]');
    const passwordLabel = this.page.locator('label[for="password"]');

    return {
      hasEmailLabel: await emailLabel.isVisible(),
      hasPasswordLabel: await passwordLabel.isVisible(),
      hasProperHeading: await this.page.locator("h1, h2").first().isVisible(),
      formHasRole: await this.page
        .locator("form[role], form")
        .first()
        .isVisible(),
    };
  }

  /**
   * Test keyboard navigation
   */
  async testKeyboardNavigation() {
    await this.navigate();

    // Tab through form elements
    await this.page.keyboard.press("Tab"); // Should focus email
    const emailFocused = await this.page
      .locator(this.selectors.emailInput)
      .evaluate((el) => document.activeElement === el);

    await this.page.keyboard.press("Tab"); // Should focus password
    const passwordFocused = await this.page
      .locator(this.selectors.passwordInput)
      .evaluate((el) => document.activeElement === el);

    await this.page.keyboard.press("Tab"); // Should focus submit button
    const submitFocused = await this.page
      .locator(this.selectors.loginButton)
      .evaluate((el) => document.activeElement === el);

    return {
      emailFocused,
      passwordFocused,
      submitFocused,
    };
  }
}

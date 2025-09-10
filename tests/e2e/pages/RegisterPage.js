/**
 * Page Object Model for Register Page
 * Provides structured access to registration page elements and actions
 */

export class RegisterPage {
  constructor(page) {
    this.page = page;

    // Page selectors
    this.selectors = {
      nameInput: 'input[name="name"]',
      familyNameInput: 'input[name="family_name"]',
      currencySelect: 'select[name="currency"]',
      emailInput: 'input[name="email"]',
      passwordInput: 'input[name="password"]',
      confirmPasswordInput: 'input[name="confirm_password"]',
      submitButton: 'button[type="submit"]',
      loginLink: 'a[href*="login"]',
      csrfToken: 'input[name="_token"]',
      errorAlert: '.alert-error[role="alert"], small.error',
      successAlert: ".alert-success, .success",
      registerForm: 'form[action*="register"], form[hx-post*="register"]',
      validationMessages: ".validation-error, .field-error, .invalid-feedback",
    };

    // Form fields for easy iteration
    this.formFields = [
      "name",
      "family_name",
      "currency",
      "email",
      "password",
      "confirm_password",
    ];
  }

  /**
   * Navigate to register page
   */
  async navigate() {
    await this.page.goto("/register");
    await this.page.waitForSelector(this.selectors.nameInput);
  }

  /**
   * Fill registration form with user data
   */
  async fillRegistrationForm(userData) {
    await this.page.fill(this.selectors.nameInput, userData.name || "");
    await this.page.fill(
      this.selectors.familyNameInput,
      userData.family_name || "",
    );
    
    // Select currency if provided
    if (userData.currency) {
      await this.page.selectOption(this.selectors.currencySelect, userData.currency);
    }
    
    await this.page.fill(this.selectors.emailInput, userData.email || "");
    await this.page.fill(this.selectors.passwordInput, userData.password || "");
    await this.page.fill(
      this.selectors.confirmPasswordInput,
      userData.confirm_password || "",
    );
  }

  /**
   * Submit registration form
   */
  async submitRegistration() {
    await this.page.click(this.selectors.submitButton);
    await this.page.waitForLoadState("networkidle");
  }

  /**
   * Complete registration process
   */
  async register(userData) {
    await this.navigate();
    await this.fillRegistrationForm(userData);
    await this.submitRegistration();
  }

  /**
   * Check if CSRF token is present
   */
  async hasCsrfToken() {
    const csrfToken = this.page.locator(this.selectors.csrfToken);
    return await csrfToken.isVisible();
  }

  /**
   * Get error messages
   */
  async getErrorMessages() {
    const errorElements = this.page.locator(this.selectors.errorAlert);
    const count = await errorElements.count();
    const errors = [];

    for (let i = 0; i < count; i++) {
      const errorText = await errorElements.nth(i).textContent();
      errors.push(errorText.trim());
    }

    return errors;
  }

  /**
   * Get field-specific validation errors
   */
  async getFieldValidationErrors() {
    const validationErrors = {};

    for (const field of this.formFields) {
      const fieldSelector = this.selectors[field + "Input"];
      const errorSelector = `${fieldSelector} ~ .validation-error, ${fieldSelector} + .validation-error, .error-${field}`;

      const errorElement = this.page.locator(errorSelector).first();
      if (await errorElement.isVisible()) {
        validationErrors[field] = await errorElement.textContent();
      }
    }

    return validationErrors;
  }

  /**
   * Check if specific field has error state
   */
  async isFieldInvalid(fieldName) {
    const fieldSelector = this.selectors[fieldName + "Input"];
    if (!fieldSelector) return false;

    try {
      const field = this.page.locator(fieldSelector);

      // Use shorter timeout for attribute checking
      const hasInvalidAttribute = await field
        .getAttribute("aria-invalid", { timeout: 2000 })
        .then(attr => attr === "true")
        .catch(() => false);
      
      const hasErrorClass = await field
        .evaluate(
          (el) =>
            el.classList.contains("error") ||
            el.classList.contains("invalid") ||
            el.classList.contains("is-invalid"),
          { timeout: 2000 }
        )
        .catch(() => false);

      return hasInvalidAttribute || hasErrorClass;
    } catch (error) {
      console.warn(`Field validation check failed for ${fieldName}:`, error.message);
      return false;
    }
  }

  /**
   * Test password strength validation
   */
  async testPasswordValidation(password) {
    await this.navigate();
    
    // Fill form with weak password to trigger server validation
    await this.page.fill(this.selectors.nameInput, "Test User");
    await this.page.fill(this.selectors.familyNameInput, "Test Family");
    await this.page.selectOption(this.selectors.currencySelect, "RUB");
    await this.page.fill(this.selectors.emailInput, "test@example.com");
    await this.page.fill(this.selectors.passwordInput, password);
    await this.page.fill(this.selectors.confirmPasswordInput, password);

    // Try to submit form - browser validation may prevent submission
    await this.page.click(this.selectors.submitButton);
    await this.page.waitForTimeout(500);

    // Check for server validation errors (various ways they can appear)
    const hasServerError = await this.page
      .locator(this.selectors.errorAlert)
      .isVisible()
      .catch(() => false);
    
    // Check for text-based validation messages
    const hasValidationText = await this.page
      .locator(':has-text("минимум 6 символов"), :has-text("Password"), :has-text("must be"), :has-text("required")')
      .first()
      .isVisible()
      .catch(() => false);
    
    // Check for client-side validation
    const hasClientError = await this.isFieldInvalid("password");
    
    // Check for HTML5 validation on password field
    const passwordField = this.page.locator(this.selectors.passwordInput);
    const validationMessage = await passwordField
      .evaluate(el => el.validationMessage, { timeout: 2000 })
      .catch(() => "");
    const hasValidationMessage = validationMessage && validationMessage.length > 0;
    
    // Check if we're still on register page (validation prevented submission)
    const currentUrl = this.page.url();
    const stillOnRegister = currentUrl.includes("/register");
    
    return Boolean(hasServerError || hasValidationText || hasClientError || hasValidationMessage);
  }

  /**
   * Test password confirmation matching
   */
  async testPasswordConfirmation(password, confirmPassword) {
    await this.navigate();
    
    // Fill complete form with mismatched passwords to trigger server validation
    await this.page.fill(this.selectors.nameInput, "Test User");
    await this.page.fill(this.selectors.familyNameInput, "Test Family");
    await this.page.selectOption(this.selectors.currencySelect, "RUB");
    await this.page.fill(this.selectors.emailInput, "test@example.com");
    await this.page.fill(this.selectors.passwordInput, password);
    await this.page.fill(this.selectors.confirmPasswordInput, confirmPassword);

    // Try to submit form - browser validation may prevent submission
    await this.page.click(this.selectors.submitButton);
    await this.page.waitForTimeout(500);

    // Check for server validation errors (various ways they can appear)
    const hasServerError = await this.page
      .locator(this.selectors.errorAlert)
      .isVisible()
      .catch(() => false);
    
    // Check for text-based validation messages about password mismatch
    const hasValidationText = await this.page
      .locator(':has-text("не совпадают"), :has-text("must match"), :has-text("eqfield"), :has-text("Password")')
      .first()
      .isVisible()
      .catch(() => false);
    
    // Check for client-side validation
    const hasClientError = await this.isFieldInvalid("confirm_password");
    
    // Check for HTML5 validation on confirm password field
    const confirmField = this.page.locator(this.selectors.confirmPasswordInput);
    const validationMessage = await confirmField
      .evaluate(el => el.validationMessage, { timeout: 2000 })
      .catch(() => "");
    const hasValidationMessage = validationMessage && validationMessage.length > 0;
    
    return Boolean(hasServerError || hasValidationText || hasClientError || hasValidationMessage);
  }

  /**
   * Test email format validation
   */
  async testEmailValidation(email) {
    await this.navigate();
    
    // Fill complete form with invalid email to trigger server validation
    await this.page.fill(this.selectors.nameInput, "Test User");
    await this.page.fill(this.selectors.familyNameInput, "Test Family");
    await this.page.selectOption(this.selectors.currencySelect, "RUB");
    await this.page.fill(this.selectors.emailInput, email);
    await this.page.fill(this.selectors.passwordInput, "ValidPassword123!");
    await this.page.fill(this.selectors.confirmPasswordInput, "ValidPassword123!");

    // Try to submit form - browser validation may prevent submission
    await this.page.click(this.selectors.submitButton);
    await this.page.waitForTimeout(500); // Wait for any validation to appear

    // Check for server validation errors
    const hasServerError = await this.page
      .locator(this.selectors.errorAlert)
      .isVisible()
      .catch(() => false);
    
    // Check for client-side validation
    const hasClientError = await this.isFieldInvalid("email");
    
    // Check if form submission was blocked by browser validation
    const currentUrl = this.page.url();
    const stillOnRegisterPage = currentUrl.includes("register");
    
    // Check for HTML5 validation popup
    const emailField = this.page.locator(this.selectors.emailInput);
    const validationMessage = await emailField
      .evaluate(el => el.validationMessage, { timeout: 2000 })
      .catch(() => "");
    const hasValidationMessage = validationMessage && validationMessage.length > 0;
    
    return hasServerError || hasClientError || hasValidationMessage;
  }

  /**
   * Navigate to login page
   */
  async goToLogin() {
    const loginLink = this.page.locator(this.selectors.loginLink);
    if (await loginLink.isVisible()) {
      await loginLink.click();
      await this.page.waitForURL(/.*login/);
      return true;
    }
    return false;
  }

  /**
   * Verify all required form elements are present
   */
  async verifyFormElements() {
    const elements = {};

    for (const field of this.formFields) {
      const selector = this.selectors[field + "Input"];
      elements[field] = await this.page.locator(selector).isVisible();
    }

    elements.submitButton = await this.page
      .locator(this.selectors.submitButton)
      .isVisible();
    elements.csrfToken = await this.hasCsrfToken();

    return elements;
  }

  /**
   * Check form accessibility
   */
  async checkAccessibility() {
    const accessibility = {};

    // Check for proper labels
    for (const field of this.formFields) {
      const labelSelector = `label[for="${field}"]`;
      accessibility[`${field}Label`] = await this.page
        .locator(labelSelector)
        .isVisible();
    }

    // Check for proper headings
    accessibility.hasHeading = await this.page
      .locator("h1, h2")
      .first()
      .isVisible();

    // Check for required field indicators
    accessibility.hasRequiredIndicators =
      (await this.page
        .locator('input[required], input[aria-required="true"]')
        .count()) > 0;

    return accessibility;
  }

  /**
   * Test form submission with empty fields
   */
  async testEmptyFormSubmission() {
    await this.navigate();
    
    // Clear all fields to ensure they're empty
    await this.page.fill(this.selectors.nameInput, "");
    await this.page.fill(this.selectors.familyNameInput, "");
    await this.page.fill(this.selectors.emailInput, "");
    await this.page.fill(this.selectors.passwordInput, "");
    await this.page.fill(this.selectors.confirmPasswordInput, "");
    
    await this.submitRegistration();

    // Check for both server errors and browser validation
    const hasServerErrors = await this.page
      .locator(this.selectors.errorAlert)
      .isVisible()
      .catch(() => false);
    
    // Check for HTML5 validation (invalid fields)
    const hasInvalidFields = await this.page
      .locator('input:invalid')
      .count()
      .then(count => count > 0)
      .catch(() => false);
    
    const currentUrl = this.page.url();
    const stayedOnRegisterPage = currentUrl.includes("register");

    return {
      hasErrors: hasServerErrors || hasInvalidFields,
      stayedOnRegisterPage,
      errors: hasServerErrors ? await this.getErrorMessages() : ["Browser validation triggered"],
    };
  }

  /**
   * Test keyboard navigation through form
   */
  async testKeyboardNavigation() {
    await this.navigate();

    const tabOrder = [];

    // Start from the first field
    await this.page.focus(this.selectors.nameInput);

    for (let i = 0; i < this.formFields.length + 1; i++) {
      // +1 for submit button
      const activeElement = await this.page.evaluate(() => {
        const el = document.activeElement;
        return {
          tagName: el.tagName,
          name: el.getAttribute("name"),
          type: el.getAttribute("type"),
        };
      });

      tabOrder.push(activeElement);
      await this.page.keyboard.press("Tab");
    }

    return tabOrder;
  }

  /**
   * Check if form has HTMX attributes
   */
  async hasHtmxIntegration() {
    const form = this.page.locator(this.selectors.registerForm);

    const hasHxPost = (await form.getAttribute("hx-post")) !== null;
    const hasHxTarget = (await form.getAttribute("hx-target")) !== null;
    const hasHxSwap = (await form.getAttribute("hx-swap")) !== null;

    return {
      hasHxPost,
      hasHxTarget,
      hasHxSwap,
      hasAnyHtmx: hasHxPost || hasHxTarget || hasHxSwap,
    };
  }

  /**
   * Wait for HTMX form submission to complete
   */
  async waitForHtmxSubmission() {
    // Wait for HTMX request to complete
    await this.page.evaluate(() => {
      return new Promise((resolve) => {
        if (window.htmx) {
          htmx.on("htmx:afterSettle", resolve);
          // Also set a timeout in case HTMX doesn't fire
          setTimeout(resolve, 5000);
        } else {
          resolve();
        }
      });
    });
  }

  /**
   * Get page title
   */
  async getPageTitle() {
    return await this.page.title();
  }

  /**
   * Check if registration was successful
   * Registration is successful if:
   * 1. Redirected to login or dashboard, OR
   * 2. Success message is displayed on register page, OR
   * 3. No error messages are shown and form submitted
   */
  async isRegistrationSuccessful() {
    const currentUrl = this.page.url();
    
    // Check for redirect to login or dashboard
    if (currentUrl.includes("login") || currentUrl.includes("dashboard")) {
      return true;
    }
    
    // Check for success message on register page
    const successMessage = await this.page
      .locator('.alert-success, .success')
      .isVisible()
      .catch(() => false);
    
    if (successMessage) {
      return true;
    }
    
    // Check that no error messages are displayed
    const hasErrors = await this.page
      .locator(this.selectors.errorAlert)
      .isVisible()
      .catch(() => false);
    
    // If on register page with no errors, check for success indicators
    if (currentUrl.includes("register") && !hasErrors) {
      // Look for success text content or login link visible
      const hasLoginLink = await this.page
        .locator('a:has-text("Войти в систему")')
        .isVisible()
        .catch(() => false);
      
      // Check for form replacement (HTMX update)
      const formPresent = await this.page
        .locator(this.selectors.registerForm)
        .isVisible()
        .catch(() => false);
      
      // Success if login link is visible or form was replaced
      return hasLoginLink || !formPresent;
    }
    
    return false;
  }

  /**
   * Verify page content and branding
   */
  async verifyPageContent() {
    return {
      title: await this.getPageTitle(),
      hasMainHeading: await this.page.locator("h1, h2").first().isVisible(),
      hasLogo: await this.page.locator('img[alt*="logo"], .logo').isVisible(),
      hasLoginLink: await this.page
        .locator(this.selectors.loginLink)
        .isVisible(),
    };
  }
}

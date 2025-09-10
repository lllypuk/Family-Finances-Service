/**
 * UX Testing Utilities
 * Provides comprehensive user experience testing tools for E2E tests
 */

export class UXTestingUtils {
  constructor(page) {
    this.page = page;
    this.accessibilityIssues = [];
    this.usabilityIssues = [];
  }

  /**
   * Test accessibility compliance
   */
  async testAccessibility() {
    const issues = [];

    try {
      // Check for basic accessibility requirements
      const accessibilityChecks = await this.page.evaluate(() => {
        const results = {
          missingAltText: [],
          missingLabels: [],
          lowContrastElements: [],
          missingHeadings: [],
          focusableElements: [],
          ariaIssues: [],
        };

        // Check images without alt text
        const images = document.querySelectorAll("img");
        images.forEach((img, index) => {
          if (!img.getAttribute("alt") && img.getAttribute("alt") !== "") {
            results.missingAltText.push({
              element: "img",
              index: index,
              src: img.src,
            });
          }
        });

        // Check form inputs without labels
        const inputs = document.querySelectorAll(
          'input:not([type="hidden"]), textarea, select',
        );
        inputs.forEach((input, index) => {
          const id = input.id;
          const hasLabel = id && document.querySelector(`label[for="${id}"]`);
          const hasAriaLabel = input.getAttribute("aria-label");
          const hasPlaceholder = input.placeholder;

          if (!hasLabel && !hasAriaLabel && !hasPlaceholder) {
            results.missingLabels.push({
              element: input.tagName.toLowerCase(),
              index: index,
              type: input.type || "text",
            });
          }
        });

        // Check heading structure
        const headings = document.querySelectorAll("h1, h2, h3, h4, h5, h6");
        if (headings.length === 0) {
          results.missingHeadings.push("No headings found on page");
        } else {
          let previousLevel = 0;
          headings.forEach((heading, index) => {
            const level = parseInt(heading.tagName.charAt(1));
            if (level > previousLevel + 1 && index > 0) {
              results.missingHeadings.push({
                issue: "Heading level skipped",
                element: heading.tagName,
                index: index,
                text: heading.textContent.substring(0, 50),
              });
            }
            previousLevel = level;
          });
        }

        // Check focusable elements
        const focusableElements = document.querySelectorAll(
          'a[href], button, input, select, textarea, [tabindex]:not([tabindex="-1"])',
        );

        focusableElements.forEach((element, index) => {
          const tabIndex = element.getAttribute("tabindex");
          if (tabIndex && parseInt(tabIndex) > 0) {
            results.focusableElements.push({
              issue: "Positive tabindex found (should use 0 or -1)",
              element: element.tagName.toLowerCase(),
              index: index,
              tabindex: tabIndex,
            });
          }
        });

        // Check ARIA attributes
        const ariaElements = document.querySelectorAll(
          "[aria-expanded], [aria-hidden], [role]",
        );
        ariaElements.forEach((element, index) => {
          const ariaExpanded = element.getAttribute("aria-expanded");
          const ariaHidden = element.getAttribute("aria-hidden");
          const role = element.getAttribute("role");

          if (ariaExpanded && !["true", "false"].includes(ariaExpanded)) {
            results.ariaIssues.push({
              issue: "Invalid aria-expanded value",
              element: element.tagName.toLowerCase(),
              index: index,
              value: ariaExpanded,
            });
          }

          if (ariaHidden && !["true", "false"].includes(ariaHidden)) {
            results.ariaIssues.push({
              issue: "Invalid aria-hidden value",
              element: element.tagName.toLowerCase(),
              index: index,
              value: ariaHidden,
            });
          }
        });

        return results;
      });

      this.accessibilityIssues = accessibilityChecks;
      return accessibilityChecks;
    } catch (error) {
      issues.push(`Accessibility check failed: ${error.message}`);
      return { error: error.message, issues };
    }
  }

  /**
   * Test keyboard navigation
   */
  async testKeyboardNavigation() {
    const results = {
      focusableElements: [],
      trapFocus: true,
      skipLinks: false,
      escapeWorks: true,
      issues: [],
    };

    try {
      // Get all focusable elements
      const focusableElements = await this.page.evaluate(() => {
        const elements = Array.from(
          document.querySelectorAll(
            'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
          ),
        );

        return elements.map((el, index) => ({
          tagName: el.tagName.toLowerCase(),
          type: el.type || null,
          id: el.id || null,
          text: el.textContent.trim().substring(0, 50),
          tabIndex: el.tabIndex,
          index: index,
        }));
      });

      results.focusableElements = focusableElements;

      // Test tab navigation
      for (let i = 0; i < Math.min(focusableElements.length, 10); i++) {
        await this.page.keyboard.press("Tab");

        const focusedElement = await this.page.evaluate(() => {
          const focused = document.activeElement;
          return {
            tagName: focused.tagName.toLowerCase(),
            id: focused.id || null,
            text: focused.textContent?.trim().substring(0, 50) || "",
          };
        });

        // Verify focus is visible
        const isFocusVisible = await this.page.evaluate(() => {
          const focused = document.activeElement;
          const computedStyle = window.getComputedStyle(focused);
          return (
            computedStyle.outline !== "none" && computedStyle.outline !== "0px"
          );
        });

        if (!isFocusVisible) {
          results.issues.push(
            `Focus not visible on element: ${focusedElement.tagName}`,
          );
        }
      }

      // Test escape key functionality (if modal/dialog is present)
      const hasModal = await this.page
        .locator('.modal, .dialog, [role="dialog"]')
        .isVisible();
      if (hasModal) {
        await this.page.keyboard.press("Escape");
        const modalStillVisible = await this.page
          .locator('.modal, .dialog, [role="dialog"]')
          .isVisible();
        results.escapeWorks = !modalStillVisible;
        if (modalStillVisible) {
          results.issues.push("Modal/dialog does not close with Escape key");
        }
      }

      // Check for skip links
      const skipLinks = await this.page.evaluate(() => {
        const links = document.querySelectorAll('a[href^="#"]');
        return Array.from(links).some(
          (link) =>
            link.textContent.toLowerCase().includes("skip") ||
            link.textContent.toLowerCase().includes("jump"),
        );
      });

      results.skipLinks = skipLinks;
      if (!skipLinks && focusableElements.length > 5) {
        results.issues.push(
          "Consider adding skip links for better keyboard navigation",
        );
      }

      return results;
    } catch (error) {
      results.issues.push(`Keyboard navigation test failed: ${error.message}`);
      return results;
    }
  }

  /**
   * Test responsive design across different viewports
   */
  async testResponsiveDesign() {
    const viewports = [
      { name: "mobile", width: 375, height: 667 },
      { name: "tablet", width: 768, height: 1024 },
      { name: "desktop", width: 1280, height: 720 },
      { name: "large", width: 1920, height: 1080 },
    ];

    const results = {};

    for (const viewport of viewports) {
      await this.page.setViewportSize(viewport);
      await this.page.waitForTimeout(1000); // Wait for layout to settle

      const viewportResult = await this.page.evaluate(() => {
        const result = {
          hasHorizontalScroll:
            document.documentElement.scrollWidth > window.innerWidth,
          hasOverflowElements: [],
          hiddenElements: [],
          contentVisible: true,
          navigationAccessible: false,
        };

        // Check for elements that cause horizontal scroll
        const allElements = document.querySelectorAll("*");
        allElements.forEach((element, index) => {
          const rect = element.getBoundingClientRect();
          if (rect.right > window.innerWidth) {
            result.hasOverflowElements.push({
              element: element.tagName.toLowerCase(),
              class: element.className,
              index: index,
            });
          }
        });

        // Check if main navigation is accessible
        const nav = document.querySelector("nav, .navigation, .navbar");
        if (nav) {
          const navRect = nav.getBoundingClientRect();
          result.navigationAccessible = navRect.width > 0 && navRect.height > 0;
        }

        // Check if main content is visible
        const main = document.querySelector("main, .main-content, .content");
        if (main) {
          const mainRect = main.getBoundingClientRect();
          result.contentVisible = mainRect.width > 0 && mainRect.height > 0;
        }

        return result;
      });

      results[viewport.name] = {
        ...viewport,
        ...viewportResult,
      };
    }

    // Reset to default viewport
    await this.page.setViewportSize({ width: 1280, height: 720 });

    return results;
  }

  /**
   * Test form usability
   */
  async testFormUsability(formSelector) {
    const results = {
      hasLabels: true,
      hasValidation: false,
      hasErrorMessages: false,
      hasSuccessMessages: false,
      fieldTypes: [],
      accessibilityScore: 0,
      issues: [],
    };

    try {
      const formExists = await this.page.locator(formSelector).isVisible();
      if (!formExists) {
        results.issues.push("Form not found or not visible");
        return results;
      }

      const formAnalysis = await this.page.evaluate((selector) => {
        const form = document.querySelector(selector);
        if (!form) return null;

        const analysis = {
          inputs: [],
          hasLabels: 0,
          totalInputs: 0,
          hasValidation: false,
          hasErrorElements: false,
          hasSuccessElements: false,
        };

        const inputs = form.querySelectorAll(
          'input:not([type="hidden"]), textarea, select',
        );
        analysis.totalInputs = inputs.length;

        inputs.forEach((input, index) => {
          const inputData = {
            type: input.type || input.tagName.toLowerCase(),
            hasLabel: false,
            hasPlaceholder: !!input.placeholder,
            hasAriaLabel: !!input.getAttribute("aria-label"),
            required: input.hasAttribute("required"),
            id: input.id,
            name: input.name,
          };

          // Check for associated label - try multiple methods
          let hasLabel = false;

          if (input.id) {
            const label = document.querySelector(`label[for="${input.id}"]`);
            hasLabel = !!label;
          }

          // Also check if input is wrapped in a label
          if (
            !hasLabel &&
            input.parentElement &&
            input.parentElement.tagName === "LABEL"
          ) {
            hasLabel = true;
          }

          // Check for preceding label element
          if (
            !hasLabel &&
            input.previousElementSibling &&
            input.previousElementSibling.tagName === "LABEL"
          ) {
            hasLabel = true;
          }

          inputData.hasLabel = hasLabel;

          if (
            inputData.hasLabel ||
            inputData.hasAriaLabel ||
            inputData.hasPlaceholder
          ) {
            analysis.hasLabels++;
          }

          analysis.inputs.push(inputData);
        });

        // Check for validation indicators
        const validationElements = form.querySelectorAll(
          ".error, .invalid, [aria-invalid], .validation-error",
        );
        analysis.hasValidation = validationElements.length > 0;

        // Check for error message areas
        const errorElements = form.querySelectorAll(
          ".error-message, .error, .alert-error",
        );
        analysis.hasErrorElements = errorElements.length > 0;

        // Check for success message areas
        const successElements = form.querySelectorAll(
          ".success-message, .success, .alert-success",
        );
        analysis.hasSuccessElements = successElements.length > 0;

        return analysis;
      }, formSelector);

      if (formAnalysis) {
        results.hasLabels = formAnalysis.hasLabels === formAnalysis.totalInputs;
        results.hasValidation = formAnalysis.hasValidation;
        results.hasErrorMessages = formAnalysis.hasErrorElements;
        results.hasSuccessMessages = formAnalysis.hasSuccessElements;
        results.fieldTypes = formAnalysis.inputs.map((i) => i.type);

        // Calculate accessibility score
        let score = 0;
        if (results.hasLabels) score += 30;
        if (results.hasValidation) score += 25;
        if (results.hasErrorMessages) score += 25;
        if (results.hasSuccessMessages) score += 20;
        results.accessibilityScore = score;

        // Generate issues
        if (!results.hasLabels) {
          results.issues.push(
            `${formAnalysis.totalInputs - formAnalysis.hasLabels} form fields missing labels`,
          );
        }
        if (!results.hasValidation) {
          results.issues.push("Form lacks validation indicators");
        }
        if (!results.hasErrorMessages) {
          results.issues.push("Form lacks error message areas");
        }
      }

      return results;
    } catch (error) {
      results.issues.push(`Form usability test failed: ${error.message}`);
      return results;
    }
  }

  /**
   * Test loading states and feedback
   */
  async testLoadingStates() {
    const results = {
      hasLoadingIndicators: false,
      loadingIndicatorTypes: [],
      feedbackMessages: [],
      issues: [],
    };

    try {
      // Check for loading indicators
      const loadingElements = await this.page.evaluate(() => {
        const indicators = [];

        // Common loading indicator selectors
        const selectors = [
          ".loading",
          ".spinner",
          ".loader",
          '[aria-busy="true"]',
          ".progress",
          ".loading-spinner",
          ".loading-indicator",
        ];

        selectors.forEach((selector) => {
          const elements = document.querySelectorAll(selector);
          elements.forEach((element) => {
            indicators.push({
              selector: selector,
              visible: element.offsetHeight > 0 && element.offsetWidth > 0,
              text: element.textContent.trim(),
              ariaLabel: element.getAttribute("aria-label"),
            });
          });
        });

        return indicators;
      });

      results.hasLoadingIndicators = loadingElements.length > 0;
      results.loadingIndicatorTypes = [
        ...new Set(loadingElements.map((el) => el.selector)),
      ];

      // Check for feedback messages
      const feedbackElements = await this.page.evaluate(() => {
        const feedback = [];

        const selectors = [
          ".alert",
          ".message",
          ".notification",
          ".success",
          ".error",
          ".warning",
          ".info",
          ".toast",
          ".banner",
          ".feedback",
        ];

        selectors.forEach((selector) => {
          const elements = document.querySelectorAll(selector);
          elements.forEach((element) => {
            feedback.push({
              type: selector.replace(".", ""),
              text: element.textContent.trim().substring(0, 100),
              visible: element.offsetHeight > 0 && element.offsetWidth > 0,
            });
          });
        });

        return feedback;
      });

      results.feedbackMessages = feedbackElements;

      // Generate recommendations
      if (!results.hasLoadingIndicators) {
        results.issues.push(
          "No loading indicators found - users may not know when operations are in progress",
        );
      }

      if (results.feedbackMessages.length === 0) {
        results.issues.push(
          "No feedback message areas found - users may not receive operation results",
        );
      }

      return results;
    } catch (error) {
      results.issues.push(`Loading states test failed: ${error.message}`);
      return results;
    }
  }

  /**
   * Test error handling and recovery
   */
  async testErrorHandling() {
    const results = {
      errorPagesExist: false,
      gracefulDegradation: true,
      errorRecovery: false,
      issues: [],
    };

    try {
      // Test 404 error page
      const current404Response = await this.page.goto(
        "/nonexistent-page-12345",
        { waitUntil: "networkidle" },
      );
      const is404Handled = current404Response.status() === 404;

      if (is404Handled) {
        const has404Content = await this.page.evaluate(() => {
          const body = document.body.textContent.toLowerCase();
          return (
            body.includes("not found") ||
            body.includes("404") ||
            body.includes("page not found") ||
            body.includes("страница не найдена") ||
            body.includes("page not found") ||
            body.includes("error") ||
            body.includes("ошибка")
          );
        });

        results.errorPagesExist = has404Content;

        if (!has404Content) {
          results.issues.push(
            "404 error page lacks user-friendly error message",
          );
        }
      } else {
        // If not 404, check if we're redirected to a valid page (like login)
        const isRedirectToValidPage =
          current404Response.status() >= 200 &&
          current404Response.status() < 400;
        results.errorPagesExist = isRedirectToValidPage; // Redirect is acceptable behavior
        results.gracefulDegradation = isRedirectToValidPage;

        if (isRedirectToValidPage) {
          console.log(
            `404 page redirected to valid page (${current404Response.status()})`,
          );
        }
      }

      // Go back to a valid page
      await this.page.goBack();

      return results;
    } catch (error) {
      results.issues.push(`Error handling test failed: ${error.message}`);
      return results;
    }
  }

  /**
   * Test color contrast
   */
  async testColorContrast() {
    const contrastIssues = await this.page.evaluate(() => {
      const issues = [];

      // Get all text elements
      const textElements = document.querySelectorAll("*");

      Array.from(textElements)
        .slice(0, 50)
        .forEach((element, index) => {
          if (element.children.length === 0 && element.textContent.trim()) {
            const computedStyle = window.getComputedStyle(element);
            const textColor = computedStyle.color;
            const backgroundColor = computedStyle.backgroundColor;

            // Simple heuristic for potential contrast issues
            if (
              textColor &&
              backgroundColor &&
              textColor !== "rgba(0, 0, 0, 0)" &&
              backgroundColor !== "rgba(0, 0, 0, 0)"
            ) {
              // This is a simplified check - in production, you'd use a proper contrast ratio calculator
              if (
                textColor.includes("rgb(128") ||
                backgroundColor.includes("rgb(128")
              ) {
                issues.push({
                  element: element.tagName.toLowerCase(),
                  index: index,
                  text: element.textContent.trim().substring(0, 50),
                  textColor: textColor,
                  backgroundColor: backgroundColor,
                  warning: "Potential low contrast detected",
                });
              }
            }
          }
        });

      return issues;
    });

    return {
      issues: contrastIssues,
      totalChecked: Math.min(50, contrastIssues.length),
      recommendation:
        contrastIssues.length > 0
          ? "Consider using a contrast checking tool for detailed analysis"
          : "No obvious contrast issues detected in sample",
    };
  }

  /**
   * Generate comprehensive UX report
   */
  async generateUXReport() {
    const report = {
      timestamp: new Date().toISOString(),
      url: this.page.url(),
      tests: {},
    };

    console.log("Running accessibility tests...");
    report.tests.accessibility = await this.testAccessibility();

    console.log("Running keyboard navigation tests...");
    report.tests.keyboardNavigation = await this.testKeyboardNavigation();

    console.log("Running responsive design tests...");
    report.tests.responsiveDesign = await this.testResponsiveDesign();

    console.log("Running loading states tests...");
    report.tests.loadingStates = await this.testLoadingStates();

    console.log("Running error handling tests...");
    report.tests.errorHandling = await this.testErrorHandling();

    console.log("Running color contrast tests...");
    report.tests.colorContrast = await this.testColorContrast();

    // Calculate overall score
    let totalScore = 0;
    let maxScore = 0;

    // Accessibility score
    const accessibilityScore = this.calculateAccessibilityScore(
      report.tests.accessibility,
    );
    totalScore += accessibilityScore;
    maxScore += 100;

    // Keyboard navigation score
    const keyboardScore =
      report.tests.keyboardNavigation.issues.length === 0
        ? 100
        : Math.max(0, 100 - report.tests.keyboardNavigation.issues.length * 20);
    totalScore += keyboardScore;
    maxScore += 100;

    // Responsive design score
    const responsiveScore = this.calculateResponsiveScore(
      report.tests.responsiveDesign,
    );
    totalScore += responsiveScore;
    maxScore += 100;

    report.overallScore = Math.round((totalScore / maxScore) * 100);
    report.recommendations = this.generateUXRecommendations(report.tests);

    return report;
  }

  /**
   * Calculate accessibility score
   */
  calculateAccessibilityScore(accessibilityResults) {
    if (accessibilityResults.error) return 0;

    let score = 100;
    score -= accessibilityResults.missingAltText.length * 5;
    score -= accessibilityResults.missingLabels.length * 10;
    score -= accessibilityResults.missingHeadings.length * 5;
    score -= accessibilityResults.ariaIssues.length * 8;
    score -= accessibilityResults.focusableElements.length * 3;

    return Math.max(0, score);
  }

  /**
   * Calculate responsive design score
   */
  calculateResponsiveScore(responsiveResults) {
    let score = 100;

    Object.values(responsiveResults).forEach((viewport) => {
      if (viewport.hasHorizontalScroll) score -= 10;
      if (!viewport.navigationAccessible) score -= 15;
      if (!viewport.contentVisible) score -= 20;
      score -= Math.min(viewport.hasOverflowElements?.length * 2 || 0, 10);
    });

    return Math.max(0, score);
  }

  /**
   * Generate UX recommendations
   */
  generateUXRecommendations(tests) {
    const recommendations = [];

    // Accessibility recommendations
    if (tests.accessibility.missingAltText?.length > 0) {
      recommendations.push(
        `Add alt text to ${tests.accessibility.missingAltText.length} images`,
      );
    }
    if (tests.accessibility.missingLabels?.length > 0) {
      recommendations.push(
        `Add labels to ${tests.accessibility.missingLabels.length} form fields`,
      );
    }

    // Keyboard navigation recommendations
    if (tests.keyboardNavigation.issues.length > 0) {
      recommendations.push(
        "Improve keyboard navigation based on identified issues",
      );
    }

    // Responsive design recommendations
    const responsiveIssues = Object.values(tests.responsiveDesign).filter(
      (v) =>
        v.hasHorizontalScroll || !v.navigationAccessible || !v.contentVisible,
    );
    if (responsiveIssues.length > 0) {
      recommendations.push(
        "Fix responsive design issues across different viewports",
      );
    }

    // Loading states recommendations
    if (!tests.loadingStates.hasLoadingIndicators) {
      recommendations.push("Add loading indicators for better user feedback");
    }

    // Error handling recommendations
    if (!tests.errorHandling.errorPagesExist) {
      recommendations.push("Implement user-friendly error pages");
    }

    if (recommendations.length === 0) {
      recommendations.push(
        "UX quality is good - continue monitoring and testing",
      );
    }

    return recommendations;
  }
}

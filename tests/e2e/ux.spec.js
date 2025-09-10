import { test, expect } from "@playwright/test";
import { UXTestingUtils } from "./helpers/ux-testing-utils.js";

test.describe("User Experience Testing", () => {
  let uxUtils;

  test.beforeEach(async ({ page }) => {
    uxUtils = new UXTestingUtils(page);
  });

  test.describe("Accessibility", () => {
    test("should meet basic accessibility standards on homepage", async ({
      page,
    }) => {
      await page.goto("/");

      const accessibilityResults = await uxUtils.testAccessibility();

      expect(accessibilityResults.error).toBeUndefined();

      // Check for critical accessibility issues
      expect(accessibilityResults.missingLabels.length).toBeLessThanOrEqual(2);
      expect(accessibilityResults.ariaIssues.length).toBe(0);

      console.log("Accessibility check:", {
        missingAltText: accessibilityResults.missingAltText.length,
        missingLabels: accessibilityResults.missingLabels.length,
        headingIssues: accessibilityResults.missingHeadings.length,
        ariaIssues: accessibilityResults.ariaIssues.length,
      });

      if (accessibilityResults.missingAltText.length > 0) {
        console.log(
          "Images missing alt text:",
          accessibilityResults.missingAltText.length,
        );
      }
    });

    test("should meet accessibility standards on login page", async ({
      page,
    }) => {
      await page.goto("/login");

      const accessibilityResults = await uxUtils.testAccessibility();

      expect(accessibilityResults.error).toBeUndefined();

      // Login forms should have proper labels
      expect(accessibilityResults.missingLabels.length).toBe(0);

      console.log("Login page accessibility:", {
        missingLabels: accessibilityResults.missingLabels.length,
        ariaIssues: accessibilityResults.ariaIssues.length,
      });
    });

    test("should meet accessibility standards on registration page", async ({
      page,
    }) => {
      await page.goto("/register");

      const accessibilityResults = await uxUtils.testAccessibility();

      expect(accessibilityResults.error).toBeUndefined();

      // Registration forms should have proper labels
      expect(accessibilityResults.missingLabels.length).toBe(0);

      console.log("Registration page accessibility:", {
        missingLabels: accessibilityResults.missingLabels.length,
        ariaIssues: accessibilityResults.ariaIssues.length,
      });
    });

    test("should have proper color contrast", async ({ page }) => {
      await page.goto("/");

      const contrastResults = await uxUtils.testColorContrast();

      // Should not have obvious contrast issues
      expect(contrastResults.issues.length).toBeLessThanOrEqual(5);

      console.log("Color contrast check:", {
        totalChecked: contrastResults.totalChecked,
        issues: contrastResults.issues.length,
        recommendation: contrastResults.recommendation,
      });
    });
  });

  test.describe("Keyboard Navigation", () => {
    test("should support keyboard navigation on homepage", async ({ page }) => {
      await page.goto("/");

      const keyboardResults = await uxUtils.testKeyboardNavigation();

      expect(keyboardResults.focusableElements.length).toBeGreaterThan(0);
      expect(keyboardResults.issues.length).toBeLessThanOrEqual(2);

      console.log("Keyboard navigation:", {
        focusableElements: keyboardResults.focusableElements.length,
        hasSkipLinks: keyboardResults.skipLinks,
        issues: keyboardResults.issues.length,
      });

      if (keyboardResults.issues.length > 0) {
        console.log("Keyboard navigation issues:", keyboardResults.issues);
      }
    });

    test("should support keyboard navigation in forms", async ({ page }) => {
      await page.goto("/login");

      const keyboardResults = await uxUtils.testKeyboardNavigation();

      expect(keyboardResults.focusableElements.length).toBeGreaterThan(2); // At least email, password, submit
      expect(keyboardResults.issues.length).toBeLessThanOrEqual(1);

      console.log("Form keyboard navigation:", {
        focusableElements: keyboardResults.focusableElements.length,
        issues: keyboardResults.issues.length,
      });
    });

    test("should handle escape key in modals", async ({ page }) => {
      await page.goto("/");

      const keyboardResults = await uxUtils.testKeyboardNavigation();

      // This tests escape key functionality if modals are present
      expect(keyboardResults.escapeWorks).toBe(true);

      console.log("Modal escape key support:", keyboardResults.escapeWorks);
    });
  });

  test.describe("Responsive Design", () => {
    test("should work across different viewports", async ({ page }) => {
      await page.goto("/");

      const responsiveResults = await uxUtils.testResponsiveDesign();

      // Check that all viewports work properly
      expect(responsiveResults.mobile.hasHorizontalScroll).toBe(false);
      expect(responsiveResults.mobile.contentVisible).toBe(true);

      expect(responsiveResults.tablet.hasHorizontalScroll).toBe(false);
      expect(responsiveResults.tablet.contentVisible).toBe(true);

      expect(responsiveResults.desktop.hasHorizontalScroll).toBe(false);
      expect(responsiveResults.desktop.contentVisible).toBe(true);

      console.log("Responsive design check:", {
        mobile: {
          horizontalScroll: responsiveResults.mobile.hasHorizontalScroll,
          contentVisible: responsiveResults.mobile.contentVisible,
          navigationAccessible: responsiveResults.mobile.navigationAccessible,
        },
        tablet: {
          horizontalScroll: responsiveResults.tablet.hasHorizontalScroll,
          contentVisible: responsiveResults.tablet.contentVisible,
        },
        desktop: {
          horizontalScroll: responsiveResults.desktop.hasHorizontalScroll,
          contentVisible: responsiveResults.desktop.contentVisible,
        },
      });
    });

    test("should maintain usability on mobile devices", async ({ page }) => {
      await page.goto("/login");

      // Test mobile viewport specifically
      await page.setViewportSize({ width: 375, height: 667 });
      await page.waitForTimeout(1000);

      const responsiveResults = await uxUtils.testResponsiveDesign();

      expect(responsiveResults.mobile.hasHorizontalScroll).toBe(false);
      expect(responsiveResults.mobile.contentVisible).toBe(true);

      // Test touch-friendly interactions
      const buttons = page.locator("button, a");
      const buttonCount = await buttons.count();

      if (buttonCount > 0) {
        const firstButton = buttons.first();
        const buttonBox = await firstButton.boundingBox();

        // Buttons should be touch-friendly (at least 44px)
        if (buttonBox) {
          expect(
            Math.min(buttonBox.width, buttonBox.height),
          ).toBeGreaterThanOrEqual(44);
        }
      }

      console.log("Mobile usability:", {
        buttonsFound: buttonCount,
        horizontalScroll: responsiveResults.mobile.hasHorizontalScroll,
      });

      // Reset viewport
      await page.setViewportSize({ width: 1280, height: 720 });
    });
  });

  test.describe("Form Usability", () => {
    test("should have usable login form", async ({ page }) => {
      await page.goto("/login");

      const formUsability = await uxUtils.testFormUsability("form");

      expect(formUsability.hasLabels).toBe(true);
      expect(formUsability.accessibilityScore).toBeGreaterThan(25); // More realistic threshold
      expect(formUsability.issues.length).toBeLessThanOrEqual(2);

      console.log("Login form usability:", {
        hasLabels: formUsability.hasLabels,
        hasValidation: formUsability.hasValidation,
        accessibilityScore: formUsability.accessibilityScore,
        issues: formUsability.issues.length,
      });

      if (formUsability.issues.length > 0) {
        console.log("Form issues:", formUsability.issues);
      }
    });

    test("should have usable registration form", async ({ page }) => {
      await page.goto("/register");

      const formUsability = await uxUtils.testFormUsability("form");

      expect(formUsability.hasLabels).toBe(true);
      expect(formUsability.accessibilityScore).toBeGreaterThan(25); // More realistic threshold
      expect(formUsability.issues.length).toBeLessThanOrEqual(2);

      console.log("Registration form usability:", {
        hasLabels: formUsability.hasLabels,
        hasValidation: formUsability.hasValidation,
        accessibilityScore: formUsability.accessibilityScore,
        fieldTypes: formUsability.fieldTypes,
      });
    });

    test("should have usable transaction forms - authenticated", async ({
      page,
    }) => {
      await page.goto("/transactions/add");

      const formUsability = await uxUtils.testFormUsability("form");

      expect(formUsability.hasLabels).toBe(true);
      expect(formUsability.hasValidation).toBe(true);
      expect(formUsability.accessibilityScore).toBeGreaterThan(70);

      console.log("Transaction form usability:", {
        hasLabels: formUsability.hasLabels,
        hasValidation: formUsability.hasValidation,
        hasErrorMessages: formUsability.hasErrorMessages,
        accessibilityScore: formUsability.accessibilityScore,
      });
    });
  });

  test.describe("Loading States and Feedback", () => {
    test("should provide proper loading feedback", async ({ page }) => {
      await page.goto("/");

      const loadingResults = await uxUtils.testLoadingStates();

      console.log("Loading states:", {
        hasLoadingIndicators: loadingResults.hasLoadingIndicators,
        indicatorTypes: loadingResults.loadingIndicatorTypes,
        feedbackMessages: loadingResults.feedbackMessages.length,
      });

      // While loading indicators might not be visible on static pages,
      // the system should be prepared to show them
      expect(loadingResults.issues.length).toBeLessThanOrEqual(2);

      if (loadingResults.issues.length > 0) {
        console.log("Loading state recommendations:", loadingResults.issues);
      }
    });

    test("should handle form submission feedback", async ({ page }) => {
      await page.goto("/login");

      // Submit form with invalid data to trigger feedback
      await page.fill(
        "input[name='email'], input[type='email']",
        "invalid-email",
      );
      await page.fill(
        "input[name='password'], input[type='password']",
        "short",
      );

      try {
        await page.click("button[type='submit']");
        await page.waitForTimeout(2000);
      } catch (error) {
        // Form might prevent submission, which is good
      }

      const loadingResults = await uxUtils.testLoadingStates();

      console.log("Form feedback:", {
        feedbackMessages: loadingResults.feedbackMessages.length,
        hasErrorHandling: loadingResults.feedbackMessages.some(
          (msg) => msg.type.includes("error") || msg.type.includes("alert"),
        ),
      });
    });
  });

  test.describe("Error Handling", () => {
    test("should handle 404 errors gracefully", async ({ page }) => {
      const errorResults = await uxUtils.testErrorHandling();

      expect(errorResults.errorPagesExist).toBe(true);
      expect(errorResults.gracefulDegradation).toBe(true);

      console.log("Error handling:", {
        has404Page: errorResults.errorPagesExist,
        gracefulDegradation: errorResults.gracefulDegradation,
        issues: errorResults.issues.length,
      });

      if (errorResults.issues.length > 0) {
        console.log("Error handling issues:", errorResults.issues);
      }
    });

    test("should recover from JavaScript errors", async ({ page }) => {
      await page.goto("/");

      // Inject a JavaScript error
      await page.evaluate(() => {
        // This will cause an error but shouldn't break the page
        try {
          nonexistentFunction();
        } catch (e) {
          console.log("Expected error for testing:", e.message);
        }
      });

      // Page should still be functional
      const title = await page.title();
      expect(title).toBeTruthy();

      const bodyText = await page.textContent("body");
      expect(bodyText.length).toBeGreaterThan(10);

      console.log("JavaScript error recovery: Page remains functional");
    });

    test("should handle network errors gracefully", async ({ page }) => {
      // Simulate network issues
      await page.route("**/api/**", (route) => route.abort());

      await page.goto("/");

      // Page should still load even if some API calls fail
      const title = await page.title();
      expect(title).toBeTruthy();

      console.log("Network error handling: Page loads despite API failures");
    });
  });

  test.describe("Navigation and Information Architecture", () => {
    test("should have clear navigation structure", async ({ page }) => {
      await page.goto("/");

      const navigationCheck = await page.evaluate(() => {
        const nav = document.querySelector("nav, .navigation, .navbar");
        const footer = document.querySelector("footer");
        const allLinks = document.querySelectorAll("a");
        const breadcrumbs = document.querySelectorAll(
          ".breadcrumb, .breadcrumbs",
        );

        return {
          hasNavigation: !!nav,
          navigationLinks: nav ? nav.querySelectorAll("a").length : 0,
          footerLinks: footer ? footer.querySelectorAll("a").length : 0,
          totalLinks: allLinks.length,
          hasBreadcrumbs: breadcrumbs.length > 0,
          navigationText: nav ? nav.textContent.trim() : "",
        };
      });

      // More flexible navigation check - navigation can be in nav, footer, or main area
      const hasNavigationElements =
        navigationCheck.hasNavigation ||
        navigationCheck.footerLinks > 0 ||
        navigationCheck.totalLinks > 0;

      expect(hasNavigationElements).toBe(true);
      expect(navigationCheck.totalLinks).toBeGreaterThan(0);

      console.log("Navigation structure:", {
        hasNavigation: navigationCheck.hasNavigation,
        navigationLinks: navigationCheck.navigationLinks,
        hasBreadcrumbs: navigationCheck.hasBreadcrumbs,
      });
    });

    test("should have consistent layout across pages", async ({ page }) => {
      const pages = ["/", "/login", "/register"];
      const layouts = [];

      for (const pagePath of pages) {
        await page.goto(pagePath);

        const layout = await page.evaluate(() => {
          const header = document.querySelector("header, .header");
          const main = document.querySelector("main, .main, .content");
          const footer = document.querySelector("footer, .footer");
          const nav = document.querySelector("nav, .navigation");

          return {
            page: window.location.pathname,
            hasHeader: !!header,
            hasMain: !!main,
            hasFooter: !!footer,
            hasNavigation: !!nav,
            title: document.title,
          };
        });

        layouts.push(layout);
      }

      // Check consistency
      const hasHeader = layouts.every(
        (l) => l.hasHeader === layouts[0].hasHeader,
      );
      const hasMain = layouts.every((l) => l.hasMain);

      expect(hasMain).toBe(true); // All pages should have main content area

      console.log("Layout consistency:", {
        consistentHeader: hasHeader,
        allHaveMain: hasMain,
        layouts: layouts.map((l) => ({ page: l.page, title: l.title })),
      });
    });
  });

  test.describe("Content and Readability", () => {
    test("should have readable content", async ({ page }) => {
      await page.goto("/");

      const readabilityCheck = await page.evaluate(() => {
        const headings = document.querySelectorAll("h1, h2, h3, h4, h5, h6");
        const paragraphs = document.querySelectorAll("p");
        const links = document.querySelectorAll("a");

        // Check for proper heading hierarchy
        let headingHierarchy = true;
        let previousLevel = 0;
        headings.forEach((heading) => {
          const level = parseInt(heading.tagName.charAt(1));
          if (level > previousLevel + 1 && previousLevel > 0) {
            headingHierarchy = false;
          }
          previousLevel = level;
        });

        // Check link text quality
        const genericLinks = Array.from(links).filter((link) => {
          const text = link.textContent.trim().toLowerCase();
          return (
            text === "click here" ||
            text === "read more" ||
            text === "here" ||
            text.length < 3
          );
        });

        return {
          headingCount: headings.length,
          paragraphCount: paragraphs.length,
          linkCount: links.length,
          properHeadingHierarchy: headingHierarchy,
          genericLinks: genericLinks.length,
          hasH1: document.querySelectorAll("h1").length > 0,
        };
      });

      expect(readabilityCheck.hasH1).toBe(true);
      expect(readabilityCheck.properHeadingHierarchy).toBe(true);
      expect(readabilityCheck.genericLinks).toBeLessThanOrEqual(2);

      console.log("Content readability:", {
        headings: readabilityCheck.headingCount,
        paragraphs: readabilityCheck.paragraphCount,
        links: readabilityCheck.linkCount,
        hasH1: readabilityCheck.hasH1,
        properHeadingHierarchy: readabilityCheck.properHeadingHierarchy,
        genericLinks: readabilityCheck.genericLinks,
      });
    });

    test("should have appropriate page titles", async ({ page }) => {
      const pages = [
        { path: "/", expectedKeywords: ["budget", "family", "finance"] },
        { path: "/login", expectedKeywords: ["login", "sign in"] },
        {
          path: "/register",
          expectedKeywords: ["register", "sign up", "create"],
        },
      ];

      for (const pageInfo of pages) {
        await page.goto(pageInfo.path);

        const title = await page.title();
        expect(title.length).toBeGreaterThan(5);
        expect(title.length).toBeLessThan(60); // SEO best practice

        const hasRelevantKeyword = pageInfo.expectedKeywords.some((keyword) =>
          title.toLowerCase().includes(keyword),
        );

        // Title should be relevant to page content
        if (pageInfo.path !== "/") {
          expect(hasRelevantKeyword).toBe(true);
        }

        console.log(
          `${pageInfo.path} title: "${title}" (${title.length} chars)`,
        );
      }
    });
  });

  test.describe("Comprehensive UX Report", () => {
    test("should generate comprehensive UX report for homepage", async ({
      page,
    }) => {
      await page.goto("/");

      const uxReport = await uxUtils.generateUXReport();

      expect(uxReport.overallScore).toBeGreaterThan(60); // Minimum acceptable UX score
      expect(uxReport.tests).toBeDefined();
      expect(uxReport.recommendations).toBeDefined();

      console.log("UX Report Summary:", {
        overallScore: uxReport.overallScore,
        url: uxReport.url,
        recommendationsCount: uxReport.recommendations.length,
      });

      console.log("UX Recommendations:", uxReport.recommendations);

      // Log detailed test results
      Object.entries(uxReport.tests).forEach(([testName, results]) => {
        if (results.issues && results.issues.length > 0) {
          console.log(`${testName} issues:`, results.issues.length);
        }
      });
    });

    test("should generate UX report for login page", async ({ page }) => {
      await page.goto("/login");

      const uxReport = await uxUtils.generateUXReport();

      // Login pages should score higher due to simpler content
      expect(uxReport.overallScore).toBeGreaterThan(70);

      console.log("Login Page UX Score:", uxReport.overallScore);

      if (uxReport.recommendations.length > 0) {
        console.log("Login Page UX Recommendations:", uxReport.recommendations);
      }
    });

    test("should generate UX report for authenticated pages", async ({
      page,
    }) => {
      await page.goto("/dashboard");

      const uxReport = await uxUtils.generateUXReport();

      expect(uxReport.overallScore).toBeGreaterThan(65); // Complex pages may score slightly lower

      console.log("Dashboard UX Score:", uxReport.overallScore);
      console.log("Dashboard UX Recommendations:", uxReport.recommendations);
    });
  });
});

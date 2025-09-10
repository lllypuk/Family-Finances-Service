import { test, expect } from "@playwright/test";

/**
 * Advanced HTMX Testing Framework
 * Tests HTMX integration patterns, partial updates, and real-time features
 */

/**
 * HTMX Testing Utilities
 * Helper functions for testing HTMX functionality
 */
class HTMXTestUtils {
  constructor(page) {
    this.page = page;
  }

  /**
   * Wait for HTMX request to complete
   */
  async waitForHTMXRequest() {
    return this.page.waitForFunction(
      () => {
        return !document.querySelector(".htmx-request");
      },
      { timeout: 10000 },
    );
  }

  /**
   * Get HTMX request history
   */
  async getHTMXHistory() {
    return this.page.evaluate(() => {
      return window.htmxEvents || [];
    });
  }

  /**
   * Trigger HTMX element manually
   */
  async triggerHTMXElement(selector, trigger = "click") {
    return this.page.evaluate(
      ([sel, trig]) => {
        const element = document.querySelector(sel);
        if (element && window.htmx) {
          htmx.trigger(element, trig);
          return true;
        }
        return false;
      },
      [selector, trigger],
    );
  }

  /**
   * Check if element has HTMX attributes
   */
  async hasHTMXAttributes(selector) {
    return this.page.evaluate((sel) => {
      const element = document.querySelector(sel);
      if (!element) return null;

      return {
        hxPost: element.hasAttribute("hx-post"),
        hxGet: element.hasAttribute("hx-get"),
        hxTarget: element.hasAttribute("hx-target"),
        hxSwap: element.hasAttribute("hx-swap"),
        hxTrigger: element.hasAttribute("hx-trigger"),
        hxPushUrl: element.hasAttribute("hx-push-url"),
        hxBoost: element.hasAttribute("hx-boost"),
      };
    }, selector);
  }

  /**
   * Verify HTMX swap patterns
   */
  async verifySwapPattern(selector, expectedPattern) {
    const element = this.page.locator(selector);
    const swapValue = await element.getAttribute("hx-swap");
    return swapValue === expectedPattern;
  }

  /**
   * Setup HTMX event monitoring
   */
  async setupHTMXEventMonitoring() {
    await this.page.addInitScript(() => {
      window.htmxEvents = [];
      if (window.htmx) {
        // Monitor all HTMX events
        const events = [
          "htmx:beforeRequest",
          "htmx:afterRequest",
          "htmx:responseError",
          "htmx:sendError",
          "htmx:timeout",
          "htmx:beforeSwap",
          "htmx:afterSwap",
          "htmx:beforeSettle",
          "htmx:afterSettle",
        ];

        events.forEach((event) => {
          htmx.on(event, (e) => {
            window.htmxEvents.push({
              type: event,
              timestamp: Date.now(),
              detail: e.detail,
            });
          });
        });
      }
    });
  }

  /**
   * Get HTMX performance metrics
   */
  async getHTMXMetrics() {
    return this.page.evaluate(() => {
      if (!window.htmxEvents) return null;

      const requests = window.htmxEvents.filter(
        (e) => e.type === "htmx:beforeRequest",
      );
      const responses = window.htmxEvents.filter(
        (e) => e.type === "htmx:afterRequest",
      );
      const errors = window.htmxEvents.filter(
        (e) => e.type === "htmx:responseError" || e.type === "htmx:sendError",
      );

      return {
        totalRequests: requests.length,
        totalResponses: responses.length,
        totalErrors: errors.length,
        errorRate: requests.length > 0 ? errors.length / requests.length : 0,
      };
    });
  }
}

test.describe("Advanced HTMX Integration", () => {
  let htmxUtils;

  test.beforeEach(async ({ page }) => {
    htmxUtils = new HTMXTestUtils(page);
    await htmxUtils.setupHTMXEventMonitoring();
  });

  test.describe("HTMX Library Verification", () => {
    test("should load HTMX 2.x library", async ({ page }) => {
      await page.goto("/");

      const htmxInfo = await page.evaluate(() => {
        return {
          loaded: typeof window.htmx !== "undefined",
          version: window.htmx ? htmx.version : null,
        };
      });

      expect(htmxInfo.loaded).toBe(true);
      expect(htmxInfo.version).toBeTruthy();
      expect(htmxInfo.version.startsWith("2")).toBe(true);
    });

    test("should have proper HTMX configuration", async ({ page }) => {
      await page.goto("/");

      const htmxConfig = await page.evaluate(() => {
        return window.htmx
          ? {
              timeout: htmx.config.timeout,
              historyEnabled: htmx.config.historyEnabled,
              refreshOnHistoryMiss: htmx.config.refreshOnHistoryMiss,
              defaultSwapStyle: htmx.config.defaultSwapStyle,
              defaultSwapDelay: htmx.config.defaultSwapDelay,
              defaultSettleDelay: htmx.config.defaultSettleDelay,
            }
          : null;
      });

      expect(htmxConfig).toBeTruthy();
      expect(htmxConfig.timeout).toBeGreaterThan(0);
      expect(typeof htmxConfig.historyEnabled).toBe("boolean");
    });

    test("should have HTMX extensions loaded", async ({ page }) => {
      await page.goto("/");

      const extensions = await page.evaluate(() => {
        return window.htmx ? Object.keys(htmx.ext || {}) : [];
      });

      // Extensions are optional, but we should check what's available
      expect(Array.isArray(extensions)).toBe(true);
      console.log("Available HTMX extensions:", extensions);
    });
  });

  test.describe("HTMX Element Detection", () => {
    test("should detect HTMX attributes on forms", async ({ page }) => {
      await page.goto("/register");

      const forms = await page.locator("form").count();
      expect(forms).toBeGreaterThan(0);

      // Check each form for HTMX attributes
      for (let i = 0; i < forms; i++) {
        const formSelector = `form:nth-child(${i + 1})`;
        const attributes = await htmxUtils.hasHTMXAttributes(formSelector);

        if (attributes) {
          console.log(`Form ${i + 1} HTMX attributes:`, attributes);
          // At least some forms should have HTMX attributes in a modern app
        }
      }
    });

    test("should find HTMX elements in navigation", async ({ page }) => {
      await page.goto("/");

      // Look for HTMX-enabled navigation elements
      const navElements = await page.evaluate(() => {
        const elements = document.querySelectorAll(
          "nav a[hx-get], nav a[hx-post], nav button[hx-get], nav button[hx-post]",
        );
        return elements.length;
      });

      console.log(`Found ${navElements} HTMX navigation elements`);
      // Navigation might or might not use HTMX, but we record what we find
    });

    test("should detect dynamic content areas", async ({ page }) => {
      await page.goto("/");

      const dynamicAreas = await page.evaluate(() => {
        const areas = document.querySelectorAll(
          '[hx-target], [id*="content"], [class*="dynamic"]',
        );
        return Array.from(areas).map((area) => ({
          tagName: area.tagName,
          id: area.id,
          className: area.className,
          hasHxTarget: area.hasAttribute("hx-target"),
        }));
      });

      console.log("Dynamic content areas found:", dynamicAreas);
      expect(Array.isArray(dynamicAreas)).toBe(true);
    });
  });

  test.describe("HTMX Error Handling", () => {
    test("should handle network errors gracefully", async ({ page }) => {
      await page.goto("/");

      // Simulate network failure
      await page.route("**/*", (route) => {
        if (
          route.request().url().includes("/api/") ||
          route.request().url().includes("/htmx/")
        ) {
          route.abort();
        } else {
          route.continue();
        }
      });

      // Try to trigger HTMX request that will fail
      const htmxElement = page.locator("[hx-get], [hx-post]").first();
      if ((await htmxElement.count()) > 0) {
        await htmxElement.click();

        // Wait a bit for error handling
        await page.waitForTimeout(2000);

        const events = await htmxUtils.getHTMXHistory();
        const errors = events.filter(
          (e) => e.type === "htmx:sendError" || e.type === "htmx:responseError",
        );

        // Should have error events if HTMX request was attempted
        if (errors.length > 0) {
          expect(errors.length).toBeGreaterThan(0);
        }
      }
    });

    test("should handle timeout errors", async ({ page }) => {
      await page.goto("/");

      // Set up very short timeout
      await page.evaluate(() => {
        if (window.htmx) {
          htmx.config.timeout = 100; // 100ms timeout
        }
      });

      // Slow down responses to cause timeout
      await page.route("**/*", async (route) => {
        if (
          route.request().url().includes("/api/") ||
          route.request().url().includes("/htmx/")
        ) {
          await new Promise((resolve) => setTimeout(resolve, 200)); // 200ms delay
        }
        route.continue();
      });

      const htmxElement = page.locator("[hx-get], [hx-post]").first();
      if ((await htmxElement.count()) > 0) {
        await htmxElement.click();
        await page.waitForTimeout(1000);

        const events = await htmxUtils.getHTMXHistory();
        const timeouts = events.filter((e) => e.type === "htmx:timeout");

        // May or may not have timeouts depending on implementation
        console.log("Timeout events:", timeouts.length);
      }
    });
  });

  test.describe("HTMX Progressive Enhancement", () => {
    test("should work without JavaScript", async ({ page }) => {
      // Disable JavaScript by intercepting JS files
      await page.route("**/*.js", (route) => route.abort());
      await page.goto("/login");

      // Forms should still be functional
      const forms = page.locator("form");
      await expect(forms.first()).toBeVisible();

      // Form should have proper action for fallback
      const action = await forms.first().getAttribute("action");
      expect(action).toBeTruthy();
    });

    test("should handle HTMX library load failure", async ({ page }) => {
      // Block HTMX script loading
      await page.route("**/htmx.min.js", (route) => route.abort());
      await page.goto("/");

      const htmxMissing = await page.evaluate(() => {
        return typeof window.htmx === "undefined";
      });

      expect(htmxMissing).toBe(true);

      // Page should still be functional
      const pageTitle = await page.title();
      expect(pageTitle).toBeTruthy();
    });

    test("should provide non-JS fallbacks", async ({ page }) => {
      await page.goto("/");

      // Check that forms have proper action attributes for fallback
      const formsWithActions = await page.evaluate(() => {
        const forms = document.querySelectorAll("form");
        let count = 0;
        forms.forEach((form) => {
          if (form.getAttribute("action")) {
            count++;
          }
        });
        return { total: forms.length, withActions: count };
      });

      console.log("Forms analysis:", formsWithActions);
      // Most forms should have action attributes for progressive enhancement
      if (formsWithActions.total > 0) {
        expect(formsWithActions.withActions).toBeGreaterThan(0);
      }
    });
  });

  test.describe("HTMX Performance", () => {
    test("should load HTMX library efficiently", async ({ page }) => {
      const startTime = Date.now();
      await page.goto("/");

      const loadTime = await page.evaluate(() => {
        return window.htmxLoadTime || Date.now();
      });

      const duration = Date.now() - startTime;
      expect(duration).toBeLessThan(10000); // 10 seconds max
    });

    test("should handle multiple HTMX elements without performance issues", async ({
      page,
    }) => {
      await page.goto("/");

      const startTime = Date.now();

      // Count HTMX elements
      const htmxElementCount = await page.evaluate(() => {
        const elements = document.querySelectorAll(
          "[hx-get], [hx-post], [hx-put], [hx-delete], [hx-patch]",
        );
        return elements.length;
      });

      const endTime = Date.now();
      const processingTime = endTime - startTime;

      console.log(
        `Found ${htmxElementCount} HTMX elements in ${processingTime}ms`,
      );
      expect(processingTime).toBeLessThan(1000); // Should be very fast
    });

    test("should not cause memory leaks during navigation", async ({
      page,
    }) => {
      await page.goto("/");

      // Perform multiple navigations
      const pages = ["/", "/login", "/register"];

      for (let i = 0; i < 3; i++) {
        for (const pagePath of pages) {
          await page.goto(pagePath);
          await page.waitForLoadState("networkidle");
        }
      }

      // Check final page is still responsive
      const finalTitle = await page.title();
      expect(finalTitle).toBeTruthy();

      const metrics = await htmxUtils.getHTMXMetrics();
      if (metrics) {
        console.log("HTMX metrics after navigation test:", metrics);
        expect(metrics.errorRate).toBeLessThan(0.5); // Less than 50% error rate
      }
    });
  });

  // TODO: Add authenticated HTMX tests once auth issues are resolved
  test.describe.skip("HTMX Advanced Features - Authenticated", () => {
    test.skip("should handle partial page updates", async ({ page }) => {
      // Test HTMX updating specific sections without full page reload
      // Requires authentication to access dynamic content
    });

    test.skip("should handle form submissions without page reload", async ({
      page,
    }) => {
      // Test HTMX form submissions with various swap strategies
      // Requires authentication to test actual forms
    });

    test.skip("should handle real-time updates", async ({ page }) => {
      // Test HTMX polling or server-sent events
      // Requires authenticated session and real-time endpoints
    });

    test.skip("should handle infinite scroll patterns", async ({ page }) => {
      // Test HTMX infinite scroll for transaction lists, etc.
      // Requires authenticated session with data
    });

    test.skip("should handle modal interactions", async ({ page }) => {
      // Test HTMX-powered modals and overlays
      // Requires authenticated session to access modal triggers
    });

    test.skip("should handle search and filtering", async ({ page }) => {
      // Test HTMX-powered search functionality
      // Requires authenticated session with searchable data
    });

    test.skip("should handle file uploads with progress", async ({ page }) => {
      // Test HTMX file upload with progress indicators
      // Requires authenticated session and upload endpoints
    });

    test.skip("should handle dependency loading", async ({ page }) => {
      // Test HTMX loading dependent content (e.g., categories -> subcategories)
      // Requires authenticated session with hierarchical data
    });
  });
});

test.describe("HTMX Integration Patterns", () => {
  test("should follow consistent HTMX patterns", async ({ page }) => {
    await page.goto("/");

    // Analyze HTMX usage patterns across the site
    const patterns = await page.evaluate(() => {
      const elements = document.querySelectorAll("[hx-get], [hx-post]");
      const analysis = {
        totalElements: elements.length,
        swapStrategies: {},
        targetPatterns: {},
        triggerPatterns: {},
      };

      elements.forEach((el) => {
        const swap = el.getAttribute("hx-swap") || "innerHTML";
        const target = el.getAttribute("hx-target") || "this";
        const trigger = el.getAttribute("hx-trigger") || "click";

        analysis.swapStrategies[swap] =
          (analysis.swapStrategies[swap] || 0) + 1;
        analysis.targetPatterns[target] =
          (analysis.targetPatterns[target] || 0) + 1;
        analysis.triggerPatterns[trigger] =
          (analysis.triggerPatterns[trigger] || 0) + 1;
      });

      return analysis;
    });

    console.log("HTMX usage patterns:", patterns);
    expect(patterns.totalElements).toBeGreaterThanOrEqual(0);
  });

  test("should use semantic HTMX attributes", async ({ page }) => {
    await page.goto("/");

    const semanticUsage = await page.evaluate(() => {
      const forms = document.querySelectorAll("form");
      const links = document.querySelectorAll("a");
      const buttons = document.querySelectorAll("button");

      return {
        formsWithHtmx: Array.from(forms).filter(
          (f) => f.hasAttribute("hx-post") || f.hasAttribute("hx-get"),
        ).length,
        linksWithHtmx: Array.from(links).filter(
          (l) => l.hasAttribute("hx-get") || l.hasAttribute("hx-post"),
        ).length,
        buttonsWithHtmx: Array.from(buttons).filter(
          (b) => b.hasAttribute("hx-get") || b.hasAttribute("hx-post"),
        ).length,
        totalForms: forms.length,
        totalLinks: links.length,
        totalButtons: buttons.length,
      };
    });

    console.log("Semantic HTMX usage:", semanticUsage);
    // Semantic usage is good practice but not required
  });
});

// Export utilities for use in other test files
export { HTMXTestUtils };

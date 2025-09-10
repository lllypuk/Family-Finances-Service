/**
 * Performance Testing Utilities
 * Provides comprehensive performance measurement tools for E2E tests
 */

export class PerformanceUtils {
  constructor(page) {
    this.page = page;
    this.measurements = [];
  }

  /**
   * Start performance measurement
   */
  async startMeasurement(label = "default") {
    const measurement = {
      label,
      startTime: Date.now(),
      navigationStart: null,
      domContentLoaded: null,
      loadComplete: null,
      firstPaint: null,
      firstContentfulPaint: null,
      largestContentfulPaint: null,
      cumulativeLayoutShift: null,
      firstInputDelay: null,
      networkRequests: [],
      memoryUsage: null,
      jsHeapSize: null,
    };

    // Listen for network requests
    this.page.on("request", (request) => {
      measurement.networkRequests.push({
        url: request.url(),
        method: request.method(),
        timestamp: Date.now(),
      });
    });

    this.measurements.push(measurement);
    return measurement;
  }

  /**
   * Complete performance measurement
   */
  async completeMeasurement(measurement) {
    measurement.endTime = Date.now();
    measurement.duration = measurement.endTime - measurement.startTime;

    // Get performance timing data
    const performanceData = await this.page.evaluate(() => {
      const navigation = performance.getEntriesByType("navigation")[0];
      const paint = performance.getEntriesByType("paint");

      return {
        navigationStart: navigation?.startTime || 0,
        domContentLoaded:
          navigation?.domContentLoadedEventEnd -
            navigation?.domContentLoadedEventStart || 0,
        loadComplete:
          navigation?.loadEventEnd - navigation?.loadEventStart || 0,
        firstPaint:
          paint.find((p) => p.name === "first-paint")?.startTime || null,
        firstContentfulPaint:
          paint.find((p) => p.name === "first-contentful-paint")?.startTime ||
          null,
        transferSize: navigation?.transferSize || 0,
        encodedBodySize: navigation?.encodedBodySize || 0,
        decodedBodySize: navigation?.decodedBodySize || 0,
      };
    });

    // Get Core Web Vitals
    const webVitals = await this.getCoreWebVitals();

    // Get memory usage
    const memoryInfo = await this.getMemoryUsage();

    // Merge all performance data
    Object.assign(measurement, performanceData, webVitals, memoryInfo);

    return measurement;
  }

  /**
   * Get Core Web Vitals (LCP, FID, CLS)
   */
  async getCoreWebVitals() {
    return await this.page.evaluate(() => {
      return new Promise((resolve) => {
        const vitals = {
          largestContentfulPaint: null,
          firstInputDelay: null,
          cumulativeLayoutShift: null,
        };

        // LCP - Largest Contentful Paint
        if ("PerformanceObserver" in window) {
          try {
            const lcpObserver = new PerformanceObserver((list) => {
              const entries = list.getEntries();
              if (entries.length > 0) {
                vitals.largestContentfulPaint =
                  entries[entries.length - 1].startTime;
              }
            });
            lcpObserver.observe({ entryTypes: ["largest-contentful-paint"] });

            // CLS - Cumulative Layout Shift
            const clsObserver = new PerformanceObserver((list) => {
              let clsValue = 0;
              for (const entry of list.getEntries()) {
                if (!entry.hadRecentInput) {
                  clsValue += entry.value;
                }
              }
              vitals.cumulativeLayoutShift = clsValue;
            });
            clsObserver.observe({ entryTypes: ["layout-shift"] });

            // FID - First Input Delay
            const fidObserver = new PerformanceObserver((list) => {
              for (const entry of list.getEntries()) {
                vitals.firstInputDelay =
                  entry.processingStart - entry.startTime;
                break; // We only care about the first input
              }
            });
            fidObserver.observe({ entryTypes: ["first-input"] });

            // Wait a bit for observers to collect data
            setTimeout(() => {
              resolve(vitals);
            }, 1000);
          } catch (error) {
            console.warn("Performance Observer not available:", error);
            resolve(vitals);
          }
        } else {
          resolve(vitals);
        }
      });
    });
  }

  /**
   * Get memory usage information
   */
  async getMemoryUsage() {
    return await this.page.evaluate(() => {
      if (performance.memory) {
        return {
          memoryUsage: {
            usedJSHeapSize: performance.memory.usedJSHeapSize,
            totalJSHeapSize: performance.memory.totalJSHeapSize,
            jsHeapSizeLimit: performance.memory.jsHeapSizeLimit,
          },
          jsHeapSize: performance.memory.usedJSHeapSize,
        };
      }
      return { memoryUsage: null, jsHeapSize: null };
    });
  }

  /**
   * Measure page load performance
   */
  async measurePageLoad(url, options = {}) {
    const { timeout = 30000, waitForSelector = null } = options;

    const measurement = await this.startMeasurement(`page-load-${url}`);

    try {
      await this.page.goto(url, { waitUntil: "networkidle", timeout });

      if (waitForSelector) {
        await this.page.waitForSelector(waitForSelector, { timeout: 10000 });
      }

      await this.completeMeasurement(measurement);
      return measurement;
    } catch (error) {
      measurement.error = error.message;
      measurement.endTime = Date.now();
      measurement.duration = measurement.endTime - measurement.startTime;
      return measurement;
    }
  }

  /**
   * Measure form submission performance
   */
  async measureFormSubmission(formSelector, formData = {}) {
    const measurement = await this.startMeasurement(
      `form-submit-${formSelector}`,
    );

    try {
      const form = this.page.locator(formSelector);

      // Fill form fields if data provided
      for (const [field, value] of Object.entries(formData)) {
        const fieldSelector = `${formSelector} input[name="${field}"], ${formSelector} select[name="${field}"], ${formSelector} textarea[name="${field}"]`;
        const fieldElement = this.page.locator(fieldSelector);
        if (await fieldElement.isVisible()) {
          await fieldElement.fill(value);
        }
      }

      // Submit form and wait for response
      const submitButton = form.locator(
        'button[type="submit"], input[type="submit"]',
      );
      await submitButton.click();

      // Wait for form processing (look for loading indicators or result elements)
      await this.page.waitForLoadState("networkidle");

      await this.completeMeasurement(measurement);
      return measurement;
    } catch (error) {
      measurement.error = error.message;
      measurement.endTime = Date.now();
      measurement.duration = measurement.endTime - measurement.startTime;
      return measurement;
    }
  }

  /**
   * Measure HTMX request performance
   */
  async measureHTMXRequest(triggerSelector, options = {}) {
    const { waitForUpdate = true, timeout = 10000 } = options;

    const measurement = await this.startMeasurement(`htmx-${triggerSelector}`);

    try {
      // Set up HTMX event monitoring
      await this.page.addInitScript(() => {
        if (window.htmx) {
          window.htmxRequestStart = null;
          window.htmxRequestEnd = null;

          htmx.on("htmx:beforeRequest", () => {
            window.htmxRequestStart = Date.now();
          });

          htmx.on("htmx:afterRequest", () => {
            window.htmxRequestEnd = Date.now();
          });
        }
      });

      // Trigger HTMX request
      await this.page.click(triggerSelector);

      if (waitForUpdate) {
        // Wait for HTMX to complete
        await this.page.waitForFunction(
          () => {
            return window.htmxRequestEnd !== null;
          },
          { timeout },
        );
      }

      // Get HTMX timing
      const htmxTiming = await this.page.evaluate(() => {
        return {
          htmxStart: window.htmxRequestStart,
          htmxEnd: window.htmxRequestEnd,
          htmxDuration: window.htmxRequestEnd - window.htmxRequestStart,
        };
      });

      Object.assign(measurement, htmxTiming);
      await this.completeMeasurement(measurement);
      return measurement;
    } catch (error) {
      measurement.error = error.message;
      measurement.endTime = Date.now();
      measurement.duration = measurement.endTime - measurement.startTime;
      return measurement;
    }
  }

  /**
   * Measure search/filter performance
   */
  async measureSearchPerformance(searchInput, query, options = {}) {
    const { waitForResults = true, timeout = 10000 } = options;

    const measurement = await this.startMeasurement(`search-${query}`);

    try {
      const searchField = this.page.locator(searchInput);

      // Clear and type search query
      await searchField.clear();
      await searchField.type(query);

      if (waitForResults) {
        // Wait for search results or network idle
        await this.page.waitForLoadState("networkidle");
      }

      await this.completeMeasurement(measurement);
      return measurement;
    } catch (error) {
      measurement.error = error.message;
      measurement.endTime = Date.now();
      measurement.duration = measurement.endTime - measurement.startTime;
      return measurement;
    }
  }

  /**
   * Stress test with concurrent operations
   */
  async stressTest(operations = [], concurrency = 5) {
    const results = [];
    const chunks = this.chunkArray(operations, concurrency);

    for (const chunk of chunks) {
      const chunkPromises = chunk.map(async (operation) => {
        try {
          const result = await operation();
          return { success: true, result };
        } catch (error) {
          return { success: false, error: error.message };
        }
      });

      const chunkResults = await Promise.all(chunkPromises);
      results.push(...chunkResults);
    }

    return {
      total: operations.length,
      successful: results.filter((r) => r.success).length,
      failed: results.filter((r) => !r.success).length,
      results,
    };
  }

  /**
   * Memory leak detection
   */
  async detectMemoryLeaks(action, iterations = 10) {
    const measurements = [];

    for (let i = 0; i < iterations; i++) {
      const beforeMemory = await this.getMemoryUsage();

      await action();

      // Force garbage collection if available
      await this.page.evaluate(() => {
        if (window.gc) {
          window.gc();
        }
      });

      const afterMemory = await this.getMemoryUsage();

      measurements.push({
        iteration: i + 1,
        beforeHeapSize: beforeMemory.jsHeapSize,
        afterHeapSize: afterMemory.jsHeapSize,
        heapIncrease: afterMemory.jsHeapSize - beforeMemory.jsHeapSize,
      });
    }

    // Calculate average heap increase
    const averageIncrease =
      measurements.reduce((sum, m) => sum + m.heapIncrease, 0) /
      measurements.length;

    return {
      measurements,
      averageHeapIncrease: averageIncrease,
      potentialLeak: averageIncrease > 1048576, // 1MB threshold
      recommendation:
        averageIncrease > 1048576
          ? "Potential memory leak detected - average increase exceeds 1MB"
          : "Memory usage appears stable",
    };
  }

  /**
   * Network performance analysis
   */
  async analyzeNetworkPerformance() {
    const requests = [];

    this.page.on("request", (request) => {
      requests.push({
        url: request.url(),
        method: request.method(),
        timestamp: Date.now(),
        resourceType: request.resourceType(),
      });
    });

    this.page.on("response", (response) => {
      const request = requests.find((r) => r.url === response.url());
      if (request) {
        request.status = response.status();
        request.responseTime = Date.now() - request.timestamp;
        request.size = response.headers()["content-length"] || 0;
      }
    });

    // Return analysis function
    return () => {
      const completed = requests.filter((r) => r.status);
      const failed = completed.filter((r) => r.status >= 400);
      const avgResponseTime =
        completed.reduce((sum, r) => sum + r.responseTime, 0) /
        completed.length;
      const totalSize = completed.reduce(
        (sum, r) => sum + parseInt(r.size || 0),
        0,
      );

      return {
        totalRequests: requests.length,
        completedRequests: completed.length,
        failedRequests: failed.length,
        averageResponseTime: avgResponseTime,
        totalTransferSize: totalSize,
        slowRequests: completed.filter((r) => r.responseTime > 2000),
        recommendations: [
          ...(avgResponseTime > 1000
            ? ["Consider optimizing server response times"]
            : []),
          ...(failed.length > 0
            ? ["Fix failed requests to improve reliability"]
            : []),
          ...(totalSize > 5242880
            ? [
                "Consider optimizing asset sizes (current: " +
                  Math.round(totalSize / 1024 / 1024) +
                  "MB)",
              ]
            : []),
        ],
      };
    };
  }

  /**
   * Generate performance report
   */
  generateReport() {
    const report = {
      summary: {
        totalMeasurements: this.measurements.length,
        avgDuration:
          this.measurements.reduce((sum, m) => sum + (m.duration || 0), 0) /
          this.measurements.length,
        failedMeasurements: this.measurements.filter((m) => m.error).length,
      },
      measurements: this.measurements,
      recommendations: this.generateRecommendations(),
    };

    return report;
  }

  /**
   * Generate performance recommendations
   */
  generateRecommendations() {
    const recommendations = [];

    // Analyze page load times
    const pageLoads = this.measurements.filter((m) =>
      m.label.includes("page-load"),
    );
    if (pageLoads.length > 0) {
      const avgLoadTime =
        pageLoads.reduce((sum, m) => sum + (m.duration || 0), 0) /
        pageLoads.length;
      if (avgLoadTime > 3000) {
        recommendations.push(
          "Page load times exceed 3 seconds - consider optimization",
        );
      }
    }

    // Analyze form submissions
    const formSubmissions = this.measurements.filter((m) =>
      m.label.includes("form-submit"),
    );
    if (formSubmissions.length > 0) {
      const avgSubmitTime =
        formSubmissions.reduce((sum, m) => sum + (m.duration || 0), 0) /
        formSubmissions.length;
      if (avgSubmitTime > 2000) {
        recommendations.push(
          "Form submissions take longer than 2 seconds - optimize processing",
        );
      }
    }

    // Analyze Core Web Vitals
    const lcpMeasurements = this.measurements.filter(
      (m) => m.largestContentfulPaint,
    );
    if (lcpMeasurements.length > 0) {
      const avgLCP =
        lcpMeasurements.reduce((sum, m) => sum + m.largestContentfulPaint, 0) /
        lcpMeasurements.length;
      if (avgLCP > 2500) {
        recommendations.push(
          "Largest Contentful Paint exceeds 2.5s - optimize critical resources",
        );
      }
    }

    // Analyze memory usage
    const memoryMeasurements = this.measurements.filter((m) => m.jsHeapSize);
    if (memoryMeasurements.length > 0) {
      const maxHeapSize = Math.max(
        ...memoryMeasurements.map((m) => m.jsHeapSize),
      );
      if (maxHeapSize > 52428800) {
        // 50MB
        recommendations.push(
          "JavaScript heap size exceeds 50MB - check for memory leaks",
        );
      }
    }

    if (recommendations.length === 0) {
      recommendations.push("Performance metrics are within acceptable ranges");
    }

    return recommendations;
  }

  /**
   * Utility function to chunk arrays
   */
  chunkArray(array, size) {
    const chunks = [];
    for (let i = 0; i < array.length; i += size) {
      chunks.push(array.slice(i, i + size));
    }
    return chunks;
  }

  /**
   * Clear all measurements
   */
  clearMeasurements() {
    this.measurements = [];
  }

  /**
   * Get measurements by label
   */
  getMeasurementsByLabel(label) {
    return this.measurements.filter((m) => m.label.includes(label));
  }

  /**
   * Get performance thresholds for different operations
   */
  static getPerformanceThresholds() {
    return {
      pageLoad: {
        good: 2000,
        needsImprovement: 3000,
        poor: 5000,
      },
      formSubmit: {
        good: 1000,
        needsImprovement: 2000,
        poor: 3000,
      },
      htmxRequest: {
        good: 500,
        needsImprovement: 1000,
        poor: 2000,
      },
      search: {
        good: 300,
        needsImprovement: 1000,
        poor: 2000,
      },
      coreWebVitals: {
        lcp: { good: 2500, needsImprovement: 4000 },
        fid: { good: 100, needsImprovement: 300 },
        cls: { good: 0.1, needsImprovement: 0.25 },
      },
    };
  }
}

/**
 * Page Object Model for Reports Page
 * Handles financial reporting, filtering, and export functionality
 */

export class ReportsPage {
  constructor(page) {
    this.page = page;

    // Page selectors
    this.selectors = {
      // Main page elements
      pageTitle: 'h1, h2',
      reportsForm: 'form[action*="reports"], form[hx-post*="reports"]',
      
      // Report generation form
      reportType: 'select[name="report_type"], input[name="report_type"]',
      startDate: 'input[name="start_date"], input[name="date_from"]',
      endDate: 'input[name="end_date"], input[name="date_to"]',
      categoryFilter: 'select[name="category"], select[name="categories[]"]',
      userFilter: 'select[name="user"], select[name="users[]"]',
      generateButton: 'button[type="submit"]:has-text("Генерировать"), button:has-text("Generate")',

      // Export options
      exportPdfButton: 'button:has-text("PDF"), a[href*="pdf"]',
      exportCsvButton: 'button:has-text("CSV"), a[href*="csv"]',
      exportExcelButton: 'button:has-text("Excel"), a[href*="excel"]',

      // Report results
      reportResults: '.report-results, .report-content, #report-content',
      reportTable: 'table.report-table, .report-table, table',
      reportChart: '.chart, .report-chart, canvas, svg',
      
      // Summary sections
      totalIncome: '[data-testid="total-income"], .total-income, .income-total',
      totalExpenses: '[data-testid="total-expenses"], .total-expenses, .expenses-total',
      netIncome: '[data-testid="net-income"], .net-income, .balance',

      // Category breakdown
      categoryBreakdown: '.category-breakdown, .categories-summary',
      categoryItem: '.category-item, .category-row',
      categoryName: '.category-name',
      categoryAmount: '.category-amount',
      categoryPercentage: '.category-percentage',

      // Filter and search
      quickFilters: '.quick-filters, .filter-buttons',
      thisMonth: 'button:has-text("This Month"), button:has-text("Текущий месяц")',
      lastMonth: 'button:has-text("Last Month"), button:has-text("Прошлый месяц")',
      thisYear: 'button:has-text("This Year"), button:has-text("Текущий год")',
      customPeriod: 'button:has-text("Custom"), button:has-text("Выбрать период")',

      // HTMX elements
      htmxReportContent: '[hx-get*="reports"], [hx-post*="reports"]',
      loadingIndicator: '[aria-busy="true"], .loading, .spinner',

      // Error handling
      errorMessages: '.alert-error, .error, .alert-danger',
      emptyState: '.empty-state, .no-data, .no-results',

      // Navigation
      backToDashboard: 'a[href="/"], a[href="/dashboard"]',
      printButton: 'button:has-text("Print"), button:has-text("Печать")',
    };

    // Report types
    this.reportTypes = [
      'income_expense',
      'category_breakdown', 
      'monthly_summary',
      'yearly_overview',
      'transaction_details'
    ];

    // Quick filter periods
    this.quickPeriods = [
      'this_month',
      'last_month', 
      'this_year',
      'last_year',
      'custom'
    ];
  }

  /**
   * Navigate to reports page
   */
  async navigate() {
    await this.page.goto('/reports');
    await this.page.waitForSelector(this.selectors.pageTitle);
  }

  /**
   * Check if reports page is loaded
   */
  async isReportsPageLoaded() {
    try {
      await this.page.waitForSelector(this.selectors.pageTitle, { timeout: 3000 });
      const url = this.page.url();
      return url.includes('/reports');
    } catch {
      return false;
    }
  }

  /**
   * Generate report with specified parameters
   */
  async generateReport(options = {}) {
    const {
      reportType = 'income_expense',
      startDate = '2024-01-01',
      endDate = '2024-12-31',
      category = null,
      user = null
    } = options;

    // Select report type if available
    const reportTypeField = this.page.locator(this.selectors.reportType);
    if (await reportTypeField.isVisible()) {
      await reportTypeField.selectOption(reportType);
    }

    // Set date range
    const startDateField = this.page.locator(this.selectors.startDate);
    if (await startDateField.isVisible()) {
      await startDateField.fill(startDate);
    }

    const endDateField = this.page.locator(this.selectors.endDate);
    if (await endDateField.isVisible()) {
      await endDateField.fill(endDate);
    }

    // Set filters
    if (category) {
      const categoryField = this.page.locator(this.selectors.categoryFilter);
      if (await categoryField.isVisible()) {
        await categoryField.selectOption(category);
      }
    }

    if (user) {
      const userField = this.page.locator(this.selectors.userFilter);
      if (await userField.isVisible()) {
        await userField.selectOption(user);
      }
    }

    // Generate report
    await this.page.click(this.selectors.generateButton);
    await this.waitForReportGeneration();
  }

  /**
   * Wait for report generation to complete
   */
  async waitForReportGeneration() {
    // Wait for loading to start
    try {
      await this.page.waitForSelector(this.selectors.loadingIndicator, { timeout: 1000 });
    } catch {
      // Loading indicator might not appear for fast reports
    }

    // Wait for loading to finish
    try {
      await this.page.waitForSelector(this.selectors.loadingIndicator, { 
        state: 'hidden', 
        timeout: 10000 
      });
    } catch {
      // Loading indicator might not have appeared
    }

    // Wait for report content
    await this.page.waitForSelector(this.selectors.reportResults, { timeout: 10000 });
  }

  /**
   * Use quick filter for common periods
   */
  async useQuickFilter(period) {
    const filterMap = {
      'this_month': this.selectors.thisMonth,
      'last_month': this.selectors.lastMonth,
      'this_year': this.selectors.thisYear,
      'custom': this.selectors.customPeriod
    };

    if (filterMap[period]) {
      await this.page.click(filterMap[period]);
      await this.waitForReportGeneration();
      return true;
    }
    return false;
  }

  /**
   * Check if report results are displayed
   */
  async hasReportResults() {
    const hasResults = await this.page.locator(this.selectors.reportResults).isVisible();
    const hasTable = await this.page.locator(this.selectors.reportTable).isVisible();
    const hasChart = await this.page.locator(this.selectors.reportChart).isVisible();
    
    return {
      hasResults,
      hasTable,
      hasChart,
      hasAnyContent: hasResults || hasTable || hasChart
    };
  }

  /**
   * Get report summary data
   */
  async getReportSummary() {
    const summary = {};

    // Total income
    const incomeElement = this.page.locator(this.selectors.totalIncome);
    if (await incomeElement.isVisible()) {
      const incomeText = await incomeElement.textContent();
      summary.totalIncome = this.extractAmount(incomeText);
    }

    // Total expenses
    const expensesElement = this.page.locator(this.selectors.totalExpenses);
    if (await expensesElement.isVisible()) {
      const expensesText = await expensesElement.textContent();
      summary.totalExpenses = this.extractAmount(expensesText);
    }

    // Net income
    const netElement = this.page.locator(this.selectors.netIncome);
    if (await netElement.isVisible()) {
      const netText = await netElement.textContent();
      summary.netIncome = this.extractAmount(netText);
    }

    return summary;
  }

  /**
   * Extract numeric amount from text
   */
  extractAmount(text) {
    if (!text) return null;
    const match = text.match(/[\d,.\s]+/);
    return match ? parseFloat(match[0].replace(/[,\s]/g, '')) : null;
  }

  /**
   * Get category breakdown data
   */
  async getCategoryBreakdown() {
    const categories = [];
    const categoryItems = this.page.locator(this.selectors.categoryItem);
    const count = await categoryItems.count();

    for (let i = 0; i < count; i++) {
      const item = categoryItems.nth(i);
      
      const nameElement = item.locator(this.selectors.categoryName);
      const amountElement = item.locator(this.selectors.categoryAmount);
      const percentageElement = item.locator(this.selectors.categoryPercentage);

      const category = {
        name: await nameElement.textContent().catch(() => ''),
        amount: await amountElement.textContent().catch(() => ''),
        percentage: await percentageElement.textContent().catch(() => '')
      };

      if (category.name) {
        categories.push(category);
      }
    }

    return categories;
  }

  /**
   * Test export functionality
   */
  async testExport(format = 'csv') {
    const exportMap = {
      'csv': this.selectors.exportCsvButton,
      'pdf': this.selectors.exportPdfButton,
      'excel': this.selectors.exportExcelButton
    };

    if (!exportMap[format]) {
      throw new Error(`Unsupported export format: ${format}`);
    }

    const exportButton = this.page.locator(exportMap[format]);
    
    if (await exportButton.isVisible()) {
      // Set up download handler
      const downloadPromise = this.page.waitForEvent('download');
      
      await exportButton.click();
      
      try {
        const download = await downloadPromise;
        return {
          success: true,
          filename: download.suggestedFilename(),
          format: format
        };
      } catch (error) {
        return {
          success: false,
          error: error.message,
          format: format
        };
      }
    }

    return { success: false, error: 'Export button not found', format: format };
  }

  /**
   * Check for empty state
   */
  async hasEmptyState() {
    return await this.page.locator(this.selectors.emptyState).isVisible();
  }

  /**
   * Get error messages
   */
  async getErrorMessages() {
    const errorElements = this.page.locator(this.selectors.errorMessages);
    const count = await errorElements.count();
    const errors = [];

    for (let i = 0; i < count; i++) {
      const errorText = await errorElements.nth(i).textContent();
      errors.push(errorText.trim());
    }

    return errors;
  }

  /**
   * Verify HTMX integration
   */
  async verifyHtmxIntegration() {
    const htmxElements = this.page.locator(this.selectors.htmxReportContent);
    const count = await htmxElements.count();
    
    const integration = {
      hasHtmxElements: count > 0,
      elementCount: count
    };

    if (count > 0) {
      const firstElement = htmxElements.first();
      integration.hasHxGet = await firstElement.getAttribute('hx-get') !== null;
      integration.hasHxPost = await firstElement.getAttribute('hx-post') !== null;
      integration.hasHxTarget = await firstElement.getAttribute('hx-target') !== null;
    }

    return integration;
  }

  /**
   * Test report form validation
   */
  async testFormValidation() {
    // Try to generate report with invalid data
    const startDateField = this.page.locator(this.selectors.startDate);
    const endDateField = this.page.locator(this.selectors.endDate);

    if (await startDateField.isVisible() && await endDateField.isVisible()) {
      // Set invalid date range (end before start)
      await startDateField.fill('2024-12-31');
      await endDateField.fill('2024-01-01');
      
      await this.page.click(this.selectors.generateButton);
      await this.page.waitForTimeout(1000);

      const errors = await this.getErrorMessages();
      return {
        hasValidation: errors.length > 0,
        errors: errors
      };
    }

    return { hasValidation: false, errors: [] };
  }

  /**
   * Test responsive design
   */
  async testResponsiveDesign() {
    // Test mobile
    await this.page.setViewportSize({ width: 375, height: 667 });
    await this.page.waitForTimeout(500);
    
    const mobile = {
      formVisible: await this.page.locator(this.selectors.reportsForm).isVisible(),
      titleVisible: await this.page.locator(this.selectors.pageTitle).isVisible(),
    };

    // Test desktop
    await this.page.setViewportSize({ width: 1280, height: 720 });
    await this.page.waitForTimeout(500);
    
    const desktop = {
      formVisible: await this.page.locator(this.selectors.reportsForm).isVisible(),
      titleVisible: await this.page.locator(this.selectors.pageTitle).isVisible(),
    };

    return { mobile, desktop };
  }

  /**
   * Get page title
   */
  async getPageTitle() {
    return await this.page.title();
  }

  /**
   * Navigate back to dashboard
   */
  async backToDashboard() {
    const backButton = this.page.locator(this.selectors.backToDashboard);
    if (await backButton.isVisible()) {
      await backButton.click();
      await this.page.waitForLoadState('networkidle');
      return true;
    }
    return false;
  }

  /**
   * Test print functionality
   */
  async testPrintFunctionality() {
    const printButton = this.page.locator(this.selectors.printButton);
    if (await printButton.isVisible()) {
      // We can't actually test printing, but we can test the button exists
      return { hasPrintButton: true };
    }
    return { hasPrintButton: false };
  }
}
# Playwright E2E Testing Setup

## Overview

Playwright E2E tests have been configured for the Family Finances Service to test the web interface built with HTMX 2.0.4 and PicoCSS 2.1.1.

## Setup Complete

✅ **Playwright installed and configured**
✅ **Test structure created**
✅ **npm scripts added**
✅ **Makefile commands added**
✅ **Basic test examples created**

## File Structure

```
tests/e2e/
├── auth.spec.js           # Authentication tests (login, register, validation)
├── dashboard.spec.js      # Dashboard tests (requires auth setup)
├── htmx.spec.js          # HTMX integration tests
└── helpers/
    └── auth.js           # Authentication helper functions
```

## Configuration

- **Base URL**: http://localhost:8080
- **Browsers**: Chromium, Mobile Chrome
- **Test Directory**: `./tests/e2e`
- **Reports**: HTML format
- **Screenshots**: On failure only
- **Videos**: On failure only

## Running Tests

### npm Commands
```bash
npm run test:e2e              # Run all Playwright tests
npm run test:e2e:headed       # Run tests in headed mode (visible browser)
npm run test:e2e:debug        # Run tests in debug mode
npm run test:e2e:ui           # Run tests with Playwright UI
npm run test:e2e:report       # Show HTML report
```

### Makefile Commands
```bash
make test-e2e                    # Run Playwright tests
make test-e2e-playwright-ui      # Run with UI
make test-e2e-playwright-debug   # Run in debug mode
```

## Test Prerequisites

**Before running tests:**
```bash
make dev-up        # Start MongoDB and Redis
make run     # Start application on localhost:8080
```

## Current Test Status

### ✅ Implemented Tests
- **Authentication UI**: Login/register form validation
- **HTMX Integration**: Library loading verification
- **Responsive Design**: Mobile viewport testing

### 🚧 Requires Authentication Setup
- **Dashboard Tests**: Currently skipped, need auth helper
- **CRUD Operations**: Transactions, categories, budgets
- **HTMX Dynamic Updates**: Form submissions, real-time updates

## Next Steps

1. **Complete authentication helper** in `tests/e2e/helpers/auth.js`
2. **Add test data fixtures** for predictable test scenarios
3. **Implement authenticated test flows**:
   - Dashboard functionality
   - Transaction management
   - Budget operations
   - Report generation
4. **Add visual regression tests** for UI components
5. **Test HTMX dynamic behaviors** (form submissions, partial updates)

## Authentication Helper Usage

```javascript
import { AuthHelper } from './helpers/auth.js';

test('dashboard access', async ({ page }) => {
  const auth = new AuthHelper(page);
  await auth.registerAndLogin();

  await page.goto('/dashboard');
  // Test authenticated dashboard functionality
});
```

## Best Practices

- Use `data-testid` attributes for reliable element selection
- Test user journeys, not just individual pages
- Focus on critical business flows
- Test both desktop and mobile viewports
- Verify HTMX dynamic updates work correctly
- Test form validation and error states

## CI/CD Integration

Tests are ready for CI/CD integration. Consider adding to GitHub Actions workflow for automated testing on pull requests.

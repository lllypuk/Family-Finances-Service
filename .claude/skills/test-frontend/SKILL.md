---
name: test-frontend
description: Test the web frontend using agent-browser for interactive testing, HTMX verification, and screenshot capture
disable-model-invocation: true
argument-hint: [ url or endpoint ]
allowed-tools: Bash(agent-browser *), Bash(make run-local)
---

# Frontend Testing with Agent-Browser

Test the web interface using **agent-browser** for interactive browser automation and validation.

## Prerequisites

Ensure the local server is running:

```bash
make run-local  # Starts server on localhost:8080
```

## Basic Usage

### 1. Open the browser

```bash
agent-browser open "http://127.0.0.1:8080$ARGUMENTS"
```

If no arguments provided, opens the root page.

### 2. Take a snapshot

Get the accessibility tree with element references:

```bash
agent-browser snapshot
```

This shows all interactive elements with unique IDs (e.g., `@e5`, `@e12`)

### 3. Interact with elements

```bash
# Click on an element
agent-browser click @e5

# Fill a form field
agent-browser fill @e12 "test@example.com"

# Type text
agent-browser type @e8 "password123"
```

### 4. Capture screenshots

```bash
agent-browser screenshot /tmp/page.png
```

### 5. Close browser

```bash
agent-browser close
```

## Common Testing Scenarios

### Login Flow Validation

1. Open login page
2. Snapshot to find form elements
3. Fill username and password fields
4. Click submit button
5. Verify redirect and session

### HTMX Dynamic Updates

1. Load page with HTMX components
2. Take snapshot before interaction
3. Trigger HTMX action (click, form submit)
4. Take snapshot after to verify DOM updates
5. Confirm expected changes occurred

### Form Submission

1. Navigate to form page
2. Identify form fields via snapshot
3. Fill all required fields
4. Submit form
5. Verify success message or redirect

### Session Management

1. Test login/logout flow
2. Verify session persistence
3. Check unauthorized access handling

## Full Example Workflow

```bash
# 1. Start local server (if not running)
make run-local

# 2. In another terminal, test login
agent-browser open "http://127.0.0.1:8080/login"
agent-browser snapshot  # Find username/password field IDs

# 3. Fill login form (replace @e5, @e6 with actual IDs)
agent-browser fill @e5 "admin@example.com"
agent-browser fill @e6 "password123"
agent-browser click @e7  # Submit button

# 4. Verify redirect to dashboard
agent-browser snapshot  # Should show dashboard elements

# 5. Take screenshot for documentation
agent-browser screenshot /tmp/dashboard.png

# 6. Cleanup
agent-browser close
```

## Tips

- Always take a snapshot first to find element IDs
- Use `--noproxy '*'` with curl for health checks
- Element IDs are temporary - get fresh snapshot after page changes
- Screenshots help document UI state for debugging

## Documentation

Full command reference: https://github.com/vercel-labs/agent-browser

---
name: pre-commit
description: Run comprehensive pre-commit checks (format, test, lint) before committing code
disable-model-invocation: true
allowed-tools: Bash(make *)
---

# Pre-Commit Quality Checks

Run the complete pre-commit validation sequence to ensure code quality before committing.

## What This Does

Executes the following checks in order:

1. **Format** (`make fmt`) - Auto-format code with gofmt
2. **Test** (`make test`) - Run full test suite with in-memory SQLite
3. **Lint** (`make lint`) - Run golangci-lint with 50+ quality rules

## Usage

Simply invoke this skill to run all checks:

```bash
make pre-commit
```

## Expected Behavior

- **All checks must pass** with 0 errors
- **Format** will auto-fix code style issues
- **Tests** must achieve current coverage levels (36.2%+)
- **Lint** must report 0 issues - no exceptions

## When to Use

Run this skill:

- **Before every commit** - ensures quality standards
- **After completing any development work** - mandatory validation
- **Before creating a pull request** - CI will run same checks
- **When fixing bugs** - verify no regressions introduced

## Workflow Integration

This skill enforces the mandatory quality workflow:

```
1. Make changes → 2. make fmt → 3. make test → 4. make lint → 5. Commit
```

If any check fails, **do not commit** until all issues are resolved.

## Common Failures

### Formatting Issues

- **Cause**: Code not following gofmt standards
- **Fix**: `make fmt` auto-corrects these

### Test Failures

- **Cause**: Breaking changes, missing test updates
- **Fix**: Update tests to match new behavior, verify logic is correct

### Linter Errors

- **Cause**: Code quality violations (see `.golangci.yml` for rules)
- **Fix**: Address each error individually - no bypassing allowed
- **Common**: unused variables, unchecked errors, complexity violations

## Zero Tolerance Policy

**MANDATORY LINTER COMPLIANCE**: All linter errors must be fixed before committing.

- ✅ 0 issues = ready to commit
- ❌ Any issues = fix required

No exceptions, no compromises - clean code is mandatory for project integrity.

## See Also

- `make fmt` - Format only
- `make test` - Test only
- `make lint` - Lint only
- `make test-coverage` - Test with coverage report

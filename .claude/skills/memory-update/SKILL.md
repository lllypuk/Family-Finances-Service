---
name: memory-update
description: Update project documentation in .memory_bank/ to reflect architectural changes, decisions, and current state
argument-hint: [topic or file]
allowed-tools: Read, Edit, Write, Glob
---

# Update Project Memory Bank

Update documentation in `.memory_bank/` to keep project knowledge current and accurate.

## What is Memory Bank?

The `.memory_bank/` directory contains comprehensive project documentation:

- **Product Brief** (`product_brief.md`) - Business context and goals
- **Tech Stack** (`tech_stack.md`) - Architecture and technology choices
- **Testing Plan** (`guides/testing_strategy.md`) - Testing strategy and coverage
- **Current Tasks** (`tasks/current_task.md`) - Project status and next steps
- **Patterns & Standards** (`patterns/`, `guides/`) - Best practices
- **Workflows** (`workflows/`) - Common development workflows

## When to Update

Update memory bank documentation when:

1. **Architecture changes**: Technology switches, new patterns adopted
2. **Major features completed**: Document new capabilities
3. **Decisions made**: Record rationale for future reference
4. **Testing strategy evolved**: Coverage improvements, new test types
5. **Workflows established**: Codify repeated processes
6. **Tech stack updated**: Dependencies added/removed

## Update Workflow

### 1. Identify What Changed

Determine which aspect of the project changed:
- Architecture/tech stack
- Testing approach
- Development workflows
- Coding standards
- API patterns

### 2. Find Relevant Files

```bash
# List all memory bank files
find .memory_bank -name "*.md" -type f

# Search for specific topic
grep -r "SQLite" .memory_bank/
```

Common files to update:
- `.memory_bank/tech_stack.md` - Technology choices
- `.memory_bank/product_brief.md` - Product direction
- `.memory_bank/guides/testing_strategy.md` - Testing approach
- `.memory_bank/tasks/current_task.md` - Current work
- `.memory_bank/backlog.md` - Future priorities

### 3. Update Content

Use Edit tool to update specific sections:
- Add new information
- Update outdated details
- Remove deprecated content
- Clarify ambiguous sections

### 4. Verify Consistency

Ensure updates are consistent across related files:
- Tech stack matches actual dependencies
- Testing strategy reflects current coverage
- Workflows align with current practices

## Example Updates

### After switching PostgreSQL → SQLite

Update `.memory_bank/tech_stack.md`:
```markdown
## Database
- **SQLite** (modernc.org/sqlite) - Embedded database, pure Go, no CGO
  - Single-file database at `./data/budget.db`
  - No external database server required
  - Perfect for self-hosted single-family deployment
```

### After improving test coverage

Update `.memory_bank/guides/testing_strategy.md`:
```markdown
## Current Coverage: 42.5%

- **Application**: 91.2% (excellent)
- **Domain**: 77.6% (good)
- **Infrastructure**: 69.9% (good)
- **Web**: 35.2% (improving) ⬆ from 28.4%
```

### After establishing new workflow

Create `.memory_bank/workflows/feature_development.md`:
```markdown
# Feature Development Workflow

1. Plan the feature (/plan if needed)
2. Implement with tests
3. Run pre-commit checks (/pre-commit)
4. Create commit
5. Create PR
```

## Arguments

You can pass a topic or specific file to focus the update:

```
/memory-update tech_stack
/memory-update .memory_bank/guides/testing_strategy.md
/memory-update "SQLite migration"
```

If $ARGUMENTS provided, focus on that specific area:
- Search for relevant files
- Update matching documentation
- Ensure consistency

## Best Practices

1. **Be specific**: Update concrete facts, not vague statements
2. **Date decisions**: Note when major changes were made
3. **Include rationale**: Explain why, not just what
4. **Link to code**: Reference files and line numbers when relevant
5. **Keep current**: Remove outdated information
6. **Maintain structure**: Follow existing file organization

## File Structure

```
.memory_bank/
├── README.md                           # Memory bank overview
├── product_brief.md                    # Business context
├── tech_stack.md                       # Technology choices
├── backlog.md                          # Future priorities
├── guides/
│   ├── coding_standards.md            # Code style guide
│   └── testing_strategy.md            # Testing approach
├── patterns/
│   ├── api_standards.md               # API conventions
│   └── error_handling.md              # Error patterns
├── tasks/
│   └── current_task.md                # Active work
├── workflows/
│   ├── bug_fix.md                     # Bug fix process
│   └── new_feature.md                 # Feature workflow
└── specs/
    └── feature_xyz.md                 # Feature specifications
```

## Integration with CLAUDE.md

While CLAUDE.md provides guidance for Claude Code, memory bank stores:
- **Business context** - Why this project exists
- **Architectural decisions** - What we chose and why
- **Project history** - How we got here
- **Future plans** - Where we're going

Update both when making major project changes.

## See Also

- `CLAUDE.md` - Claude Code development guidance
- `/pre-commit` - Quality checks before committing
- `.memory_bank/README.md` - Memory bank documentation

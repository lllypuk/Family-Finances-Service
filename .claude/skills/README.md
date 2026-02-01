# Project Skills

This directory contains custom skills for Claude Code that extend its capabilities for this project.

## Available Skills

### Development & Testing

#### `/test-frontend`

**Description**: Test the web frontend using agent-browser for interactive testing, HTMX verification, and screenshot
capture

**When to use**:

- Validate login flows and authentication
- Test HTMX dynamic updates
- Verify form submissions and validations
- Check session management
- Generate screenshots for documentation

**Example**:

```
/test-frontend /login
```

#### `/pre-commit`

**Description**: Run comprehensive pre-commit checks (format, test, lint) before committing code

**When to use**:

- Before every git commit
- After completing any development work
- Before creating a pull request
- When fixing bugs to verify no regressions

**Example**:

```
/pre-commit
```

### Database Management

#### `/db-backup`

**Description**: Create a backup of the SQLite database

**When to use**:

- Before major schema changes or migrations
- Before bulk data operations
- As part of regular backup schedule
- Before testing destructive operations

**Example**:

```
/db-backup
```

#### `/db-shell`

**Description**: Open interactive SQLite shell to query and inspect the database

**When to use**:

- Inspect database schema and data
- Run ad-hoc queries for debugging
- Verify migration results
- Export data or generate reports

**Example**:

```
/db-shell
```

#### `/migrate-create`

**Description**: Create a new database migration with up/down SQL files

**When to use**:

- Adding new tables or columns
- Modifying database schema
- Creating indexes or constraints
- Removing deprecated fields

**Example**:

```
/migrate-create add_user_preferences
```

### Deployment & Operations

#### `/docker-up`

**Description**: Start the application in Docker with SQLite database

**When to use**:

- Testing production-like environment
- Deploying locally with Docker
- Debugging container issues
- Verifying Docker build and configuration

**Example**:

```
/docker-up
```

### Documentation

#### `/memory-update`

**Description**: Update project documentation in .memory_bank/ to reflect architectural changes, decisions, and current
state

**When to use**:

- After major architectural changes (e.g., PostgreSQL → SQLite)
- When completing significant features
- Recording important decisions and rationale
- Updating testing strategies or coverage
- Documenting new workflows or patterns

**Example**:

```
/memory-update tech_stack
/memory-update "SQLite migration"
```

## Skill Structure

Each skill follows the [Agent Skills](https://agentskills.io) open standard:

```
skill-name/
├── SKILL.md           # Main skill file with YAML frontmatter
└── [optional files]   # Templates, examples, scripts, etc.
```

### SKILL.md Format

```yaml
---
name: skill-name
description: What this skill does and when to use it
disable-model-invocation: true  # Only user can invoke (optional)
user-invocable: false           # Hide from menu (optional)
argument-hint: [ args ]           # Autocomplete hint (optional)
allowed-tools: Bash(make *)     # Tools accessible without approval (optional)
---

Markdown content with instructions for Claude...
```

## Creating New Skills

1. **Create directory**:
   ```bash
   mkdir -p .claude/skills/my-skill
   ```

2. **Create SKILL.md**:
    - Add YAML frontmatter between `---` markers
    - Include `name` and `description`
    - Write clear instructions in markdown

3. **Test the skill**:
    - Invoke with `/my-skill`
    - Or let Claude load it automatically based on description

4. **Document it**:
    - Add to this README
    - Update CLAUDE.md if relevant

## Skill Invocation

Skills can be invoked in two ways:

1. **Manual invocation**: Type `/skill-name` to run it directly
2. **Automatic loading**: Claude loads skills when relevant to conversation

Control invocation behavior with frontmatter:

- `disable-model-invocation: true` - Only manual invocation allowed
- `user-invocable: false` - Only automatic loading (hidden from menu)

## Best Practices

1. **Descriptive names**: Use lowercase with hyphens (e.g., `test-frontend`)
2. **Clear descriptions**: Help Claude know when to use the skill
3. **Focused purpose**: One skill = one clear responsibility
4. **Include examples**: Show common usage patterns
5. **Document arguments**: Use `argument-hint` for expected parameters
6. **Test thoroughly**: Verify both manual and automatic invocation

## See Also

- [Claude Code Skills Documentation](https://code.claude.com/docs/en/skills)
- [Agent Skills Standard](https://agentskills.io)
- Project CLAUDE.md for development guidelines
- `.claude/settings.local.json` for permissions

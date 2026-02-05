# Self-Hosted Deployment Tasks

This directory contains detailed task specifications for implementing self-hosted deployment capabilities for Family
Budget Service.

## Overview

These tasks transform the application from a development-ready state to a production-ready self-hosted solution that can
be easily deployed on any Linux VM.

## Task List

| ID  | Task                                                          | Priority | Status | Complexity  |
|-----|---------------------------------------------------------------|----------|--------|-------------|
| 001 | [Installation Script](001-install-script.md)                  | HIGH     | TODO   | Medium-High |
| 002 | [Reverse Proxy Configuration](002-reverse-proxy-config.md)    | HIGH     | TODO   | Medium      |
| 003 | [Production Docker Compose](003-docker-compose-production.md) | HIGH     | TODO   | Medium      |
| 004 | [Systemd Service Files](004-systemd-service.md)               | MEDIUM   | TODO   | Medium      |
| 005 | [Upgrade Script](005-upgrade-script.md)                       | MEDIUM   | TODO   | Medium-High |
| 006 | [Security Hardening](006-security-hardening.md)               | HIGH     | TODO   | Medium      |
| 007 | [Deployment Documentation](007-deployment-documentation.md)   | MEDIUM   | TODO   | Medium      |
| 008 | [Uninstall Script](008-uninstall-script.md)                   | LOW      | TODO   | Low         |

## Implementation Phases

### Phase 1: Core Deployment (Critical)

**Goal**: Enable basic secure deployment

1. **Task 001**: Installation script for automated setup
2. **Task 002**: Nginx/Caddy configs for TLS/SSL
3. **Task 003**: Production Docker Compose

**Deliverable**: User can deploy with single command and have HTTPS working

### Phase 2: Operations (Important)

**Goal**: Enable day-to-day operations

4. **Task 004**: Systemd service for native deployment
5. **Task 005**: Upgrade script with rollback
6. **Task 006**: Security hardening (firewall, fail2ban)

**Deliverable**: Automated backups, safe updates, brute-force protection

### Phase 3: Documentation (Recommended)

**Goal**: Self-service deployment

7. **Task 007**: Comprehensive deployment guide
8. **Task 008**: Clean uninstall script

**Deliverable**: Non-technical users can deploy and manage

## Directory Structure After Implementation

```
deploy/
├── scripts/
│   ├── install.sh           # Task 001
│   ├── upgrade.sh           # Task 005
│   ├── backup.sh            # Task 004
│   ├── health-check.sh      # Task 004
│   ├── setup-firewall.sh    # Task 006
│   ├── setup-fail2ban.sh    # Task 006
│   ├── check-security.sh    # Task 006
│   └── uninstall.sh         # Task 008
├── nginx/                   # Task 002
│   ├── nginx.conf
│   ├── conf.d/
│   │   └── family-budget.conf.template
│   └── snippets/
│       ├── ssl-params.conf
│       ├── security-headers.conf
│       └── proxy-params.conf
├── caddy/                   # Task 002
│   └── Caddyfile.template
├── systemd/                 # Task 004
│   ├── family-budget.service
│   ├── family-budget-backup.service
│   └── family-budget-backup.timer
├── fail2ban/                # Task 006
│   ├── family-budget.conf
│   └── jail.local
├── docker-compose.prod.yml  # Task 003
├── docker-compose.caddy.yml # Task 003
├── docker-compose.minimal.yml # Task 003
└── .env.production.example  # Task 003

docs/
├── DEPLOYMENT.md            # Task 007
├── SECURITY.md              # Task 006
├── BACKUP.md                # Task 007
├── TROUBLESHOOTING.md       # Task 007
└── FAQ.md                   # Task 007
```

## Dependencies Graph

```
Task 001 (install.sh)
    ├── depends on → Task 002 (nginx/caddy configs)
    └── depends on → Task 003 (docker-compose.prod.yml)

Task 005 (upgrade.sh)
    ├── depends on → Task 003 (docker-compose)
    └── depends on → Task 004 (systemd)

Task 006 (security)
    └── depends on → Task 002 (nginx for rate limiting)

Task 007 (documentation)
    └── depends on → Tasks 001-006 (all scripts)
```

## Getting Started

To implement these tasks:

1. Start with **Task 003** (docker-compose.prod.yml) - foundation for everything
2. Then **Task 002** (reverse proxy) - enables HTTPS
3. Then **Task 001** (install script) - combines everything
4. Continue with remaining tasks in priority order

## Testing Strategy

Each task includes a testing checklist. Full integration testing should be done on:

- Fresh Ubuntu 22.04 LTS VM
- Fresh Ubuntu 24.04 LTS VM
- Fresh Debian 12 VM

Use cloud providers like DigitalOcean, Hetzner, or Vultr for testing (cheapest VMs are sufficient).

## Notes

- All scripts use `/opt/family-budget` as the default installation directory
- Scripts are designed to be idempotent (safe to run multiple times)
- Data preservation is prioritized in all destructive operations
- Both Docker and native (systemd) deployments are supported

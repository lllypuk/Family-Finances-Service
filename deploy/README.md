# Deployment

One topology: the application and Caddy in a single compose on a home mini-server, one family per
instance ([spec 005](../docs/specs/005-api-only-redesign.md)). Caddy terminates TLS with Let's Encrypt
and proxies everything to `app:8080`; the service itself is never exposed on the host.

| File | Purpose |
|---|---|
| `docker-compose.yml` | `app` (built from source) + `caddy`, network `172.20.0.0/16` |
| `caddy/Caddyfile` | TLS, security headers, JSON access log, `log_skip @health` |
| `.env.example` | template for `.env` — `DOMAIN`, `ACME_EMAIL`, paths, `BACKUP_KEEP` |
| `scripts/install.sh` | first install: Docker, firewall, `.env`, build, `up` |
| `scripts/upgrade.sh` | git ref → backup → rebuild → health, automatic rollback |
| `scripts/uninstall.sh` | removal, `--keep-data` keeps the database and backups |
| `scripts/health-check.sh` | `GET /health` with retries, for monitoring (`HEALTH_URL`, default `https://$DOMAIN/health` with `DOMAIN` read from `$INSTALL_DIR/.env`: port 8080 is not published) |

There are **no secrets**: authentication is bearer tokens stored in the database.

## Requirements

Ubuntu 22.04/24.04, Debian 11/12 or Rocky/AlmaLinux 9; `git` and outbound network access.
512MB RAM is sized for the image build on the server — the running service needs 128–256MB.

**The image is built from source, not pulled.** Nothing is published to GHCR for this installation
until the first `v*` tag ([`docker.yml`](../.github/workflows/docker.yml)); `install.sh` clones the
repository into `/opt/family-budget/src` (`REPO_GIT_URL` / `REPO_REF`, default upstream `main`) and
builds there. `VERSION` comes from `git describe` in that checkout and ends up in `GET /health`.

## Install

1. Point an A record for your domain at the external IP and forward 80 and 443 to the mini-server.
   Caddy needs both reachable from the internet to obtain a certificate.
2. Clone and run the installer as root:

   ```bash
   git clone https://github.com/lllypuk/Family-Finances-Service.git
   cd Family-Finances-Service
   sudo ./deploy/scripts/install.sh --domain ffs.shatrov.tech --email you@example.com
   ```

   `curl … | sudo bash` does not work: the script sources `lib/*.sh` from its own directory.
   `--dry-run` prints every mutating command instead of running it; `--reinstall` moves an existing
   `/opt/family-budget` to `/opt/family-budget.backup.<ts>` — without it a repeated run keeps
   `data/`, `backups/` and `.env`.

3. Create the family and the first admin over ssh (there is no HTTP bootstrap):

   ```bash
   cd /opt/family-budget && printf 'YourPassword1!\n' | docker compose exec -T app \
       /app/family-budget-service setup --family 'Family' --currency RUB \
       --timezone Europe/Moscow --email you@example.com \
       --first-name Name --last-name Surname --password-stdin
   ```

   Until this runs, `POST /api/v1/auth/login` answers `409 SETUP_REQUIRED` and `/health` reports
   `setup_complete: false` while staying `200`.

4. The second user is created by the admin over the API — `POST /api/v1/users`, no invites.

5. Add the daily backup to the host crontab:

   ```cron
   0 3 * * * cd /opt/family-budget && docker compose run --rm --no-deps -T app backup
   ```

   `exec` bypasses the ENTRYPOINT, hence the full path to the binary.

## Layout on the server

```
/opt/family-budget/
├── docker-compose.yml     # copied from deploy/
├── caddy/Caddyfile        # copied from deploy/
├── .env                   # rendered from .env.example, chmod 600
├── src/                   # git checkout, the build context (BUILD_CONTEXT=./src)
├── data/budget.db         # SQLite, mounted at /data
├── backups/               # backup_<date>_<time>.db, mounted at /backups
└── logs/caddy/access.log  # JSON, rolled at 100MB × 5
```

`data/`, `backups/` and `logs/` belong to `1000:1000` — the UID the image runs as. Docker creates a
missing bind-mount directory as root, and SQLite then cannot open the database.

`TRUSTED_PROXIES=172.20.0.0/16` is set in `environment:` in the compose file and nowhere else
(`environment` overrides `env_file`, so a copy in `.env` would be dead). It has to match
`ipam.config.subnet`: with a mismatch the login limiter sees every request as coming from the Caddy
container and both users share one per-IP bucket.

## Backups

`family-budget-service backup` runs `VACUUM INTO` against the live database and keeps the newest
`BACKUP_KEEP` (default 30) files in `/backups`. The same retention applies to `POST /api/v1/backups`.
Off-site copies: `GET /api/v1/backups/{name}/download` from the phone, or `scp`.

Restore is manual, over ssh — there is no restore endpoint or subcommand:

```bash
cd /opt/family-budget
docker compose down
cp backups/backup_20260904_030000.db data/budget.db
chown 1000:1000 data/budget.db
rm -f data/budget.db-wal data/budget.db-shm
docker compose up -d
```

Check a backup before trusting it: `sqlite3 backups/<file>.db 'PRAGMA integrity_check'`.

## Upgrade and rollback

```bash
sudo /opt/family-budget/src/deploy/scripts/upgrade.sh --version v0.1.0
```

It records the current ref, takes a database copy with `docker compose run --rm --no-deps app backup`
(a `run` container works whether `app` is up or down), then fetches the target ref, copies
`deploy/docker-compose.yml` and `deploy/caddy/Caddyfile` from the new checkout over the installed
copies (the image is rebuilt from `src/`, the topology lives next to it and would otherwise stay on
the previous release), rebuilds, restarts and waits for `/health`. A changed Caddyfile is applied with
`caddy reload` — a bind-mounted file changing does not recreate the container. The copy is taken **before** the rebuild on purpose: the subcommand applies
migrations when it opens the database, so a copy taken with the new image would already carry the new
schema and there would be nothing to roll back to. A failed health check rolls back the ref, the database, `.env`, the compose file and the Caddyfile automatically;
`--no-rollback` disables that, and `upgrade.sh rollback` replays the most recent
`backups/upgrade_<ts>/` by hand. If the rollback cannot rebuild the previous image or restore the
deploy files, it leaves the service **stopped**: the failed upgrade's image over the restored
database would migrate it and undo the rollback.

## Operations

```bash
cd /opt/family-budget
docker compose ps
docker compose logs -f app            # application, JSON slog
tail -f logs/caddy/access.log         # requests, /health excluded
docker compose restart app
curl -s https://ffs.shatrov.tech/health
```

Removal: `sudo ./uninstall.sh --keep-data` stops and deletes the containers, images and networks but
leaves `/opt/family-budget` and the Caddy volumes; without the flag everything goes.

## Troubleshooting

- **`up` fails with "Pool overlaps with other one"** — a network from an older installation sits on
  `172.20.0.0/16`. `install.sh` removes `family-budget_family-budget-net` before starting; for any
  other leftover, `docker network ls` then `docker network rm <name>`.
- **No certificate** — 80 and 443 must reach the server from the internet, and the A record must
  already resolve. `docker compose logs caddy` shows the ACME error.
- **`app` unhealthy, "unable to open database file"** — ownership of `data/`:
  `sudo chown -R 1000:1000 data backups logs`.
- **A `500` with no detail in the response** — the message is only in the log
  (`internal/application/error_handler.go`); read `docker compose logs app`.

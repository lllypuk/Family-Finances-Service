# Deployment

One family per instance ([spec 005](../docs/specs/005-api-only-redesign.md)). The image is built by
GitLab CI and pulled from `registry.gitlab.shatrov.tech`; nothing is compiled on the server.

| File | Purpose |
|---|---|
| `docker-compose.yml` | `app` (image from the registry) + `caddy`, network `172.20.0.0/16` |
| `docker-compose.proxied.yml` | overlay for a host where 80/443 already belong to another Caddy |
| `caddy/Caddyfile` | TLS, security headers, JSON access log, `log_skip @health` |
| `.env.example` | template for `.env` — `FFS_IMAGE`, layout, `DOMAIN`, paths, `BACKUP_KEEP` |
| `release.sh` | host-side update: registry login, DB snapshot, image swap, `up --wait` |
| `scripts/install.sh` | first install: Docker, firewall, `.env`, pull, `up` |
| `scripts/uninstall.sh` | removal, `--keep-data` keeps the database and backups |
| `scripts/health-check.sh` | `GET /health` with retries, for monitoring |

There are **no secrets**: authentication is bearer tokens stored in the database.

## Two layouts

`.env` decides, through `COMPOSE_FILE`, which one the machine runs — the file set is the same.

**Own TLS** (default): compose starts Caddy, which obtains a Let's Encrypt certificate for `DOMAIN`
and proxies to `app:8080`. The host must own ports 80 and 443.

**Behind a shared terminator** (`COMPOSE_FILE=docker-compose.yml:docker-compose.proxied.yml`): the
bundled Caddy moves into the `own-tls` profile and never starts, and `app` joins the external docker
network `edge`, where the other Caddy reaches it as `ffs:8080`. This is how **mini-server** runs it —
80 and 443 there belong to the shatrov.tech landing. Two consequences:

- `TRUSTED_PROXIES` is overridden with `FFS_EDGE_SUBNET`, the subnet of `edge`. `X-Forwarded-For`
  arrives from a container outside `internal`, and without this the login limiter would treat both
  users as one client.
- The headers in `caddy/Caddyfile` are not read at all. HSTS and CSP have to travel with the site
  block in the shared Caddy:

  ```caddyfile
  ffs.{$DOMAIN} {
    encode zstd gzip
    reverse_proxy ffs:8080
    header {
      -Server
      Strict-Transport-Security "max-age=31536000; includeSubDomains"
      Content-Security-Policy "default-src 'none'; frame-ancestors 'none'"
      X-Content-Type-Options "nosniff"
      Referrer-Policy "no-referrer"
    }
  }
  ```

## Requirements

Ubuntu 22.04/24.04, Debian 11/12 or Rocky/AlmaLinux 9; outbound network access. 256MB RAM is enough —
the server only runs the image. The registry is private, so the host needs `docker login
registry.gitlab.shatrov.tech` once, with a personal or deploy token that can read packages.

## Install

1. Point an A record for your domain at the external IP. For the own-TLS layout forward 80 and 443 to
   the host: Caddy needs both reachable from the internet to obtain a certificate.
2. Clone and run the installer as root:

   ```bash
   git clone ssh://git@gitlab.shatrov.tech:2222/shatrov.tech/family-finances-service.git
   cd family-finances-service
   sudo ./deploy/scripts/install.sh --domain ffs.shatrov.tech --email you@example.com
   ```

   The compose files and the Caddyfile are taken from the checkout you run it in; nothing is cloned.
   `curl … | sudo bash` does not work: the script sources `lib/*.sh` from its own directory.
   `--image` overrides the image, `--dry-run` prints every mutating command instead of running it,
   `--reinstall` moves an existing `/opt/family-budget` aside — without it a repeated run keeps
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

   `run` starts a throwaway container, so the job works whether `app` is up, down or unhealthy.

## Layout on the server

```
/opt/family-budget/          # mini-server: /home/sasha/ffs
├── docker-compose.yml       # from deploy/, replaced by every deploy
├── docker-compose.proxied.yml
├── caddy/Caddyfile
├── release.sh
├── .env                     # stays on the server; CI rewrites one line, FFS_IMAGE
├── data/budget.db           # SQLite, mounted at /data
├── backups/                 # backup_<date>_<time>.db, mounted at /backups
└── logs/caddy/access.log    # own-TLS layout only, JSON, rolled at 100MB × 5
```

`data/`, `backups/` and `logs/` belong to `1000:1000` — the UID the image runs as. Docker creates a
missing bind-mount directory as root, and SQLite then cannot open the database.

`TRUSTED_PROXIES` is set in `environment:`, not in `.env` (`environment` overrides `env_file`, so a
copy under that name would be dead). In the own-TLS layout it is `FFS_INTERNAL_SUBNET`, the same
value the network is created with; in the proxied layout the overlay replaces it with
`FFS_EDGE_SUBNET`, the subnet of `edge`.

## Deploy

Every green pipeline on `main` deploys: CI builds `…/family-finances-service:main-<sha>`, copies this
directory to the host and runs `release.sh`, which logs into the registry, snapshots the database,
rewrites `FFS_IMAGE` in `.env`, pulls and restarts with `--wait`. A tag `vX.Y.Z` does the same with
the tag as the image version. The pipeline fails if the service does not reach `healthy`, and the
`smoke` job then checks `/health` over the public domain.

The snapshot is taken **before** the image is swapped, with the container that is still running: the
new release migrates the database at startup, and a copy taken afterwards is not what a rollback
needs.

Rollback is a previous image, no rebuild involved:

```bash
cd /home/sasha/ffs
grep FFS_IMAGE .env                      # what is running now
sed -i 's|^FFS_IMAGE=.*|FFS_IMAGE=registry.gitlab.shatrov.tech/shatrov.tech/family-finances-service:main-1a2b3c4d|' .env
docker compose up -d --wait
```

If the newer release migrated the database, restore the snapshot it took first — see below. Image
tags older than the registry cleanup policy are gone; re-running the old pipeline's `release-image`
job builds that commit again.

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

## Operations

```bash
cd /opt/family-budget
docker compose ps
docker compose logs -f app            # application, JSON slog
tail -f logs/caddy/access.log         # own-TLS layout; otherwise the shared Caddy has them
docker compose restart app
curl -s https://ffs.shatrov.tech/health
```

Removal: `sudo ./uninstall.sh --keep-data` stops and deletes the containers, images and networks but
leaves the installation directory and the Caddy volumes; without the flag everything goes.

## Troubleshooting

- **`pull access denied` / `unauthorized`** — the registry is private. `docker login
  registry.gitlab.shatrov.tech` on the host; in CI the job token does it.
- **`up` fails with "Pool overlaps with other one"** — the internal subnet is taken on this host.
  `docker network ls` and `docker network inspect` show by whom; put a free range into
  `FFS_INTERNAL_SUBNET` in `.env` (it feeds both the network and `TRUSTED_PROXIES`). A leftover
  network from an older installation is removed by `install.sh`, or by hand with
  `docker network rm <name>`.
- **`network edge not found`** — the proxied layout expects it to exist:
  `docker network create edge`.
- **No certificate** (own-TLS layout) — 80 and 443 must reach the server from the internet, and the A
  record must already resolve. `docker compose logs caddy` shows the ACME error.
- **`app` unhealthy, "unable to open database file"** — ownership of `data/`:
  `sudo chown -R 1000:1000 data backups logs`.
- **A `500` with no detail in the response** — the message is only in the log
  (`internal/application/error_handler.go`); read `docker compose logs app`.

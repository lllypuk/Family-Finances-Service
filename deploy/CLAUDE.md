# deploy

**Deployment** is `deploy/`: `docker-compose.yml` (`app` pulled from `registry.gitlab.shatrov.tech` + Caddy),
`docker-compose.proxied.yml` (the overlay for mini-server, where 80/443 belong to the shatrov.tech Caddy: the
bundled Caddy goes into the `own-tls` profile and `app` joins the external network `edge` as `ffs`),
`caddy/Caddyfile`, `.env.example`, `release.sh` and the `install`/`uninstall`/`health-check` scripts — see
`deploy/README.md`. The `.env` on the server holds the machine's layout and `FFS_IMAGE`; a deploy rewrites that
one line, so it never travels from the repository. Backups in production are the `backup` subcommand from a host
cron job; restore is manual over ssh. There is no `upgrade.sh` any more — the pipeline is the upgrade path.
Prometheus `/metrics` is on a second listener: `METRICS_ADDR` and the `prometheus.io/*` labels live in
`docker-compose.yml` (topology, not `.env`), the port is not published, and the dev compose leaves it off.
Screenshot recognition: `LLM_*` are passed through `environment:` from `.env` (empty = off, `503`); on mini the
Ollama daemon is the host's address, not `localhost`. The upload route needs its own body limit in the Caddy in
front — `caddy/Caddyfile` has it, the production vhost lives in the landing repository (`deploy/README.md`).

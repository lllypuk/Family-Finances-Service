---
name: run-local
description: Start the server locally, create the family from the CLI and call /api/v1 with curl
---

# Local testing

`make run-local` — localhost:8080, SQLite at `./data/budget.db`, `LOG_LEVEL=debug`. Start it from the repo root
(`./migrations` is resolved relative to the CWD).

- Curl the local server with `--noproxy '*'`: `curl -s --noproxy '*' 127.0.0.1:8080/health`
- The family is created from the CLI, not over HTTP. From the repo root, with the same `DATABASE_PATH` the server
  uses (`./data/budget.db` is the default for both):

  ```bash
  printf 'Admin1234!\n' | go run ./cmd/server setup --family 'Test Family' --currency RUB \
      --timezone Europe/Moscow --email admin@test.com --first-name Admin --last-name Test --password-stdin
  curl -s --noproxy '*' -X POST 127.0.0.1:8080/api/v1/auth/login \
      -H 'Content-Type: application/json' -d '{"email":"admin@test.com","password":"Admin1234!"}'
  curl -s --noproxy '*' 127.0.0.1:8080/api/v1/me -H "Authorization: Bearer $TOKEN"
  ```

  `go run ./cmd/server reset-password --email … --password-stdin` sets a new password and revokes every session.
- Project skills in `.claude/skills/`: `/pre-commit`, `/db-backup`, `/db-shell`, `/docker-up`, `/migrate-create`,

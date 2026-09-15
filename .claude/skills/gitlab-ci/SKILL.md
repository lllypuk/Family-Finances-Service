---
name: gitlab-ci
description: GitLab CI pipeline, runner, GitHub mirror and pinned tool/image versions — read before editing .gitlab-ci.yml, docker/Dockerfile digests or deploy/docker-compose.yml
---

# CI

**CI is GitLab (`.gitlab-ci.yml`), the repository lives on `gitlab.shatrov.tech`**; GitHub is a read-only
mirror with no workflows, pushed by the `mirror:github` job over a GitHub deploy key (the native GitLab push
mirror hands out its public key only in the UI, which no script can pick up). The push is deliberately not
forced: a divergence means somebody wrote to the mirror by hand, and that should turn a job red rather than
disappear. The pipeline runs golangci-lint (`fmt --diff` + `run`), `shellcheck -e SC1091`
over `deploy/**` (SC1091 is off: the libs are sourced through a computed path), `make test-coverage`,
`govulncheck`, `make build`, `docker compose config` for all three layouts and `caddy validate`. `gosec` is
part of golangci-lint here, not a separate job: run standalone it ignores the `//nolint:gosec` suppressions and
the `.golangci.yml` exclusions, and fails on lines the linter deliberately passes.
On a merge request it also builds the image and curls `/health` inside it; on `main` and on a `vX.Y.Z` tag it
pushes the image to `registry.gitlab.shatrov.tech` and deploys to the mini-server (see "Deployment" below).
Both tag patterns (`.release-tags`, `.android-rules`) are exact on purpose: the Android client shares this
repository (decision A-13 in `docs/specs/005-api-only-redesign.md`) and releases under its own tag namespace,
which must not build or deploy the server — and a tag with a slash could not name a Docker image anyway.

The runner is a single instance-wide docker executor on home-server: `privileged = true`, `/certs/client`
(dind) and `/home/sasha/ci-cache:/ci-cache` (Go caches, hence `GOCACHE`/`GOMODCACHE` pointing there instead of
the `cache:` mechanism). Jobs carry no tags.

**Tool versions in CI are pinned exactly** (`GOLANGCI_LINT_VERSION`, `GOVULNCHECK_VERSION`),
never `@latest`; both `FROM` lines in `docker/Dockerfile` and the `caddy` image in `deploy/docker-compose.yml`
are pinned by digest, and `make caddy-validate` reads that digest back out of the compose file. Dependabot went
away with GitHub, so these digests are now bumped by hand — nothing watches them.

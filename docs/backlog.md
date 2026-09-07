# Бэклог задач для проекта "Family Finances Service"

## Отложенные обновления зависимостей

- **github.com/labstack/echo-contrib** — PR #66 закрыт как obsolete 2026-08; зависимость ушла
  вместе с веб-слоем (план 03), бампить нечего.

- **modernc.org/sqlite** — PR #77 закрыт как obsolete 2026-08, бамп до v1.48.1 применён
  коммитом `013d9e0` напрямую на main (Go modules batch bump).

- **github.com/go-playground/validator/v10** — PR #78 закрыт как obsolete, патч до v10.30.2
  применён коммитом `013d9e0`.

- **github.com/labstack/echo/v4** — PR #68 закрыт как obsolete, патч до v4.15.1 применён
  коммитом `013d9e0`.

- **golang.org/x/crypto** — PR #74 закрыт как obsolete. На main уже v0.50.0 (новее, чем
  target v0.49.0 в PR).

## Открытые находки аудита (август 2026)

Полные описания и шаги воспроизведения — в [docs/specs/](specs/README.md).
План [03-bearer-auth-web-removal](plans/completed/20260904-03-bearer-auth-web-removal.md) закрыл
S-03 (лимитер логина в `internal/auth/ratelimit.go`), `CSRF_SECRET` (вместе со всем cookie-конфигом),
`echo-contrib` и долг `application → web`.

- **Тег `v0.1.0` не поставлен** — действие владельца репозитория. `docker.yml` и `release.yml`
  уже указывают `file: docker/Dockerfile`, блокера нет. Ставить тег имеет смысл после перехода
  на API-only ([005](specs/005-api-only-redesign.md)): все пять планов закрыты, код и `openapi.yaml`
  совпадают, деплой — один compose с Caddy.

## Обновление зависимостей после переезда на GitLab

- **Автообновлений больше нет.** Dependabot остался на GitHub вместе с `.github/`, а GitLab CE
  своего аналога не даёт. Руками обновляются: модули Go (`go get -u`), дайджесты `FROM` в
  `docker/Dockerfile`, дайджест образа Caddy в `deploy/docker-compose.yml` и версии инструментов
  в `.gitlab-ci.yml` (`GOLANGCI_LINT_VERSION`, `GOVULNCHECK_VERSION`).
  `govulncheck` в пайплайне поймает уязвимую версию Go-зависимости, но не устаревший базовый образ.

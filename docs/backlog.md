# Бэклог задач для проекта "Family Finances Service"

## Отложенные обновления зависимостей

- **github.com/labstack/echo-contrib** — текущая v0.17.1. PR #66 закрыт как obsolete 2026-08.
  Используется в `internal/web/middleware/{session,csrf}.go` для cookie-сессий и CSRF.
  Делать осознанно отдельным коммитом после проверки API `session.Middleware`/`session.Store`
  и CSRF-хелперов между v0.17 и v0.50. Версия должна быть привязана к используемой версии
  Echo (в проекте используется `labstack/echo/v4 v4.15.4`).

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

- **S-03 — нет rate limiting на логин** ([002-security-audit.md](specs/002-security-audit.md#s-03)).
  Защита от перебора живёт только в конфигурациях nginx/Caddy и fail2ban, значит
  `docker-compose.minimal.yml` и native systemd не покрыты. Нужен отдельный план для защиты
  на уровне приложения.

- **`.github/workflows/docker.yml` не может собрать образ.** Шаг `docker/build-push-action`
  задан как `context: .` без `file:`, то есть ищет `./Dockerfile`, которого нет (файл лежит в
  `docker/Dockerfile`). Пуш тега уронит сборку на «failed to read dockerfile». См.
  [D-02](specs/004-deployment-readiness.md#d-02).

- **Тег `v0.1.0` не поставлен** — действие владельца репозитория. Ставить его можно только
  после починки `docker.yml`, иначе первый же релиз упадёт.

- **`CSRF_SECRET` — конфигурационная ручка, которую никто не читает.** `internal/config.go`
  объявляет и валидирует `Web.CSRFSecret` (в production — не плейсхолдер и не короче 32 символов),
  но CSRF-токены генерируются случайно и хранятся в сессии, которую подписывает `SESSION_SECRET`,
  так что значение переменной ни на что не влияет. Либо использовать её для подписи токенов, либо
  убрать из конфига, compose-файлов и документации.

## Инфраструктура Dependabot

- В `.github/dependabot.yml` указаны метки `dependencies`, `go`, `github-actions`, `docker`.
  На GitHub часть из них (`go`, `github-actions`, `docker`) не существуют, из-за чего каждый
  dependabot-PR стартует с warning "labels could not be found".
  Решение: создать эти метки в репозитории или убрать из конфига dependabot и оставить
  только `dependencies`.

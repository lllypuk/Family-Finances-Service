# Бэклог задач для проекта "Family Finances Service"

## Отложенные обновления зависимостей

- **github.com/labstack/echo-contrib** — текущая v0.17.1. PR #66 закрыт как obsolete 2026-08.
  Используется в `internal/web/middleware/{session,csrf}.go` для cookie-сессий и CSRF.
  Делать осознанно отдельным коммитом после проверки API `session.Middleware`/`session.Store`
  и CSRF-хелперов между v0.17 и v0.50. Версия должна быть привязана к используемой версии
  Echo (в проекте используется `labstack/echo/v4 v4.15.1`).

- **modernc.org/sqlite** — PR #77 закрыт как obsolete 2026-08, бамп до v1.48.1 применён
  коммитом `013d9e0` напрямую на main (Go modules batch bump).

- **github.com/go-playground/validator/v10** — PR #78 закрыт как obsolete, патч до v10.30.2
  применён коммитом `013d9e0`.

- **github.com/labstack/echo/v4** — PR #68 закрыт как obsolete, патч до v4.15.1 применён
  коммитом `013d9e0`.

- **golang.org/x/crypto** — PR #74 закрыт как obsolete. На main уже v0.50.0 (новее, чем
  target v0.49.0 в PR).

## Инфраструктура Dependabot

- В `.github/dependabot.yml` указаны метки `dependencies`, `go`, `github-actions`, `docker`.
  На GitHub часть из них (`go`, `github-actions`, `docker`) не существуют, из-за чего каждый
  dependabot-PR стартует с warning "labels could not be found".
  Решение: создать эти метки в репозитории или убрать из конфига dependabot и оставить
  только `dependencies`.

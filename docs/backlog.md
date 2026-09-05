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
  на API-only ([005](specs/005-api-only-redesign.md)) — после плана 04, когда контракт денег и дат
  совпадёт с `openapi.yaml`.

## Технический долг

- **`deploy/**` до плана 05 описывает прежнюю веб-сборку**: compose-файлы требуют
  `SESSION_SECRET`/`CSRF_SECRET`, которые приложение не читает (Makefile `COMPOSE_VALIDATE_ENV`
  подставляет заглушки ради `make compose-config`), nginx/Caddy/fail2ban ограничивают `/login`.
  Чинить точечно не нужно — план 05 заменяет каталог целиком.

- **`--timezone` в CLI `setup` принимается и не сохраняется** — у `user.Family` нет колонки до плана 04.

- **`ensureNotLastAdmin` — read-then-write без транзакции** (`internal/services/user_service.go`): два
  одновременных `PATCH`, понижающих друг друга, оба увидят «второй админ ещё есть». Для одной семьи из
  двух пользователей это не воспроизводимо на практике; лечится транзакционным вариантом репозитория
  (`_txlock=immediate` уже в DSN), когда репозитории получат tx-API.

- **Комментарии в `deploy/scripts/{install,upgrade}.sh` двуязычны.** Пояснения, добавленные
  в августе 2026, написаны по-русски поверх англоязычных скриптов (47 и 71 строка
  соответственно). CLAUDE.md требует держаться стиля файла; привести их к одному языку
  стоит отдельным проходом, целиком по файлу.

## Инфраструктура Dependabot

- В `.github/dependabot.yml` указаны метки `dependencies`, `go`, `github-actions`, `docker`.
  На GitHub часть из них (`go`, `github-actions`, `docker`) не существуют, из-за чего каждый
  dependabot-PR стартует с warning "labels could not be found".
  Решение: создать эти метки в репозитории или убрать из конфига dependabot и оставить
  только `dependencies`.

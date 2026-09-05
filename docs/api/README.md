# API-контракт

`openapi.yaml` (OpenAPI 3.1, один файл) — единственное описание HTTP-контракта сервиса.
Он описывает **целевое** состояние из [spec 005](../specs/005-api-only-redesign.md); код приводят
к нему планы [02](../plans/completed/20260904-02-api-completeness.md)–[04](../plans/20260904-04-schema-and-money.md).
До конца плана 04 расхождения ожидаемы: деньги ещё `float64`, даты — RFC3339. Аутентификация уже bearer (план 03).

## Как читать

- Все пути даны целиком (`/api/v1/...`), `GET /health` — вне версионированного префикса.
- Роль, которой доступна операция, написана в `description` каждой операции: «Только admin»,
  «admin и member», «Доступно любой роли».
- Успешный ответ всегда `{"data": ..., "meta": {...}}`, ошибка — `{"error": {...}, "meta": {...}}`
  (схема `Error`). Списки несут `meta.pagination {limit, offset, total}`.
- Суммы — `*_minor`, целые в минимальных единицах валюты семьи. Даты операций — `format: date`,
  служебные метки — `format: date-time` (UTC).

## Правило синхронизации

Роут `/api/v1/*` (плюс `GET /health`), не описанный в `openapi.yaml`, роняет `make test`:
`tests/integration/openapi_coverage_test.go` сверяет `e.Routes()` тестового сервера со списком
операций. Обратное — описанная, но не реализованная операция — допустимо до конца плана 04.

Значит: **новый роут добавляется вместе с описанием в этом файле**, в том же коммите.

## Валидация

```bash
npx --yes @redocly/cli@latest lint docs/api/openapi.yaml
```

В CI линтер спецификации не добавлен намеренно: он тянет node в Go-пайплайн, а покрытие роутов
уже проверяет Go-тест. Результат прогона фиксируется в PR, меняющем контракт.

## Kotlin-клиент для Android

```bash
npx --yes @openapitools/openapi-generator-cli generate \
  -i docs/api/openapi.yaml -g kotlin -o build/android-client \
  --additional-properties=library=jvm-retrofit2,serializationLibrary=kotlinx_serialization,dateLibrary=java8
```

Генератор — деталь Android-проекта, а не этого репозитория: здесь нет ни node-зависимостей,
ни шага сборки клиента. Проверить после генерации: `Money` → `Long`, `CalendarDate` → `LocalDate`.

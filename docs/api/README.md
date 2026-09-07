# API-контракт

`openapi.yaml` (OpenAPI 3.1, один файл) — единственное описание HTTP-контракта сервиса.
Код совпадает с ним: планы [02](../plans/completed/20260904-02-api-completeness.md)–04 привели
к спецификации из [spec 005](../specs/005-api-only-redesign.md) роуты, конверт ошибок,
bearer-аутентификацию, деньги в минимальных единицах и календарные даты.

## Как читать

- Все пути даны целиком (`/api/v1/...`), `GET /health` — вне версионированного префикса.
- Роль, которой доступна операция, написана в `description` каждой операции: «Только admin»,
  «admin и member», «Доступно любой роли».
- Успешный ответ всегда `{"data": ..., "meta": {...}}`, ошибка — `{"error": {...}, "meta": {...}}`
  (схема `Error`). Списки несут `meta.pagination {limit, offset, total}`.
- Суммы — `*_minor`, целые в минимальных единицах валюты семьи. Даты операций — `format: date`,
  служебные метки — `format: date-time` (UTC).

## Правило синхронизации

`tests/integration/openapi_coverage_test.go` сверяет `e.Routes()` тестового сервера со списком
операций **в обе стороны**: роут без описания роняет `make test`, описание без роута — тоже.
Исключений нет.

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
  --additional-properties=library=jvm-retrofit2,serializationLibrary=kotlinx_serialization \
  --additional-properties=dateLibrary=java8,useCoroutines=true,useResponseAsReturnType=true \
  --additional-properties=packageName=tech.shatrov.familyfinances.core.api \
  --additional-properties=apiPackage=tech.shatrov.familyfinances.core.api \
  --additional-properties=modelPackage=tech.shatrov.familyfinances.core.api \
  --additional-properties=sourceFolder=kotlin \
  --global-property=apis,models,apiDocs=false,modelDocs=false,apiTests=false,modelTests=false
```

Набор параметров не сокращается: без `useCoroutines` интерфейсы отдают `Call<T>`, без
`useResponseAsReturnType` тело ошибки недоступно, без `sourceFolder` вывод уезжает в
`generated/src/main/kotlin`, а без `--global-property` рядом с кодом появляются чужие
`build.gradle`, `README.md` и `docs/`. Те же значения держит задача `generateApiClient`
в `android/core/api/build.gradle.kts`.

Генератор — деталь Android-проекта, а не этого репозитория: здесь нет ни node-зависимостей,
ни шага сборки клиента. Проверить после генерации: `Money` → `Long`, `CalendarDate` → `LocalDate`.

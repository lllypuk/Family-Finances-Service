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
- Файлы идут `multipart/form-data`: массив `type: string, format: binary` под одним именем поля и
  `encoding: <поле>: {style: form, explode: true}` — каждый файл отдельной частью с тем же именем
  (`images`, `images`, …). Без `encoding` генератор волен склеить массив в одну часть, а сервер читает
  части потоком и считает их по имени. Kotlin-клиент получает `List<MultipartBody.Part>`; проверить
  сигнатуру после генерации. Ошибка части — `422` с `field: images[i]`.
- Суммы — `*_minor`, целые в минимальных единицах валюты семьи. Даты операций — `format: date`,
  служебные метки — `format: date-time` (UTC).

## Счета и сверка

- `account_id` у операции необязателен. В `PUT /transactions/{id}` отсутствие поля (и `null`) оставляет
  счёт, отвязка — `clear_account: true`: клиент шлёт с `explicitNulls = false` и `null` передать не
  может. Выборка «без счёта» — `?unassigned=true`, не `account_id=none`, чтобы параметр оставался UUID.
- Сверки — `PUT`/`DELETE /accounts/{id}/reconciliations/{month}`, сводка — `GET /stats/reconciliation`:
  объект без пагинации, `recorded_minor` считается по расходам на чтении, `bank_expense_minor`,
  `diff_minor`, `note`, `updated_at` — `null`, пока сверки нет.

## Активы, пассивы и капитал

- `/holdings` — справочник позиций; `side` — `enum`, `kind` — строка с перечнем в описании: новый вид не
  должен ронять установленный клиент, неизвестный рисуется как `other`. `side` задаётся только при создании.
- Снимки — `PUT`/`DELETE /holdings/{id}/values/{date}`, история — `GET /holdings/{id}/values`, новые сверху.
  Будущая дата — `422`; `current` у позиции — последний снимок не позже сегодня в зоне семьи.
- `GET /stats/net-worth` — месячный ряд, значение позиции переносится вперёд до следующего снимка, архивные
  входят. Итоги — `int64` без `maximum` (сумма может превысить `Money`); `to` позже сегодня — `422`.
  Капитал на экране берётся из последней корзины ряда, а не суммой списка.
- План позиции (сервер `v0.8.0`): `monthly_income_minor`, `monthly_expense_minor` и `plan_updated_at` у
  `Holding`. Числа сервер шлёт всегда (`0` — нет), дату — только при плане; все три не `required`, чтобы
  клиент разбирал ответ старого сервера. В `PUT` нет поля — не трогать, `0` — убрать. Итог «План в месяц»
  считает клиент.

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
  --global-property=apis,models,supportingFiles=CollectionFormats.kt,apiDocs=false,modelDocs=false,apiTests=false,modelTests=false
```

Набор параметров не сокращается: без `useCoroutines` интерфейсы отдают `Call<T>`, без
`useResponseAsReturnType` тело ошибки недоступно, без `sourceFolder` вывод уезжает в
`generated/src/main/kotlin`, а без `--global-property` рядом с кодом появляются чужие
`build.gradle`, `README.md` и `docs/`, а без `supportingFiles=CollectionFormats.kt` не собирается
сгенерированный код: этот файл импортируют все интерфейсы. Те же значения держит задача
`generateApiClient` в `android/core/api/build.gradle.kts`.

Боевая генерация идёт не этой командой, а задачей `:core:api:generateApiClient`
(`make -C android api-gen`, без node); вывод коммитится в `android/core/api/generated`, свежесть
держат `make -C android api-check` и джоба `android:api-check`. Команда выше — для разовой проверки
контракта. Проверить после генерации: `Money` → `Long`, `CalendarDate` → `LocalDate`.

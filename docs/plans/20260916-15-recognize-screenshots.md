# План 15. Распознавание скриншотов банковских операций

## Overview

Пользователь шарит в приложение скриншот уведомления банка или списка операций (или выбирает его из
галереи / снимает камерой), сервер отдаёт картинки модели через `github.com/lllypuk/llm` и
возвращает **кандидатов** операций. Приложение показывает их списком с правкой, человек
подтверждает, запись идёт обычными `POST /transactions` с клиентским `id`. Ничего не создаётся без
взгляда человека, картинки на сервере не хранятся, фоновых заданий нет.

Решения владельца (16.09.2026): предпросмотр и подтверждение; синхронный вызов модели внутри
HTTP-запроса; старт на `llm` v0.2.0 с Ollama, переезд на `Router` (v0.4.0, план DPP) — отдельным
заходом; входы — share-intent, пикер галереи, камера.

## Context (from discovery)

- Библиотека `/home/sasha/projects/shatrov.tech/llm` v0.2.0: `llm.New(ollama.New(host), timeout)`,
  `Chat(ctx, Request{Task, Model, Messages[{Role, Text, Images}], Output{Mode}, Temperature})`,
  повторы `immediate`/`after_delay`, `Budget()` — худший случай всех попыток (при `Attempts=2`,
  `Timeout=60s`, `MaxRetryAfter=10s` — **130 с**, `client.go:62,285`), `Observer{Attempt, Call}`,
  `Report` и у отказа. Ответ модели принимается как текст (`ollama.go:143`): предметный мусор
  повторов **не** вызывает — это забота потребителя. `ollama.Provider` ходит через `http.DefaultClient`
  (`ollama.go:208`) — транспортного таймаута у плеча нет, единственная граница — контекст попытки.
- Прод: контейнер на `mini`, там же демон Ollama `http://192.168.1.10:11434` с облачными моделями;
  ключ ollama.com живёт у демона — секретов в конфиге сервиса не появляется. Прод-раскладка —
  `deploy/docker-compose.proxied.yml`: собственный Caddy этого репозитория **не используется**, vhost
  `ffs.shatrov.tech` живёт в репозитории лендинга (`deploy/Caddyfile.prod`).
- **Пробы 16.09.2026 на `gemma4:31b-cloud`** (curl с хоста): `format` со JSON-схемой облачная модель
  **игнорирует** — отвечает своим JSON в ограде ```` ```json ````; `format: "json"` с формой,
  продиктованной в системном промпте, отдаёт ровно её и на тексте, и на картинке (скриншот списка
  из четырёх операций → четыре верных `items`, ~1 с, 519/202 токенов). Без года на картинке модель
  придумала `2024` — происхождение даты надо запрашивать у модели явно и проверять.
- Сервер: `POST /transactions` требует `category_id`, `date`, `description` 2…200
  (`internal/application/handlers/types.go:173`) и идемпотентен по клиентскому `id`
  (`handlers/transactions.go:61`: повтор — `200` с существующей записью, создание — `201`); тип категории при создании **не сверяется**
  с типом операции (`internal/services/transaction_service.go:710` проверяет только существование) — **вынесено
  из этого плана** в `docs/backlog.md`: распознавание назначает категорию только по `(type, путь)`, а
  клиент фильтрует категории по типу (`TransactionEditViewModel.kt:118`), так что несоответствие
  здесь возникнуть не может. `ContextTimeout` 30 с на всё
  (`internal/application/http_server.go:110`), `SERVER_READ/WRITE_TIMEOUT` 15 с (`internal/config.go:85`),
  `BodyLimit` не стоит; `deploy/caddy/Caddyfile:12` режет тело на 1 МБ (dev/own-TLS). Уникальность
  категорий — `(family_id, name, type, parent_id)` (`migrations/001_consolidated.up.sql:138`): одно имя под
  разными родителями законно. Обёртки ответа проекта (`internal/observability/middleware.go:14`,
  `internal/metrics/echo.go:18`) writer не подменяют, `echo.Response` и gzip имеют `Unwrap` —
  `http.ResponseController` доходит до `*http.response`; но `ResponseRecorder` в handler-тестах
  (`handlers/auth_test.go:153`) дедлайны не поддерживает.
- Тесты: `testhelpers.SetupHTTPServer` собирает сервисы сам (`internal/testhelpers/integration_server.go:62`),
  опции применяются **после** сборки сервисов (`:114`); спека и роуты сверяются 1:1 в обе стороны
  (`tests/integration/openapi_coverage_test.go:140,156`). Перегенерация клиента не входит в
  `make -C android check` (`android/Makefile:37`), её свежесть проверяет CI (`android:api-check`, `.gitlab-ci.yml:264`).
- Android: `ApiClient` — `readTimeout` 30 с (`android/core/api/.../net/ApiClient.kt:17`); без
  повторов соединения только `httpWithoutRetries` через `createWithoutRetries` (`:43`, `:64`), обычный
  `transactions` (`ApiGraph.kt:36`) повторяет; `IOException` → `ApiFailure.Network` (`:85`);
  `unwrapWithCode` (`:70`) отличает `201` от `200` идемпотентного `POST`. OkHttp повторяет `503`
  только при `Retry-After: 0`. `okhttp` в `:core:api` — `implementation`, не `api`
  (`android/core/api/build.gradle.kts:55`): типы OkHttp наружу модуля не выходят. Манифест ловит только `LAUNCHER`; `AppGraph` переживает
  поворот, но не смерть процесса, после которой бутстрап уводит в `Loading → Home` (`MainActivity.kt:135`);
  `forms`-store чистится по булевому `onForm` (`:106`). Генератор 7.24.0 на минимальной multipart-спеке
  с `encoding: images: {style: form, explode: true}` даёт
  `suspend fun recognizeTransactions(@Part images: List<MultipartBody.Part>)` (проверено codex запуском).
- Ревью плана (plan-review, 16.09.2026): `unknown`/`GET` лишний при идемпотентном `POST`; строки без
  категории и с коротким описанием надо блокировать; `UploadTimeout` — параметр handler'а; валюта
  нормализуется; `MultipartBody.Part` остаётся в `:core:api`; сверка типа категории — отдельно.
- Codex, тред `ffs-recognize`, раунды 1–2 (`disagree` оба): опровергнутые посылки и риски внесены ниже
  (лимиты Caddy и тела, сквозные таймауты в две фазы, повторы платного вызова, дубль ≠ похожая,
  неоднозначные имена категорий, валюта строки, происхождение даты, атрибуция `source`, `onNewIntent`
  и повторная доставка intent, владение импортом и второй share, жизнь UUID строки, порядок задач).

## Development Approach

- **testing approach**: Regular — код, затем тесты в той же задаче.
- Сервер и клиент — один репозиторий; задачи закрываются по очереди: сервер (1–6), затем клиент
  (7–11). Каждая задача оставляет репозиторий зелёным: изменение спеки идёт в одном коммите с
  маршрутом **и** с `make -C android api-gen` (CI сверяет свежесть generated).
- Каждая серверная задача — `make fmt`, `make test`, `make lint` (0 issues); каждая клиентская —
  `make -C android check`.
- **CRITICAL: update this plan file when scope changes during implementation**
- Обратная совместимость: пустой `LLM_OLLAMA_HOST` — плечо выключено, маршрут отвечает `503`,
  остальной контракт не меняется.

## Testing Strategy

- **Сервер**: `internal/recognize` — чистые тесты на фикстурах реальных ответов модели
  (`testdata/*.json`, обезличенные; ограда ```` ```json ````, лишние поля, `year_present=false`,
  сумма с запятой, `source` вне диапазона, `items` не массив, мусор после JSON, 60 строк); handler —
  unit на `ResponseRecorder` (дедлайны там `http.ErrNotSupported` и терпятся; `principalContext`
  ставит `Content-Type: application/json` — multipart-тесты перебивают заголовок после вызова); сервис и маршрут — `SetupHTTPServer` с подменным движком
  (`testhelpers.WithRecognizer`): `200`/`503`/`502`/`413`/`422`, `Retry-After`, метрики; сетевые
  дедлайны — `httptest.NewUnstartedServer` с Echo проекта и сокращёнными сроками через параметры
  (секунды, не десятки).
- **Клиент**: Robolectric — `RecognizeViewModel` c `MockWebServer` (успех, `503` без второго запроса,
  обрыв после отправки → `failed` → повтор того же `POST` с тем же `draft` → `200` = уже сохранено,
  `422` → ошибка поля, частичный отказ пачки), реальный multipart-запрос с двумя файлами (имя `images`,
  `filename`, MIME, байты), экраны — `onNodeWithText`, `MainActivityTest` — share в `onCreate`,
  `onNewIntent`, пересоздание без повторной доставки.
- **Качество на реальных скриншотах** — не тестами: подкоманда `recognize` (задача 5) на скриншотах
  уведомлений Сбера, Т-Банка, Альфы и списков операций. **Порог приёмки** (проверяется руками, Post-Completion): суммы и типы
  верны ≥ 90 %; непустая дата верна ≥ 90 %, неверная непустая дата — не более 1 из 20 (лучше `null`,
  чем ошибка); категория верна или `null` — ошибочная категория ≤ 10 %.
- e2e с моделью в CI нет и не будет: плечо платное и внешнее.

## Progress Tracking

- `[x]` сразу по факту; ➕ — найденное по ходу; ⚠️ — блокеры.

## Solution Overview

**Контракт.** `POST /api/v1/transactions/recognize`, `multipart/form-data`, поле `images`
(1…5 частей, PNG или JPEG, ≤ 2 МиБ каждая; клиент сжимает сам), роль admin/member. Ответ `200`:

```
data: {
  items: [{ source: int | null, amount_minor, currency: string | null, type,
            date: date | null, date_assumed: bool, description,
            category_id: uuid | null, similar: [{ id, date, description }] }],
  incomplete: bool,
  model: "gemma4:31b"
}
```

`source` группирует строки под превью своей картинки, `incomplete` показывается заметкой «список мог
быть неполным», `similar` — до трёх ближайших операций для бейджа. Пустой `items` — успех. Отказы: `503 RECOGNITION_UNAVAILABLE` (плечо выключено, `needs_configuration`,
сеть, просрочка; `Retry-After` только если > 0, в секундах вверх), `502 RECOGNITION_FAILED` (модель
ответила, но ответ не разобрался — второго цикла нет), `413 PAYLOAD_TOO_LARGE` (тело больше
лимита), `422 VALIDATION_ERROR` с `field: images[i]` (не картинка, лишняя часть, больше пяти).
Вызов **не идемпотентен**: повтор — платный вызов, клиент никогда не повторяет его сам.

**Слои.** `internal/recognize` — чистый пакет: типы входа/выхода, промпт, разбор и нормализация
ответа; о `llm` не знает. Плечо — `internal/infrastructure/llmengine`: собирает `llm.Request`
(`Task: receipt.screenshot`, `Output.Mode: json`, `Temperature: 0`, по одному сообщению `user` на
картинку с текстом «Картинка N»), зовёт `llm.Client`, разбирает через `recognize`, классы отказа
`llm` переводит в `recognize.ErrUnavailable{RetryAfter}`. `services.RecognizeService` объявляет
узкий интерфейс `Recognizer{Recognize(ctx, recognize.Input) (recognize.Result, error);
Budget() time.Duration}`, подмешивает категории семьи, опорную дату в `family.Timezone`, валюту и
похожие операции. Handler тонкий: лимиты, multipart, дедлайны, коды. Переезд на `Router` меняет
только `llmengine` и сборку в `run.go`.

**Промпт и разбор.** Форма JSON диктуется системным промптом (проба показала, что схема через
`format` не действует): `{"items":[{source, amount, currency, type, date, date_text, year_present,
description, category}], "incomplete": bool}`; `amount` — десятичная строка, `date` — `YYYY-MM-DD`
или `null` (нет даты, «сегодня/вчера»), `date_text` — как на картинке, `year_present` — был ли год;
`category` — имя из переданного списка `тип: [родитель /] имя` или `null`. Разбор: снять ограду из
бэктиков, `json.Decoder` с `UseNumber`, лишние поля терпятся, после объекта — EOF, `items`
обязан быть массивом (`{}`/`null`/отсутствие → `ErrBadAnswer`). Нормализация: сумма — точный разбор
строки в `money.Minor` без `float64`, `0 < amount ≤ money.MaxAmount`; `date` — `date.Parse`; **год**:
`year_present=false` → год опорной даты, а если получившаяся дата позже опорной — предыдущий год, и
строка помечается `date_assumed=true`; `date > today` при `year_present=true` → `null`
(допуска «завтра» нет); `type` — enum; `description` — trim, до 200 (длиннее — усечение; короче 2 — остаётся, блокирует
сохранение на клиенте); `currency` — символ или слово в ISO-код (`₽`, `руб`, `RUB` → `RUB`; `$`, `€`,
`USD`, `EUR`, …), нераспознанное → `null` (валюта семьи);
`category` — единственное точное совпадение по `(type, путь)` среди активных категорий, иначе
`null`; `source` вне `[0, n)` → `null`; больше `MaxItems=50` — усечение и `incomplete`. Балансы, лимиты, комиссии, переводы между своими счетами и итоги промпт исключает явно.

**Похожие, не дубли.** Для каждого кандидата с датой — до трёх существующих операций с тем же
`amount_minor` и `type` и датой в `[date−1, date+1]` (`GetByFilter`), в `similar` с датой и описанием.
Клиент показывает бейдж «похоже на «шавуха», 14 сен» и **не снимает** галочку: две покупки по 300 ₽ в день — обычное дело. Правка
суммы, типа или даты в строке сбрасывает бейдж. Совпадения внутри пачки видны человеку списком.

**Дата и валюта.** `date: null` — строку нельзя сохранить, пока человек не выберет дату
(`DatePickerSheet`); `date_assumed` — дата показана как предположение и тоже требует нажатия
(подтвердить или сменить). `currency` не равна валюте семьи — строка заблокирована с пояснением
(`POST` валюту не принимает, конвертации нет); `null` — валюта семьи. Без категории или с описанием
короче двух символов строка тоже не сохраняется: `POST` их требует.

**Таймауты сквозные, две фазы.** `llm.Client`: `Timeout = LLM_TIMEOUT` (60 с, **валидация
≤ 60 с** — верхняя граница цепочки закреплена), `Attempts = 2`, `MaxRetryAfter = 10 с`;
`Budget()` = 130 с. Handler: `ContextTimeout` middleware пропускает маршрут (`Skipper`); до чтения
тела — `SetWriteDeadline(now + UploadTimeout + Budget() + 15 с)` и `SetReadDeadline(now +
UploadTimeout)` (`UploadTimeout` = 60 с), фаза загрузки под `context.WithTimeout(UploadTimeout)`;
после разбора тела — новый `context.WithTimeout(Budget() + 5 с)` на движок. `UploadTimeout` — поле handler'а (в тестах
короче). Дедлайны ставит `http.NewResponseController`; `http.ErrNotSupported` (тестовый
`ResponseRecorder`) терпится, любая другая ошибка — `500` в лог. Максимум цепочки: 60 + 130 + 15 = 205 с. Клиент:
отдельный `OkHttpClient` для распознавания — connect 10 с, write 60 с, read 210 с, без повторов
(`retryOnConnectionFailure(false)`; повтор `503` OkHttp делает только при `Retry-After: 0`, чего
сервер не шлёт — это и есть «без повторного `503`»), собранный через `newBuilder()` общих
клиента и Retrofit, чтобы не потерять `TokenInterceptor`, обработку истечения сессии и сериализаторы.
Тело: Echo `BodyLimit("11M")` на маршруте (11 000 000 байт: 5 × 2 МиБ + 514 240 на оболочку — хватает),
каждая часть читается не дальше `MaxImageBytes+1`, ошибка лимитера из-под multipart-парсера
распознаётся `errors.As` и остаётся `413`; Caddy dev-раскладки: `@recognize path
/api/v1/transactions/recognize` → `request_body max_size 11MB`, остальным 1 МБ — правило 1 МБ должно
этот путь **исключать**, а не дополнять.

**Клиент.** Владение импортом — `AppGraph.imports: ImportStore`: `offer(uris): UUID` кладёт
`PendingImport(id, uris)` и публикует `pending` в `StateFlow`; `claim(id): PendingImport?` отдаёт
его ровно одному потребителю (атомарно, второй получает `null`). `MainActivity` зовёт `offer` из
`onCreate` **только при `savedInstanceState == null`** и из `onNewIntent` всегда
(`launchMode="singleTop"`, `SEND` и `SEND_MULTIPLE`, `EXTRA_STREAM` и `ClipData`), после чего
`intent.removeExtra(EXTRA_STREAM)` — пересозданная активити тот же intent не предлагает.
`AppRoot` только маршрутизирует: при сессии и `pending` — `screen = Recognize(id)`; бутстрап после
успеха переходит в `Recognize(id)`, если `pending` есть, иначе в `Home`; `onForm` включает
`Recognize`, иначе `forms`-store чистится. `RecognizeViewModel` (ключ `importId`) в `init` делает
`claim` и **сразу** читает URI (права действуют, пока жива задача) в JPEG в
`cacheDir/import/<importId>/`, затем один запрос; при повороте ни `offer`, ни запрос не повторяются.
Второй share поверх открытого `Recognize`: в `Preparing/Recognizing/Saving` — удерживается в `ImportStore` и открывается после завершения (пользователю показывается «после
сохранения»); в `Review/Failure` — заменяет текущий: старая модель удаляется из store по ключу, её
файлы чистятся. Смерть процесса теряет незавершённый импорт и статусы строк — гарантия «без дублей»
действует в живом процессе, что записано в приёмке. Каждая строка получает `draft` UUID в модели;
«Сохранить N» шлёт выбранные по очереди; статусы `pending | saving | saved | failed(error)`; `422` — ошибка поля строки; любой другой
отказ, включая неопределённый исход после отправки, — `failed` с «Повторить»: повтор шлёт **тот же**
`POST` с тем же `draft`, `201` — создано, `200` — уже было сохранено, и строка принимает поля из
ответа сервера (правки после обрыва не теряются молча — человек видит, что записалось);
`saved` заперты от правки и повторной отправки. Результат — наблюдаемое `savedCount` в состоянии
модели, `AppRoot` по нему взводит `listStale`, `homeStale`, `budgetsStale` (без колбэка, переживающего
композицию). Точки входа: share-intent; на «Операциях» в шапке иконка сканирования → лист
«Из галереи» (`PickMultipleVisualMedia(5)`) / «Снять» (`TakePicture` в `cacheDir` через `FileProvider`,
URI снимка — в `rememberSaveable`, разрешение `CAMERA` не объявляется).

## Technical Details

Сервер:

```
internal/recognize/
  recognize.go   Input{Images []Image{MIME, Data}, Categories []Category{ID, Type, Path}, Today date.Date, Currency string}
                 Item{Source *int, AmountMinor money.Minor, Currency *string, Type transaction.Type,
                      Date *date.Date, DateAssumed bool, Description string, CategoryID *uuid.UUID, Similar []Similar{ID, Date, Description}}
                 Result{Items, Incomplete bool, Model}
                 const MaxImages = 5, MaxItems = 50, MaxImageBytes = 2 << 20, MaxPixels = 16e6
                 ErrUnavailable{RetryAfter time.Duration}, ErrBadAnswer, ErrDisabled
  image.go       CheckImage(data []byte) (mime string, err error)   // MIME по сигнатуре, image.DecodeConfig png/jpeg — размеры, не целостность
  prompt.go      System(input) string; UserText(i int) string        // форма JSON, правила, список категорий «expense: Еда / Кафе»
  parse.go       Parse(text string) (raw, error)                     // ограда, UseNumber, EOF после объекта, items — массив
  normalize.go   Normalize(raw, input) (Result, error)               // суммы, валюта, даты и год, категории, source, усечение

internal/infrastructure/llmengine/
  ollama.go      New(host, model string, timeout time.Duration, obs llm.Observer) *Engine
                 (*Engine) Recognize(ctx, recognize.Input) (recognize.Result, error); Budget()

internal/services/recognize_service.go
  Recognizer interface; RecognizeService{Recognize(ctx, images) (recognize.Result, error); Budget()}
  RecognizeObserver{Observe(outcome string, d time.Duration, items int)} + NopRecognizeObserver

internal/application/handlers/recognize.go   RecognizeHandler{service, uploadTimeout}; дедлайны через http.NewResponseController
internal/application/handlers/errors.go      ErrCode RECOGNITION_UNAVAILABLE, RECOGNITION_FAILED, PAYLOAD_TOO_LARGE
internal/application/http_server.go          transactions.POST("/recognize", ..., middleware.BodyLimit("11M")); Skipper у ContextTimeout
internal/config.go                            LLM_OLLAMA_HOST (пусто — выключено), LLM_MODEL (gemma4:31b-cloud), LLM_TIMEOUT (60s, 0 < t ≤ 60s)
internal/metrics                              ffs_recognitions_total{outcome=ok|empty|failed|unavailable}, ffs_recognition_duration_seconds
cmd/server/recognize.go                       go run ./cmd/server recognize <file>... — по образцу cmd/server/backup.go, печатает Result как JSON
```

Клиент:

```
android/core/api/generated/...          recognizeTransactions(images: List<MultipartBody.Part>) — из спеки, руками не править
android/core/api/.../ApiGraph.kt        suspend fun recognize(files: List<File>): RecognizeOk — части MultipartBody собираются здесь,
                                        OkHttp наружу модуля не выходит (okhttp — implementation); долгий клиент внутри
android/app/.../AppScreen.kt            data class Recognize(val importId: UUID)
android/app/.../ImportStore.kt          offer/claim/pending, политика второго share
android/app/.../MainActivity.kt         onCreate(savedInstanceState == null)/onNewIntent → graph.imports.offer; AppRoot: маршрут по pending
android/app/.../ui/recognize/ImportFiles.kt          URI → JPEG (android.media.ExifInterface, ≤ 2048, q85 → ниже, пока > 2 МиБ; иначе отказ строки), sweep
android/app/.../ui/recognize/RecognizeViewModel.kt   claim, состояния, строки, save loop с повтором того же POST, savedCount
android/app/.../ui/recognize/RecognizeScreen.kt      превью, прогресс, список строк, «Сохранить N», ошибки
android/app/.../ui/recognize/ImportSourceSheet.kt    «Из галереи» / «Снять»
android/app/src/main/AndroidManifest.xml  intent-filter SEND/SEND_MULTIPLE image/*, singleTop, FileProvider
android/app/src/main/res/xml/file_paths.xml
android/app/.../ui/AppIcons.kt            ScanLine (одна иконка; лист — текстовые пункты)
```

## Implementation Steps

### Task 1: Чистый пакет `internal/recognize`

**Files:**
- Create: `internal/recognize/{recognize,image,prompt,parse,normalize}.go` и `_test.go`
- Create: `internal/recognize/testdata/*.json` (ответы модели из проб, обезличенные)

- [ ] типы `Input`/`Result`/`Item` (`Source *int`, `Currency *string`, `DateAssumed`, `Similar`, `Incomplete`), константы лимитов, ошибки
- [ ] `CheckImage`: MIME по сигнатуре (PNG/JPEG, иначе отказ), `image.DecodeConfig`, `MaxPixels`, `MaxImageBytes`; целостность не обещается
- [ ] `System(input)`: форма JSON с `date_text`/`year_present`, правила (десятичная строка, `null` для «сегодня/вчера» и без даты, исключения балансов/итогов/переводов между своими счетами), список категорий с путями; `UserText(i)`
- [ ] `Parse`: ограда, `UseNumber`, лишние поля терпятся, EOF после объекта, `items` — массив
- [ ] `Normalize`: сумма без `float64`, диапазон, валюта → ISO или `nil`, дата и правило года с `DateAssumed`, `null` для будущей даты, единственное совпадение категории по `(type, path)`, `source` → `nil`, усечение до `MaxItems` с `Incomplete`
- [ ] тесты на фикстурах: скриншот из проб (четыре строки), ограда, `year_present=false` в декабре/январе (обе стороны опорной даты), запятая в сумме, `₽`/`руб.` → `RUB` и незнакомая валюта → `nil`, неоднозначная категория → `null`, `items: {}` и мусор после JSON → `ErrBadAnswer`, 60 строк
- [ ] тесты `CheckImage` (PNG, JPEG, GIF, 20 Мп, обрезанный заголовок)
- [ ] `make test`/`make lint` — зелёные до задачи 2

### Task 2: Плечо `llmengine`, `RecognizeService`, сборка сервисов и стенда

**Files:**
- Create: `internal/infrastructure/llmengine/ollama.go`, `ollama_test.go` (httptest-Ollama)
- Create: `internal/services/recognize_service.go`, `recognize_service_test.go`
- Modify: `internal/services/container.go`, `internal/run.go` (движок `nil`), `internal/testhelpers/integration_server.go` (`ServerOption` над параметрами стенда, применяется **до** сборки сервисов; `WithTrustedProxies` сохраняется; `WithRecognizer`)
- Modify: `go.mod` (`github.com/lllypuk/llm v0.2.0`)

- [ ] `go get github.com/lllypuk/llm@v0.2.0`; `Engine` собирает `llm.Request` (одно `user`-сообщение на картинку), `Attempts=2`, `MaxRetryAfter=10s`, `Temperature=0`, `Task=receipt.screenshot`
- [ ] отказы: `needs_configuration`/`immediate`/`after_delay`/просрочка → `ErrUnavailable{RetryAfter}`; `ErrBadAnswer` — без повторного `Chat`
- [ ] один `slog` на вызов по `Report`: outcome, попытки, usage, latency, число items; байты картинок и текст ответа в лог не попадают
- [ ] `RecognizeService`: `nil`-движок → `ErrDisabled`; активные категории с путями; `Today(family.Location())`, валюта семьи; `similar` (id, дата, описание) через `GetByFilter` (сумма точно, тип, дата ±1, `Limit 3`) только у строк с датой
- [ ] `RecognizeObserver` (+ `Nop`) — исход считается **после** нормализации
- [ ] `NewServices` получает `Recognizer` и наблюдателя; оба вызова (`run.go:122`, стенд) обновлены в этой задаче, движок пока `nil`
- [ ] тесты движка на `httptest` (ограда в ответе, 429 с `Retry-After`, 401, обрыв, отмена контекста); тесты сервиса с подменным движком (категории, `similar`, выключенное плечо)
- [ ] `make test`/`make lint` — зелёные до задачи 3

### Task 3: Handler, маршрут, лимиты, дедлайны, спека

**Files:**
- Create: `internal/application/handlers/recognize.go`, `recognize_test.go`
- Modify: `internal/application/http_server.go`, `internal/application/error_handler.go`, `handlers/errors.go` (коды), `handlers/helpers.go` (маппинг `413`)
- Modify: `docs/api/openapi.yaml` (в этой же задаче — иначе тест покрытия красный); `make -C android api-gen`
- Create: `tests/integration/recognize_test.go`

- [ ] `transactions.POST("/recognize", h.Recognize, middleware.BodyLimit("11M"))`; `Skipper` у `ContextTimeout` для этого пути
- [ ] дедлайны через `http.NewResponseController`: write `now+UploadTimeout+Budget()+15s` и read `now+UploadTimeout` до тела; `http.ErrNotSupported` терпится, иная ошибка → `500` с логом; `UploadTimeout` — поле handler'а (60 с)
- [ ] две фазы контекста: загрузка под `UploadTimeout`, движок под `Budget()+5s`
- [ ] потоковый `MultipartReader`: только части `images`, ≤ 5, каждая читается до `MaxImageBytes+1`, `CheckImage`; отказ — `422` с `field: images[i]`; ошибка лимитера через `errors.As` → `413`
- [ ] коды: `ErrDisabled`/`ErrUnavailable` → `503 RECOGNITION_UNAVAILABLE` (+ `Retry-After` ceil, только > 0); `ErrBadAnswer` → `502 RECOGNITION_FAILED`; `413` → конверт `PAYLOAD_TOO_LARGE` в error handler
- [ ] спека: `recognizeTransactions`, `requestBody` multipart с `encoding: images: {style: form, explode: true}`, схемы `RecognizeOk` (`incomplete`, `model`), `RecognizedTransaction` (`source`/`currency`/`date` nullable, `date_assumed`, `similar[{id, date, description}]`), ответы `413`, `422`, `502`, `503`; `npx @redocly/cli lint`; перегенерация клиента, сигнатура `List<MultipartBody.Part>` проверена
- [ ] unit-тесты handler'а с `principalContext` (заголовок `Content-Type` перебивается на multipart после вызова): успех, каждая ветка отказа, шестая часть, чужое поле
- [ ] интеграционные тесты: `200` с `similar`, `503` без движка, `503`+`Retry-After`, `502`, `413`, `422`, роль `member` ок, без токена `401`
- [ ] `make fmt && make test && make lint`; `make -C android check` — зелёные до задачи 4

### Task 4: Конфиг, движок в сборке, метрики, тест дедлайнов

**Files:**
- Modify: `internal/config.go`, `config_test.go`; `internal/run.go`; `internal/metrics/*.go` и тесты
- Create: `tests/integration/recognize_deadlines_test.go`

- [ ] `LLM_OLLAMA_HOST` (URL, пусто — выключено), `LLM_MODEL`, `LLM_TIMEOUT` (`0 < t ≤ 60s`); `Validate`
- [ ] `run.go`: движок только при заданном хосте; `RecognizeObserver` → `ffs_recognitions_total{outcome}`, `ffs_recognition_duration_seconds`; без реестра — `Nop` (`llm.Observer` в этом выпуске не подключается)
- [ ] метрики в интеграционных тестах через `ts.Metrics.Handler()`
- [ ] тест дедлайнов: `httptest.NewUnstartedServer` с Echo проекта, `Config.WriteTimeout`/`ReadTimeout` в 1–2 с до `Start`, сокращённые `UploadTimeout` (поле handler'а) и бюджет (подменный движок) через параметры стенда; проверить: поздний ответ движка доходит; медленная загрузка дольше `UploadTimeout` → отказ; отмена контекста доходит до движка
- [ ] `make fmt && make test && make lint` — зелёные до задачи 5

### Task 5: Подкоманда `recognize` для реальных скриншотов

**Files:**
- Create: `cmd/server/recognize.go` (по образцу `cmd/server/backup.go`: `internal.OpenDatabaseNoMigrate`)
- Modify: `cmd/server/main.go`

- [ ] `go run ./cmd/server recognize a.png b.jpg`: конфиг из env, категории из БД (`OpenDatabaseNoMigrate`), печать `Result` JSON и отчёта вызова в stderr
- [ ] без `LLM_OLLAMA_HOST` — понятный отказ, код выхода 2
- [ ] тест отказа без хоста
- [ ] `make test`/`make lint` — зелёные до задачи 6

### Task 6: Деплой и документация сервера

**Files:**
- Modify: `deploy/docker-compose.yml`, `deploy/docker-compose.proxied.yml`, `deploy/.env.example`, `deploy/README.md`, `deploy/caddy/Caddyfile`, `docker/docker-compose.yml`
- Modify: `CLAUDE.md`, `docs/api/README.md`

- [ ] `LLM_OLLAMA_HOST`, `LLM_MODEL`, `LLM_TIMEOUT` в `environment:` обеих раскладок, пустые по умолчанию; `make compose-config`
- [ ] Caddyfile dev-раскладки: `@recognize path /api/v1/transactions/recognize` → `request_body max_size 11MB`, остальным 1 МБ с исключением этого пути; `make caddy-validate`
- [ ] `deploy/README.md`: плечо, адрес демона на хосте, отсутствие секретов, что значит `503`, **обязательная правка vhost в репозитории лендинга** для прода
- [ ] `CLAUDE.md`: раздел «Распознавание скриншотов» — слои, коды, две фазы дедлайнов, «второго цикла повторов нет», подкоманда
- [ ] `docs/api/README.md`: multipart и `encoding` как правило для файловых операций
- [ ] `make fmt && make test && make lint`; `make compose-config`; `make caddy-validate` — зелёные до задачи 7

### Task 7: Долгий транспорт клиента и `ApiGraph.recognize`

**Files:**
- Modify: `android/core/api/.../ApiGraph.kt`, `net/ApiClient.kt`, `ApiClientTest.kt`, `ApiGraphTest.kt`

- [ ] `ApiClient.createLongCall`: `http.newBuilder()` (connect 10 с, write 60 с, read 210 с, `retryOnConnectionFailure(false)`) и `retrofit.newBuilder()` — интерсепторы, истечение сессии и сериализаторы общие; обычный клиент со старыми сроками
- [ ] `ApiGraph.recognize(files: List<File>): RecognizeOk` собирает части `images` (`filename`, `image/jpeg`) внутри `:core:api`; `MultipartBody.Part` в `:app` не выходит (`okhttp` остаётся `implementation`)
- [ ] тест: `MockWebServer` с задержкой 3 с и обычным клиентом с укороченным read → `ApiFailure.Network` с причиной `SocketTimeoutException`; долгий клиент отвечает; `503` — ровно один запрос по счётчику; обрыв — ровно один
- [ ] тест `ApiGraph.recognize` с двумя файлами: части `images`, `filename`, MIME, байты — по `RecordedRequest`
- [ ] `make -C android check` — зелёный до задачи 8

### Task 8: Импорт картинок на клиенте

**Files:**
- Create: `android/app/.../ui/recognize/ImportFiles.kt`, тест
- Modify: `android/app/src/main/AndroidManifest.xml`, `res/xml/file_paths.xml` (create), `FamilyFinancesApp.kt`

- [ ] `ImportFiles.prepare(context, importId, uris)`: чтение через `ContentResolver`, ориентация по `android.media.ExifInterface` (без новой зависимости, `minSdk 26`), `inSampleSize`, длинная сторона ≤ 2048, JPEG q85 с понижением качества, пока файл > 2 МиБ (ниже q60 — отказ строки), ≤ 5 файлов, отказ на не-картинке
- [ ] `ImportFiles.sweep(context)` на старте приложения; удаление каталога импорта по завершении
- [ ] `FileProvider` для камеры: `authorities="${applicationId}.files"`, `cache-path import/`
- [ ] тесты Robolectric: PNG 1080×2400 → JPEG ≤ 2048 и ≤ 2 МиБ, EXIF-поворот, шестой URI отбрасывается, мусорный URI → ошибка строки, sweep чистит
- [ ] `make -C android check` — зелёный до задачи 9

### Task 9: `ImportStore` и `RecognizeViewModel`

**Files:**
- Create: `android/app/.../ImportStore.kt`, `ImportStoreTest.kt`
- Create: `android/app/.../ui/recognize/RecognizeViewModel.kt`, `RecognizeViewModelTest.kt`
- Modify: `android/app/.../AppGraph.kt`

- [ ] `ImportStore`: `offer(uris): UUID`, `claim(id)` — ровно одному, `pending: StateFlow<UUID?>`, `hold`/`release` для второго share во время `Preparing/Recognizing/Saving`
- [ ] состояния модели: `Preparing → Recognizing → Review(rows) | Failure(error, retryable)`; строка: `draft`, поля, `included`, `similarTo`, `dateAssumed`, `currencyMismatch`, `status`
- [ ] `claim` и один запрос в `init`; при рекомпозиции и повороте повторов нет; «Повторить» — только по нажатию
- [ ] правки строки; смена суммы/типа/даты сбрасывает `similar` и `dateAssumed` (после явного выбора даты); `date == null`, `dateAssumed`, `currencyMismatch`, `categoryId == null` или описание короче 2 — сохранить нельзя
- [ ] `save()`: выбранные строки по очереди `POST` с `draft` через `unwrapWithCode`; `201` → `saved`; `200` → `saved` с полями из ответа (запись уже была); `422` → ошибка поля; иной отказ → `failed` с «Повторить» тем же `draft`; `saved` заперты; `savedCount` в состоянии
- [ ] `onCleared` чистит файлы импорта
- [ ] тесты: `ImportStore` (двойной `claim`, `hold`), модель с `MockWebServer` (успех, `503` — один запрос, обрыв после отправки → повтор тем же `draft` → `200` принимает поля сервера, `422`, частичный отказ пачки, повтор не шлёт `saved`)
- [ ] `make -C android check` — зелёный до задачи 10

### Task 10: Экран, точки входа, share-intent

**Files:**
- Create: `android/app/.../ui/recognize/RecognizeScreen.kt`, `ImportSourceSheet.kt`, тесты
- Modify: `AppScreen.kt`, `AppScreenSaver`, `MainActivity.kt`, `MainActivityTest.kt`, `ui/transactions/TransactionsScreen.kt`, `ui/AppIcons.kt`, `res/values/strings.xml`

- [ ] `RecognizeScreen`: «Распознаём…», строки сгруппированы под превью своей картинки (`source`; без `source` — группа «не привязано»), заметка при `incomplete`; строка: чекбокс, сумма со знаком по типу, дата / «выбрать дату» / «год подставлен — проверьте», категория через `CategorySheet` («выбрать категорию» при `null`), описание; бейдж «похоже на «…», дата»; блокировка по валюте; «Сохранить N»; ошибки строк и «Повторить»; «Повторить» при `Failure`; `BackHandler` — уход заблокирован в `Saving`
- [ ] `AppScreen.Recognize(importId)` + Saver; `MainActivity`: `offer` из `onCreate` при `savedInstanceState == null` и из `onNewIntent`, затем `removeExtra`; `launchMode="singleTop"`; `onForm` включает `Recognize`
- [ ] `AppRoot`: маршрут по `imports.pending` при сессии; бутстрап → `Recognize`, если `pending` есть, иначе `Home`; второй share по политике `ImportStore`; `savedCount` → три флага
- [ ] «Операции»: иконка `ScanLine` в шапке → `ImportSourceSheet`: `PickMultipleVisualMedia(5)` и `TakePicture` (URI снимка в `rememberSaveable`); результат → `imports.offer`
- [ ] строки: заголовки, кнопки, бейджи, пояснение валюты, пустой результат («Операций на картинке не нашлось»), `503` («Распознавание на сервере не настроено»)
- [ ] экранные тесты: группы по картинкам, «Сохранить N» считает выбранные, `date == null`, `dateAssumed` и `categoryId == null` блокируют, бейдж, заметка `incomplete`; `MainActivityTest`: `SEND` до логина → `Recognize` после бутстрапа; `onNewIntent` при открытом приложении; пересоздание не предлагает intent повторно
- [ ] `make -C android check` — зелёный до задачи 11

### Task 11: Документация клиента

**Files:**
- Modify: `android/CLAUDE.md`, `android/gradle/libs.versions.toml` (версия приложения)

- [ ] `android/CLAUDE.md`: `ImportStore` и владение импортом, временные файлы и sweep, `draft` строк и повтор тем же `POST`, второй share, «клиент никогда не повторяет распознавание сам», потеря импорта при смерти процесса
- [ ] `make -C android check` — зелёный

### Task 12: Verify acceptance criteria
- [ ] распознавание уведомления и списка даёт кандидатов с суммой, типом, датой и категорией; пустой результат — не ошибка
- [ ] строка без даты, с подставленным годом, без категории, с коротким описанием или в чужой валюте не сохраняется, пока не исправлена; похожая операция помечена, но не снята
- [ ] в живом процессе обрыв после отправки не создаёт дубль: повтор тем же `draft` отвечает `200`; после смерти процесса гарантии нет — записано
- [ ] `LLM_OLLAMA_HOST` пуст → `503`, остальной контракт как раньше; секретов в конфиге нет
- [ ] `make fmt && make test && make lint` (0 issues); `make -C android check`; `make compose-config`; `make caddy-validate`

### Task 13: [Final] Update documentation
- [ ] `CLAUDE.md` (корень и `android/`) сверены с кодом
- [ ] `docs/backlog.md`: переезд на `Router` после v0.4.0; `ffs_llm_attempts_total` из `llm.Observer`; WebP/HEIC; журнал `draft`-UUID для восстановления после смерти процесса — если понадобится

## Post-Completion

*Пункты для человека и внешних систем — без чекбоксов; ralphex закрывает план по задачам выше, релиз
считается готовым только после этого раздела.*

**Ручная проверка**
- Релиз на A204SO (память `android-device-testing`): share из галереи и из уведомления банка, пикер,
  камера, поворот на экране проверки, второй share, `503` при выключенном плече.
- Порог качества из Testing Strategy на наборе реальных скриншотов уведомлений Сбера, Т-Банка, Альфы
  и списков операций через `go run ./cmd/server recognize`; протокол — в `docs/specs/`.
- Латентность и расход `gemma4:31b-cloud` на пяти картинках за вызов; если хуже 30 с — один вызов
  на картинку с ограничением конкурентности (отдельное решение).

**Внешние системы**
- **Прод-ingress — условие готовности релиза:** MR в репозиторий лендинга (`deploy/Caddyfile.prod`) с
  `request_body` для `/api/v1/transactions/recognize`, ручной `deploy:prod`, затем загрузка пяти
  картинок через `ffs.shatrov.tech`; без этого прод режет тело на 1 МБ.
- `~/ffs/.env` на mini: `LLM_OLLAMA_HOST=http://192.168.1.10:11434`, `LLM_MODEL=gemma4:31b-cloud`;
  демон слушает `*:11434`, из контейнера адрес хоста достижим.
- Репозиторий `observability`: согласовать имена `ffs_recognitions_total`,
  `ffs_recognition_duration_seconds`; панель плеча — потом.
- Отдельным заходом (вне этого плана): сверка типа категории с типом операции на `POST`/`PUT
  /transactions` — `422 VALIDATION_ERROR` `field: category_id`, проверка итоговой пары после слияния
  полей; старая несогласованная запись получит `422` и при правке описания (находка codex).
- После `llm` v0.4.0: `llmengine` на `Router`/`llmconfig`, идентичность вызова, платное плечо —
  отдельный план; `internal/recognize` не меняется.
- Persistable-права PhotoPicker и восстановление импорта после смерти процесса — если понадобится.

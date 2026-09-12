# План 09 — Серверные находки планов 07–08

Закрывает два раздела [бэклога](../backlog.md) «Сервер: находки при планировании …» (11.09.2026).
Все изменения контракта собраны в один план, чтобы Android-клиент перегенерировать один раз.
Ревью: plan-review и codex (тред `plan09-server-findings`), 12.09.2026.

## Overview

- Бюджеты: `is_active` уходит из `PUT /budgets/:id`, удалённый бюджет перестаёт читаться по id;
  бизнес-отказы получают `409` со своими кодами; пересечение периодов проверяется включительно;
  `PUT` сверяет сумму с расходом по новым датам.
- Пользователи: пароль, деактивация и отзыв сессий — одна транзакция в репозитории;
  `PATCH /users/:id` применяет роль и активность одной записью; `409 EMAIL_TAKEN` описан у `updateUser`.
- Бэкап: длительность `VACUUM INTO` пишется в лог — сначала измерение, таймауты не трогаем.
- Клиент: перегенерация из спеки и перевод новых кодов `409` бюджетов.

## Context (from discovery)

- `internal/services/budget_service.go`: `UpdateBudget` (`:218`) пересчитывает расход по старым датам
  (`recalculateAndUpdateSpent`, сбой только логируется — `:234`), затем `applyBudgetUpdate` (`:262`)
  сверяет новую сумму с этим расходом; `budgetPeriodsOverlap` (`:581`, сравнение `:592`) строгий;
  проверка пересечения только при смене дат (`:246`); `validateBudgetPeriodForUpdate` (`:610`);
  ошибки `ErrBudgetOverlapExists`, `ErrBudgetAlreadyExceeded`, `ErrBudgetNameExists`,
  `ErrBudgetCalculationFailed` (`:19–31`).
- `internal/application/handlers/budgets.go:349` `handleBudgetServiceError` — все бизнес-отказы одной
  веткой `422` c `bodyDetail`. Коды — `handlers/errors.go` (`ErrCodeEmailTaken` и соседи, `:33–40`).
- `UpdateBudgetDTO.IsActive` (`internal/services/dto/budget_dto.go:40`), `UpdateBudgetRequest.is_active`
  (`docs/api/openapi.yaml:1489`). Категории в `UpdateBudgetDTO` нет. Репозиторий бюджетов
  (`internal/infrastructure/budget/budget_repository_sqlite.go`): списки фильтруют `is_active = 1`,
  `Delete` (`:587`) — мягкий, **но `GetByID` (`:175`) и `Update` (`:410`) фильтра не имеют** — удалённый
  бюджет читается и правится по известному id. `UNIQUE(family_id, name, start_date, end_date)`
  (`migrations/001_consolidated.up.sql:112`) считает и удалённые строки — вне плана, в бэклог.
  `BudgetFilterDTO.IsActive` (`:48`) и `active_only` остаются.
- `internal/infrastructure/user/user_repository_sqlite.go`: `updateGuarded` (`:282`) — одна транзакция с
  проверкой последнего админа **до** записи (`isLastActiveAdmin`, `:324`: строка — активный админ и
  других активных нет); `UpdateRole` (`:267`), `SetActive` (`:274`), `UpdatePassword` (`:340`, без
  транзакции). Таблица `sessions` в той же БД, `sessions.user_id` с `ON DELETE CASCADE`
  (`001_consolidated.up.sql:138`), но `UPDATE users` каскад не запускает. `DeleteByUser` —
  `internal/infrastructure/auth/session_repository_sqlite.go:153`, интерфейс `auth.SessionRepository`
  (`internal/auth/session.go:68`), фейк — `internal/auth/service_test.go:108`.
- `internal/auth/service.go`: `UserLookup` (`:27`), `setPassword` (`:204`) пишет хеш, потом
  `sessions.DeleteByUser`; `RevokeAllSessions` (`:150`) — реализация `services.SessionRevoker`
  (`user_service.go:51`), вызывается из `SetActive` (`:227`), `tests/integration/api_auth_test.go:102`
  и `internal/auth/service_test.go:313`. `NewUserService(userRepo, familyRepo, authService)` —
  `internal/services/container.go:52`. `UserRepository` — `user_service.go:36`; `UserService`
  (`SetActive`, `ChangeUserRole`) — `internal/services/interfaces.go:20–33`.
  `user_repository_test.go:403` зовёт `UpdateRole`/`SetActive`.
- `handlers/users.go:177` `PatchUser`: `ChangeUserRole`, затем `SetActive` (`:204–213`), без отката;
  роут под `adminOnly` (`http_server.go:190`), самодеактивация — `ErrCannotDeactivateSelf` до
  репозитория (`user_service.go:210`, тест `api_users_deactivate_test.go:94`). Моки `UserService`:
  `handlers/users_test.go`, `internal/application/http_server_test.go:79,84`;
  `tests/integration/session_revalidation_test.go:32` зовёт `Services.User.SetActive` напрямую.
- Спека: `updateUser` (`openapi.yaml:305`) без `'409'`; `updateCurrentUser` его уже имеет.
  **`patchUser` (`:325–330`) документирует последовательное применение** — «изменение роли уже
  сохранено»; текст попадает в `UsersApi.kt:74`. Список кодов в шапке (`:26`) и описание `Conflict`
  (`:990`). Интеграционного теста на `PUT /users/:id` с чужим email нет (только `/me`, `me_test.go:115`).
- `handlers/backups.go:23` `CreateBackup` — синхронный; серверные таймауты: `SERVER_WRITE_TIMEOUT` 15 с
  (`internal/config.go:20`) и `middleware.ContextTimeout` 30 с на всех роутах (`http_server.go:102`).
  **Android ждёт ответ 30 с** (`android/core/api/.../net/ApiClient.kt:17`, `readTimeout`), клиент
  бэкапов — без повторов (`ApiGraph.kt:41`). Продлевать серверную сторону бессмысленно, пока не
  измерено, сколько идёт `VACUUM INTO`; пул — одно соединение (`internal/infrastructure/sqlite.go:46`),
  на время бэкапа остальные запросы ждут.
- Клиент: `ui/budgets/BudgetEditViewModel.kt:303` — отказ с `field: "body"` показывается одним текстом
  `budget_error_rejected` (`strings.xml:91`); карта кодов `409` — образец
  `ui/settings/SettingsConflicts.kt`. Android `UpdateBudgetRequest` строится без `isActive`.
  `android/core/api/generated` — только `make -C android api-gen`.
- Тесты: `internal/services/budget_service_test.go`, `user_service_test.go`, `helpers_test.go` (фейки),
  `internal/auth/service_test.go`, `internal/infrastructure/user/user_repository_test.go`,
  `internal/infrastructure/budget/*_test.go`,
  `tests/integration/{budgets,users,api_users_deactivate,backups}_test.go`,
  `openapi_coverage_test.go` (спека и роуты совпадают точно).

## Development Approach

- **testing approach**: Regular (код, затем тесты) — как в планах 07–08.
- Каждая задача закрывается полностью, дерево компилируется и `make fmt && make test && make lint`
  (0 issues) зелёные до следующей задачи.
- Каждая задача с изменением кода включает новые или обновлённые тесты на успех и отказ.
- Обратная совместимость контракта не требуется: клиент в этом же репозитории и перегенерируется
  в задаче 9. Единственный внешний потребитель — он.
- Отклонение от плана — сразу в этот файл.

## Testing Strategy

- **unit**: сервис бюджетов и пользователей (фейки из `helpers_test.go`), `auth.Service`, репозитории
  на in-memory SQLite. Атомарность транзакции пользователей проверяется SQLite-триггером, запрещающим
  `DELETE` из `sessions`: после ошибки хеш, роль, активность и сессии на месте.
- **integration**: `tests/integration/*` через `testhelpers.SetupHTTPServer` — коды и статусы на проводе,
  `openapi_coverage_test.go` держит спеку и роуты в согласии.
- **Android**: `make -C android api-check` (генерация совпадает с закоммиченной), `make -C android check`.
- e2e-тестов у проекта нет.

## Progress Tracking

- `[x]` сразу по завершении; ➕ — найденное по ходу; ⚠️ — блокер.

## Solution Overview

- **`is_active` бюджета убирается из запроса, а не разводится в схеме.** Клиент флаг не показывает,
  «выключить, но не удалить» никому не нужен; колонка остаётся маркером мягкого удаления,
  `active_only` — фильтром по датам. Чтобы семантика была полной, `GetByID` получает `is_active = 1`:
  удалённый бюджет — `404` и на `GET`, и на `PUT`, и на `DELETE`. Бюджет, выключенный через `PUT` до
  этого плана, включить обратно будет нельзя — на проде таких нет (клиент поле не шлёт). Вторую
  половину пункта бэклога (возвращать выключенные при `active_only=false`) сознательно не делаем.
- **Бизнес-отказы бюджета — `409` с кодами**, как `CURRENCY_LOCKED`: `BUDGET_OVERLAP`,
  `BUDGET_NAME_EXISTS`, `BUDGET_BELOW_SPENT`. Отказы формы (`amount_minor` слишком большой, даты в
  обратном порядке) остаются `422` — это валидация поля.
- **Пересечение включительное**: расход считается по `date >= start AND date <= end`
  (`transaction_repository_sqlite.go:781`), значит общий день — двойной учёт. Клиентские пресеты
  делают конец = начало + N − 1, они не пострадают. Пара с общим днём, созданная раньше, остаётся в
  базе: переименование проходит, смена дат одного из них отбивается `409`, пока границу не сдвинут
  (обход — сдвинуть один конец на день). Регрессионный тест на такую пару обязателен.
- **`PUT` бюджета**: сначала применяются даты, затем расход считается по новому периоду, и только потом
  сумма сверяется с ним. Проверка суммы — только когда `amount_minor` в запросе (сегодняшняя семантика);
  расширение периода без суммы допустимо, бюджет просто станет перерасходованным. Сбой подсчёта
  расхода больше не глотается: от него зависит отказ, поэтому — `ErrBudgetCalculationFailed` → `500`.
- **Отзыв сессий уезжает в репозиторий пользователей**: транзакция с проверкой последнего админа уже
  есть, ей добавляется `DELETE FROM sessions` тем же `tx`. Репозиторий пользователей узнаёт таблицу
  `sessions` — обе живут в одной БД, и это дешевле отдельной абстракции транзакций. Взамен из
  `auth.SessionRepository` уходит `DeleteByUser` (SQL не дублируется), из `services` — `SessionRevoker`,
  из `auth.Service` — `RevokeAllSessions`. Ограничение, которое план не закрывает: параллельный
  `Login` мог проверить старый пароль до коммита и создать сессию после (`auth/service.go:82–101`);
  для двух пользователей это принимается и записывается в бэклог.
- **`PATCH` одной записью**: `UserRepository.Patch(ctx, id, role *Role, active *bool)` — один `UPDATE`
  с проверкой «останется ли активный админ» до записи; `UpdateRole`/`SetActive` уходят. Порядок
  проверок в сервисе прежний: самодеактивация (`CANNOT_DEACTIVATE_SELF`) раньше `LAST_ADMIN`, поэтому
  единственный админ, патчащий себя обоими полями, получает `CANNOT_DEACTIVATE_SELF`; `LAST_ADMIN` на
  два поля разом проверяется на уровне репозитория. Описание `patchUser` в спеке переписывается.
- **Порядок без некомпилируемых промежутков**: задача 5 меняет репозиторий и `auth` вместе, оставляя
  `UpdateRole`/`SetActive` обёртками над `Patch`; задача 6 меняет сервис и хендлер и снимает обёртки.
- **Бэкап — только измерение.** Телефон ждёт 30 с, серверные продления (`WithoutCancel`, `Skipper`,
  `SetWriteDeadline`) ему не помогут, а `VACUUM INTO` базы на две персоны — доли секунды по ожиданию.
  Сервис логирует длительность и размер; если на проде выйдет больше ~10 с, следующий план делает
  `POST /backups` асинхронным (`202` + список). Таймауты и контексты не трогаем.

## Technical Details

- Новые коды (`handlers/errors.go`): `ErrCodeBudgetOverlap = "BUDGET_OVERLAP"`,
  `ErrCodeBudgetNameExists = "BUDGET_NAME_EXISTS"`, `ErrCodeBudgetBelowSpent = "BUDGET_BELOW_SPENT"`.
  Ответ — `respondError(c, 409, code, message)` без `details`.
- `budgetPeriodsOverlap`: `!endDate.Before(existing.StartDate) && !startDate.After(existing.EndDate)`.
- `UpdateBudget`: `applyBudgetUpdate` перестаёт сверять сумму → при смене дат
  `validateBudgetPeriodForUpdate` → `spentFor(ctx, b)` по итоговому периоду (ошибка →
  `fmt.Errorf("%w: …", ErrBudgetCalculationFailed)`) → если `req.AmountMinor != nil && *req.AmountMinor
  < spent` — `ErrBudgetAlreadyExceeded` → `SpentMinor = spent` → `Update`.
- Репозиторий бюджетов: `GetByID` — `WHERE id = ? AND is_active = 1`; `Update` пишет `is_active`
  из структуры как раньше (сервис его не меняет).
- Репозиторий пользователей: `updateGuarded` разбирается на `withTx(ctx, func(tx) error)` и проверку
  `isLastActiveAdmin`; `UpdatePassword(ctx, id, hash, keepSessionID uuid.UUID)` и
  `Patch(ctx, id, role *user.Role, active *bool)` собирают свои запросы сами. В обоих —
  `DELETE FROM sessions WHERE user_id = ? AND id != ?` тем же `tx` (`Patch` — только при
  `active != nil && !*active`; `keep = uuid.Nil` удаляет всё). `Patch` с `role == nil && active == nil` —
  ошибка программиста, `errors.New`. `dropsAdmin` = `role != nil && *role != admin` или деактивация.
- `auth.UserLookup.UpdatePassword` получает `keepSessionID`; `auth.Service.setPassword` больше не
  трогает `sessions`; `auth.SessionRepository.DeleteByUser` и его реализация удаляются.
- `UserService`: `SetActive` и `ChangeUserRole` заменяются `PatchUser(ctx, id, role *user.Role,
  active *bool, actorID uuid.UUID)`; `ErrCannotDeactivateSelf` и `ErrInvalidRole` — до репозитория.
  `NewUserService(userRepo, familyRepo)`.
- Бэкап: `backup_service.go` после `VACUUM INTO` — `slog` Info `backup created` с `duration_ms`, `size`.
- Спека: `'409': Conflict` у `createBudget`, `updateBudget`, `updateUser`; описание `Conflict` и список
  кодов в `info.description`; у `createBudget` — фраза про включительные границы; `UpdateBudgetRequest`
  без `is_active`; `patchUser` — «оба поля применяются одной записью; `409` — ничего не изменено».
- Клиент: `BudgetEditViewModel` — карта `BUDGET_OVERLAP` / `BUDGET_NAME_EXISTS` / `BUDGET_BELOW_SPENT` →
  строки; `budget_error_rejected` остаётся запасным текстом на неизвестный код.

## Implementation Steps

### Task 1: Убрать `is_active` из `PUT /budgets/:id`, удалённый бюджет — `404` по id

**Files:**
- Modify: `internal/services/dto/budget_dto.go`, `internal/services/budget_service.go`
- Modify: `internal/infrastructure/budget/budget_repository_sqlite.go`
- Modify: `internal/application/handlers/budgets.go`, `docs/api/openapi.yaml`
- Modify: `internal/services/budget_service_test.go`, `internal/infrastructure/budget/*_test.go`,
  `tests/integration/budgets_test.go`

- [ ] удалить `IsActive` из `UpdateBudgetDTO`, `UpdateBudgetRequest` и `applyBudgetUpdate`
- [ ] `GetByID`: `AND is_active = 1`
- [ ] удалить `is_active` из `UpdateBudgetRequest` в спеке
- [ ] обновить unit-тесты сервиса, которые передавали `IsActive`
- [ ] тест репозитория: после `Delete` `GetByID` → `budget.ErrNotFound`
- [ ] интеграционные тесты: `PUT` с `is_active: false` не выключает бюджет (поле игнорируется);
      после `DELETE` — `GET`/`PUT`/`DELETE` по id → `404`
- [ ] `make fmt && make test && make lint` зелёные

### Task 2: Коды `409` для бизнес-отказов бюджета

**Files:**
- Modify: `internal/application/handlers/errors.go`, `internal/application/handlers/budgets.go`
- Modify: `docs/api/openapi.yaml`
- Modify: `internal/application/handlers/budgets_test.go`, `tests/integration/budgets_test.go`

- [ ] константы `ErrCodeBudgetOverlap`, `ErrCodeBudgetNameExists`, `ErrCodeBudgetBelowSpent` и сообщения
- [ ] `handleBudgetServiceError`: три ветки `409`; `ErrBudgetAmountTooLarge` и `dto.*` остаются `422`
- [ ] спека: `'409'` у `createBudget` и `updateBudget`, коды в описании `Conflict` и в шапке
- [ ] handler-тесты на каждую из трёх ошибок (статус, код, отсутствие `details`)
- [ ] интеграционные тесты: пересечение, занятое имя, сумма ниже расхода → `409` с кодом
- [ ] `make fmt && make test && make lint` зелёные

### Task 3: Включительная проверка пересечения периодов

**Files:**
- Modify: `internal/services/budget_service.go`, `docs/api/openapi.yaml`
- Modify: `internal/services/budget_service_test.go`

- [ ] `budgetPeriodsOverlap` — нестрогое сравнение
- [ ] спека: у `createBudget` фраза «границы включительные, общий день — пересечение»
- [ ] unit-тесты: конец = начало соседа → пересечение; конец = начало − 1 день → нет; другая категория → нет
- [ ] регрессионный unit-тест на старую пару с общим днём (создана в обход проверки): переименование
      проходит, сдвиг конца второго → `ErrBudgetOverlapExists`, сдвиг начала второго на день → проходит
- [ ] `make fmt && make test && make lint` зелёные

### Task 4: `PUT` бюджета сверяет сумму с расходом по новым датам

**Files:**
- Modify: `internal/services/budget_service.go`
- Modify: `internal/services/budget_service_test.go`

- [ ] вынести расчёт расхода из `recalculateAndUpdateSpent` в чистую `spentFor(ctx, b)`; запись — отдельно
- [ ] `UpdateBudget`: применить поля → проверить пересечение → `spentFor` по итоговому периоду (ошибка →
      `ErrBudgetCalculationFailed`, апдейт не идёт) → сверить сумму, только если она в запросе → записать
- [ ] unit-тест: сужение периода + уменьшение суммы, по новому периоду расход укладывается → успех
- [ ] unit-тест: сумма меньше расхода по новому периоду → `ErrBudgetAlreadyExceeded`
- [ ] unit-тест: расширение периода без суммы → сохраняется, `SpentMinor` пересчитан
- [ ] unit-тест: отказ репозитория транзакций → `ErrBudgetCalculationFailed`, `Update` не вызван
- [ ] `make fmt && make test && make lint` зелёные

### Task 5: Репозиторий пользователей и `auth` — сессии в одной транзакции

**Files:**
- Modify: `internal/infrastructure/user/user_repository_sqlite.go`, `internal/services/user_service.go`
  (интерфейс `UserRepository`)
- Modify: `internal/auth/service.go`
- Modify: `internal/infrastructure/user/user_repository_test.go` (в т.ч. `:403` → `Patch`),
  `internal/auth/service_test.go`, `internal/auth/export_test.go`, `internal/services/helpers_test.go`
  (фейк репозитория)

- [ ] `withTx` из `updateGuarded`; `Patch(ctx, id, role, active)` — один `UPDATE`, проверка последнего
      админа до записи, при деактивации `DELETE FROM sessions` тем же `tx`
- [ ] `UpdatePassword(ctx, id, hash, keepSessionID)` — хеш и удаление сессий кроме `keep` в одной транзакции
- [ ] `UpdateRole`/`SetActive` — обёртки над `Patch` (снимаются в задаче 6)
- [ ] `auth.UserLookup.UpdatePassword(ctx, id, hash, keepSessionID)`; `setPassword` без `sessions.DeleteByUser`
- [ ] тесты репозитория: пароль + сессии кроме `keep`; `Patch` только роль; только активность; оба поля;
      обе комбинации на последнем админе → `ErrLastAdmin`, ничего не записано, сессии целы; деактивация
      удаляет сессии; повышение неактивного до admin проходит; `Patch` без полей → ошибка
- [ ] тест атомарности: триггер `BEFORE DELETE ON sessions … RAISE(ABORT)` → `UpdatePassword` и
      деактивация возвращают ошибку, хеш/активность/сессии прежние
- [ ] тесты `auth`: `ChangePassword` передаёт `keepSessionID`, `AdminSetPassword` — `uuid.Nil`;
      отказ репозитория → ошибка без побочных эффектов
- [ ] `make fmt && make test && make lint` зелёные

### Task 6: `UserService.PatchUser` и `PATCH /users/:id` одной записью

**Files:**
- Modify: `internal/services/user_service.go`, `internal/services/interfaces.go`,
  `internal/services/container.go`
- Modify: `internal/auth/service.go`, `internal/auth/session.go`,
  `internal/infrastructure/auth/session_repository_sqlite.go` (удалить `RevokeAllSessions`, `DeleteByUser`)
- Modify: `internal/infrastructure/user/user_repository_sqlite.go` (снять `UpdateRole`/`SetActive`)
- Modify: `internal/application/handlers/users.go`, `docs/api/openapi.yaml` (описание `patchUser`)
- Modify: `internal/services/user_service_test.go`, `internal/services/helpers_test.go`,
  `internal/auth/service_test.go` (`:108`, `:313`, `:507`),
  `internal/infrastructure/auth/session_repository_test.go`,
  `internal/application/handlers/users_test.go`, `internal/application/http_server_test.go`,
  `tests/integration/session_revalidation_test.go`, `tests/integration/api_auth_test.go`,
  `tests/integration/api_users_deactivate_test.go`, `tests/integration/users_test.go`

- [ ] `PatchUser(ctx, id, role *user.Role, active *bool, actorID)`: `ErrCannotDeactivateSelf`,
      `ErrInvalidRole`, затем `userRepo.Patch`; `SetActive`/`ChangeUserRole` удалить из `UserService`
- [ ] удалить `SessionRevoker`, `auth.Service.RevokeAllSessions`, `SessionRepository.DeleteByUser` и
      реализацию, обёртки `UpdateRole`/`SetActive`; `NewUserService(userRepo, familyRepo)`; `container.go`
- [ ] хендлер `PatchUser`: один вызов `PatchUser`
- [ ] спека `patchUser`: оба поля применяются одной записью, при `409` ничего не изменено
- [ ] обновить моки и фейки (`users_test.go`, `http_server_test.go`, `helpers_test.go`,
      `service_test.go`) и прямые вызовы в интеграции (`session_revalidation_test.go`, `api_auth_test.go`)
- [ ] unit-тесты сервиса: самодеактивация раньше `LAST_ADMIN`, невалидная роль, проброс `ErrLastAdmin`
- [ ] интеграция: единственный админ патчит себя `{role: member, is_active: false}` →
      `409 CANNOT_DEACTIVATE_SELF`, роль не изменилась; второго админа теми же полями → `200`, оба
      применены; деактивация → следующий запрос с её токеном `401`; `PUT /users/:id/password` →
      старые сессии `401`
- [ ] `make fmt && make test && make lint` зелёные

### Task 7: Спека — `409` у `updateUser`

**Files:**
- Modify: `docs/api/openapi.yaml`
- Modify: `tests/integration/users_test.go`

- [ ] `'409': { $ref: '#/components/responses/Conflict' }` у `updateUser`, `EMAIL_TAKEN` упомянут в описании
- [ ] интеграционный тест: `PUT /users/:id` с чужим email → `409 EMAIL_TAKEN`
- [ ] `make fmt && make test && make lint` зелёные (`TestOpenAPISpec_*` в том числе)

### Task 8: Лог длительности `VACUUM INTO`

**Files:**
- Modify: `internal/services/backup_service.go`
- Modify: `internal/services/backup_service_test.go`

- [ ] после `VACUUM INTO` — `slog` Info `backup created` с `duration_ms` и `size` (через логгер сервиса)
- [ ] unit-тест: успешный бэкап пишет запись с обоими полями (логгер на `bytes.Buffer`)
- [ ] `make fmt && make test && make lint` зелёные

### Task 9: Android — перегенерация и коды `409` бюджетов

**Files:**
- Regenerate: `android/core/api/generated/**`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetEditViewModel.kt`
- Modify: `android/app/src/main/res/values/strings.xml`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetEditViewModelTest.kt`

- [ ] `make -C android api-gen`, закоммитить; `make -C android api-check`
- [ ] карта `BUDGET_OVERLAP` / `BUDGET_NAME_EXISTS` / `BUDGET_BELOW_SPENT` → свои строки; неизвестный
      `409` → `budget_error_rejected`
- [ ] тесты ViewModel: каждый код → своя строка; `422` под полями как раньше
- [ ] `make -C android check` зелёный

### Task 10: Verify acceptance criteria
- [ ] все пункты Overview реализованы; `is_active` нигде не принимается от клиента
- [ ] `grep -rn 'DeleteByUser\|RevokeAllSessions\|SessionRevoker' internal tests` пуст
- [ ] `make fmt && make test && make lint` — 0 issues; `make -C android check`

### Task 11: [Final] Update documentation
- [ ] `CLAUDE.md`: «User writes are column-scoped» — `UpdatePassword(keep)`, `Patch`, отзыв сессий в
      репозитории; коды `409` бюджетов; удалённый бюджет — `404` по id
- [ ] `docs/backlog.md`: удалить оба раздела «Сервер: находки …»; вписать три строки: решение по
      `is_active` бюджета, `UNIQUE` бюджетов считает удалённые строки, гонка `Login` со сменой пароля
- [ ] `docs/patterns/error_handling.md` — новые коды, если там есть список
- [ ] перенести план в `docs/plans/completed/`

## Post-Completion

**Ручная проверка:**
- на проде после выката: `POST /api/v1/backups` с телефона, `duration_ms` из `docker compose logs app`;
  стабильно больше ~10 с — отдельный план: асинхронный `202` + опрос списка.
- проверить `sqlite-shell`: есть ли активные бюджеты одной области с общим граничным днём — если да,
  сдвинуть границу на день руками, иначе их даты через приложение не поправить.

**Внешние системы:**
- релиз сервера `v0.2.0` (изменился контракт), затем клиент `app-v0.4.0` — старый клиент к новому
  серверу совместим (он не шлёт `is_active`, а `409` покажет запасным текстом), обратное — нет.

# BUG-001: Panic при создании отчёта (nil pointer dereference)

## Приоритет: КРИТИЧЕСКИЙ

## Статус: ✅ ИСПРАВЛЕНО

## Описание проблемы

При попытке создать отчёт через веб-интерфейс происходит panic с ошибкой
`runtime error: invalid memory address or nil pointer dereference`.

## Воспроизведение

1. Войти в систему
2. Перейти на страницу "Отчёты" (`/reports`)
3. Заполнить форму создания отчёта
4. Нажать "Save & Create Report"

## Лог ошибки

```
[PANIC RECOVER] runtime error: invalid memory address or nil pointer dereference
goroutine 15 [running]:
family-budget-service/internal/web/handlers.(*ReportHandler).Create(0xc0003cdde0, {0xe3e360, 0xc000154000})
	/home/sasha/Project/Family-Finances-Service/internal/web/handlers/reports.go:151 +0x1c2
```

## Локация бага

- **Файл**: `internal/web/handlers/reports.go`
- **Строка**: 151
- **Метод**: `(*ReportHandler).Create`

## Анализ

Panic происходил при обращении к nil указателю в методе Create. Корневая причина:
- При HTMX запросах функции `handleReportGenerationError` и `handleUnsupportedReportType`
  вызывали `renderPartial`, который возвращал `nil` при успешном рендеринге
- Это приводило к возврату `nil, nil` из `generateReport`
- Код продолжал выполнение и пытался обратиться к `reportEntity.ID` без проверки на nil

## Исправления

1. Добавлен sentinel error `errHTMXResponseSent` для обозначения уже отправленного HTMX ответа
2. Обновлены `handleReportGenerationError` и `handleUnsupportedReportType` для возврата sentinel error
3. Добавлена проверка `errors.Is(err, errHTMXResponseSent)` в методе Create
4. Добавлена защитная проверка `reportEntity == nil` для предотвращения подобных проблем

## Задачи по исправлению

1. [x] Открыть файл `internal/web/handlers/reports.go` и найти строку 151
2. [x] Определить, какой указатель является nil
3. [x] Добавить проверку на nil перед использованием
4. [x] Добавить корректную обработку ошибок
5. [x] Написать тест, воспроизводящий баг
6. [x] Убедиться, что тест проходит после исправления
7. [x] Запустить `make lint` и `make test`

## Связанные файлы

- `internal/web/handlers/reports.go` - основное исправление
- `internal/web/handlers/reports_test.go` - новые тесты

## Критерии приёмки

- [x] Отчёты создаются без panic
- [x] При ошибках отображается понятное сообщение пользователю
- [x] Добавлен unit-тест на edge case
- [x] Все тесты проходят
- [x] Линтер не выдаёт ошибок

## Дата исправления

2026-02-04

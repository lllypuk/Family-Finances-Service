# BUG-002: Ошибка валидации Limit при генерации отчёта

## Приоритет: ВЫСОКИЙ

## Описание проблемы

При попытке сгенерировать предпросмотр отчёта возникает ошибка валидации:

```
Failed to process report: failed to get expense transactions: validation failed:
Key: 'TransactionFilterDTO.Limit' Error:Field validation for 'Limit' failed on the 'min' tag
```

## Воспроизведение

1. Войти в систему
2. Перейти на страницу "Отчёты" (`/reports`)
3. Выбрать тип отчёта (например, "Expenses Report" или "Cash Flow Summary")
4. Нажать "Generate Report Preview"
5. Появляется ошибка "Произошла ошибка при загрузке данных"

## Анализ

Проблема в том, что при создании отчёта передаётся `TransactionFilterDTO` с `Limit = 0`, что не проходит валидацию по
тегу `min`.

## Локация бага

- **Вероятные файлы**:
    - `internal/web/handlers/reports.go` (метод генерации)
    - `internal/application/handlers/dto.go` (определение TransactionFilterDTO)
    - `internal/services/report_service.go`

## Задачи по исправлению

1. [ ] Найти определение `TransactionFilterDTO` и проверить теги валидации
2. [ ] Найти место, где создаётся фильтр для отчёта
3. [ ] Установить дефолтное значение Limit (например, 1000 или убрать min валидацию)
4. [ ] Либо изменить валидацию: `min=0` вместо `min=1` (если 0 означает "без лимита")
5. [ ] Добавить тест на генерацию отчёта
6. [ ] Запустить `make lint` и `make test`

## Варианты решения

### Вариант 1: Установить дефолтное значение

```go
filter := TransactionFilterDTO{
Limit: 1000, // дефолтный лимит
// ...
}
```

### Вариант 2: Изменить валидацию

```go
type TransactionFilterDTO struct {
Limit int `validate:"min=0"` // 0 = без лимита
}
```

### Вариант 3: Сделать Limit опциональным

```go
type TransactionFilterDTO struct {
Limit *int `validate:"omitempty,min=1"`
}
```

## Связанные файлы

- `internal/application/handlers/dto.go`
- `internal/web/handlers/reports.go`
- `internal/services/report_service.go`
- `internal/web/handlers/htmx.go` (если есть HTMX endpoint)

## Критерии приёмки

- [ ] Предпросмотр отчёта генерируется без ошибок
- [ ] Отчёты создаются и сохраняются корректно
- [ ] Валидация работает для корректных случаев
- [ ] Добавлены тесты
- [ ] Все тесты проходят
- [ ] Линтер не выдаёт ошибок

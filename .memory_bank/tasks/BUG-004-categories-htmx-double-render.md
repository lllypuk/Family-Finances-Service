# BUG-004: HTMX отображает форму и список категорий одновременно

## Приоритет: НИЗКИЙ (UI/UX)

## Описание проблемы

После создания новой категории страница отображает одновременно:

1. Форму создания категории (с заполненными данными)
2. Список всех категорий

Ожидаемое поведение: после успешного создания должен отображаться только список категорий.

## Воспроизведение

1. Войти в систему
2. Перейти на страницу категорий (`/categories`)
3. Нажать "Добавить категорию"
4. Заполнить форму и нажать "Создать категорию"
5. Видим и форму, и список одновременно

## Анализ

Проблема связана с HTMX обработкой ответа:

1. Форма отправляется через HTMX
2. Сервер возвращает redirect (303) на `/categories`
3. HTMX получает HTML списка категорий
4. Но вместо замены контента, HTML вставляется внутрь формы

## Локация бага

- **Шаблон формы**: `internal/web/templates/categories/new.html`
- **Шаблон списка**: `internal/web/templates/categories/index.html`
- **Handler**: `internal/web/handlers/categories.go`

## Возможные причины

1. Неправильный `hx-target` в форме
2. Отсутствует `hx-swap` или неверное значение
3. Сервер не устанавливает правильные HTMX заголовки
4. Redirect обрабатывается некорректно

## Задачи по исправлению

1. [ ] Проверить HTMX атрибуты в форме создания категории
2. [ ] Убедиться, что `hx-target` указывает на контейнер всей страницы или body
3. [ ] Проверить, что handler возвращает правильный ответ для HTMX
4. [ ] Рассмотреть использование `HX-Redirect` заголовка вместо 303
5. [ ] Протестировать исправление через agent-browser
6. [ ] Запустить `make lint` и `make test`

## Варианты решения

### Вариант 1: Использовать HX-Redirect заголовок

```go
func (h *CategoryHandler) Create(c echo.Context) error {
// ... создание категории ...

if isHTMX(c) {
c.Response().Header().Set("HX-Redirect", "/categories")
return c.NoContent(http.StatusOK)
}
return c.Redirect(http.StatusSeeOther, "/categories")
}
```

### Вариант 2: Исправить hx-target в шаблоне

```html

<form hx-post="/categories"
      hx-target="body"
      hx-swap="innerHTML"
      hx-push-url="true">
```

### Вариант 3: Использовать hx-redirect атрибут

```html

<form hx-post="/categories"
      hx-redirect="/categories">
```

## Связанные файлы

- `internal/web/templates/categories/new.html`
- `internal/web/templates/categories/index.html`
- `internal/web/handlers/categories.go`

## Критерии приёмки

- [ ] После создания категории отображается только список
- [ ] Форма очищается/скрывается после успешной отправки
- [ ] URL в браузере обновляется на `/categories`
- [ ] Протестировано через agent-browser
- [ ] Линтер не выдаёт ошибок

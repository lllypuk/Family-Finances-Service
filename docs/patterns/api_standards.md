# API Standards - Стандарты проектирования API

## 🎯 Общие принципы

### RESTful Design
- **Ресурсно-ориентированная архитектура**: URL представляют ресурсы, а не действия
- **Единообразие интерфейса**: Последовательное использование HTTP методов
- **Stateless**: Каждый запрос содержит всю необходимую информацию
- **Кешируемость**: Ответы должны явно указывать возможность кеширования

### API-First подход
- API проектируется до реализации
- Спецификация OpenAPI как источник истины
- Контрактное тестирование
- Версионирование с самого начала

## 🌐 URL Design

### Структура URL
```
https://api.familyfinances.com/api/v1/{resource}/{id}/{sub-resource}
```

### Правила именования ресурсов
- **Используйте существительные**, не глаголы
- **Множественное число** для коллекций: `/families`, `/transactions`
- **Единственное число** для отдельных ресурсов: `/families/{id}`
- **Kebab-case** для составных имен: `/financial-goals`
- **Иерархическая структура** для связанных ресурсов

### Примеры правильных URL
```
GET    /api/v1/families                    # Список семей
GET    /api/v1/families/{id}               # Конкретная семья
GET    /api/v1/families/{id}/members       # Члены семьи
GET    /api/v1/families/{id}/transactions  # Транзакции семьи
POST   /api/v1/families/{id}/transactions  # Создание транзакции
PUT    /api/v1/transactions/{id}           # Обновление транзакции
DELETE /api/v1/transactions/{id}           # Удаление транзакции
```

### ❌ Неправильные примеры
```
❌ GET /api/v1/getFamily/{id}              # Глагол в URL
❌ GET /api/v1/family                      # Единственное число для коллекции
❌ GET /api/v1/families/{id}/getMembers    # Глагол в URL
❌ POST /api/v1/createTransaction          # Действие вместо ресурса
```

## 🔧 HTTP методы

### Стандартное использование
| Метод | Назначение | Идемпотентность | Body |
|-------|------------|-----------------|------|
| GET | Получение данных | ✅ Да | ❌ Нет |
| POST | Создание ресурса | ❌ Нет | ✅ Да |
| PUT | Полное обновление | ✅ Да | ✅ Да |
| PATCH | Частичное обновление | ❌ Нет | ✅ Да |
| DELETE | Удаление ресурса | ✅ Да | ❌ Нет |

### Семантика методов
```
GET /families/{id}
# Возвращает: 200 + данные семьи
# Ошибки: 404 если не найдена, 403 если нет доступа

POST /families
# Тело: данные новой семьи
# Возвращает: 201 + созданная семья + Location header
# Ошибки: 400 если невалидные данные, 409 если конфликт

PUT /families/{id}
# Тело: полные данные семьи
# Возвращает: 200 + обновленная семья
# Ошибки: 404 если не найдена, 400 если невалидные данные

PATCH /families/{id}
# Тело: частичные данные для обновления
# Возвращает: 200 + обновленная семья
# Ошибки: 404 если не найдена, 400 если невалидные данные

DELETE /families/{id}
# Возвращает: 204 (No Content)
# Ошибки: 404 если не найдена, 409 если есть зависимости
```

## 📊 HTTP статус-коды

### Успешные ответы (2xx)
- **200 OK**: Успешный GET, PUT, PATCH
- **201 Created**: Успешный POST с созданием ресурса
- **204 No Content**: Успешный DELETE
- **206 Partial Content**: Частичное содержимое (пагинация)

### Ошибки клиента (4xx)
- **400 Bad Request**: Невалидный запрос или данные
- **401 Unauthorized**: Требуется аутентификация
- **403 Forbidden**: Аутентификация есть, но нет прав доступа
- **404 Not Found**: Ресурс не найден
- **409 Conflict**: Конфликт с текущим состоянием ресурса
- **422 Unprocessable Entity**: Валидация бизнес-правил
- **429 Too Many Requests**: Превышение лимита запросов

### Ошибки сервера (5xx)
- **500 Internal Server Error**: Внутренняя ошибка сервера
- **502 Bad Gateway**: Ошибка upstream сервиса
- **503 Service Unavailable**: Сервис временно недоступен

## 📝 Формат запросов и ответов

### Content-Type
- **Запросы**: `application/json`
- **Ответы**: `application/json`
- **Загрузка файлов**: `multipart/form-data`

### Структура ответа
```json
{
  "data": {
    // Основные данные ответа
  },
  "meta": {
    "timestamp": "2024-12-15T10:30:00Z",
    "request_id": "req_abc123",
    "version": "v1"
  },
  "pagination": {  // Только для списков
    "page": 1,
    "page_size": 20,
    "total_count": 150,
    "total_pages": 8
  }
}
```

### Структура ошибки
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validation failed for request",
    "details": [
      {
        "field": "email",
        "message": "Invalid email format",
        "code": "INVALID_FORMAT"
      }
    ]
  },
  "meta": {
    "timestamp": "2024-12-15T10:30:00Z",
    "request_id": "req_abc123",
    "version": "v1"
  }
}
```

## 🔍 Пагинация

### Параметры запроса
```
GET /families/{id}/transactions?page=1&page_size=20&sort=created_at:desc
```

### Поддерживаемые параметры
- **page**: Номер страницы (начиная с 1)
- **page_size**: Размер страницы (по умолчанию 20, максимум 100)
- **sort**: Сортировка в формате `field:direction` (asc/desc)

### Ответ с пагинацией
```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total_count": 150,
    "total_pages": 8,
    "has_next": true,
    "has_previous": false
  },
  "links": {
    "self": "/api/v1/transactions?page=1&page_size=20",
    "next": "/api/v1/transactions?page=2&page_size=20",
    "last": "/api/v1/transactions?page=8&page_size=20"
  }
}
```

## 🔐 Аутентификация и авторизация

### Сессионная cookie — единственный способ аутентификации

API-токенов (JWT, Bearer, API keys) в проекте **нет**. Вся группа `/api/v1` закрыта тем же
механизмом, что и веб-интерфейс: подписанная cookie сессии (`family-budget-session`).
Программный клиент обязан держать cookie jar — вход выполняется через форму `/login`.

```
Cookie: family-budget-session=<подписанное значение>
X-Csrf-Token: <токен из сессии>   # для POST/PUT/DELETE
```

### Контракт ответов

| Запрос | Ответ |
|---|---|
| Любой `/api/v1/*` без валидной сессии | `401` + `{"error":{"code":"UNAUTHORIZED","message":"Authentication required"}}` |
| `POST`/`PUT`/`DELETE` без `X-Csrf-Token` | `403 CSRF token validation failed` |
| Сессия валидна, но роль не подходит маршруту | `403` + `{"error":{"code":"FORBIDDEN","message":"Insufficient permissions"}}` |

Важный порядок: CSRF-middleware зарегистрирован глобально на всём Echo-инстансе и
отрабатывает **раньше** проверки аутентификации. Поэтому запись без CSRF-токена получит
`403`, а не `401`, даже если сессии нет вовсе. Токен без сессии — `401`.

### Роли и права доступа

Роли те же, что в вебе: **admin**, **member**, **child** (никаких `family_admin` /
`viewer`). Ролевая модель API повторяет веб-маршруты:

- **только `admin`** — вся группа `/api/v1/users` (включая чтение) и удаление категории
  (`DELETE /api/v1/categories/:id`: через API оно необратимо и не имеет подтверждения,
  которое есть в UI)
- **`admin` или `member`** — финансовые разделы: `/api/v1/{categories,transactions,budgets,reports}`;
  роль `child` получает `403`

Автор записи берётся из сессии, а не из тела запроса: `user_id` отсутствует в
`CreateTransactionRequest` / `CreateReportRequest`, и подстановка чужого ID в теле
игнорируется.

## 🔄 Версионирование

### Стратегия версионирования
- **URI Versioning**: `/api/v1/`, `/api/v2/`
- **Semantic Versioning**: Major.Minor.Patch
- **Backward Compatibility**: Минимум 2 версии одновременно

### Управление версиями
```
# Текущая версия
GET /api/v1/families

# Новая версия с breaking changes
GET /api/v2/families

# Deprecation warnings
Sunset: Wed, 31 Dec 2024 23:59:59 GMT
Deprecation: true
Link: </api/v2/families>; rel="successor-version"
```

## 🏷️ Соглашения по именованию

### Поля JSON
- **snake_case** для всех полей: `created_at`, `family_id`
- **ISO 8601** для дат: `2024-12-15T10:30:00Z`
- **Boolean** значения: `true`/`false`
- **null** для отсутствующих значений

### Примеры полей
```json
{
  "id": "fam_abc123",
  "name": "Smith Family",
  "created_at": "2024-12-15T10:30:00Z",
  "updated_at": "2024-12-15T10:30:00Z",
  "is_active": true,
  "member_count": 4,
  "monthly_budget": 5000.00,
  "currency": "USD",
  "settings": {
    "notifications_enabled": true,
    "auto_categorization": false
  }
}
```

## 📏 Ограничения и лимиты

### Rate Limiting
- **Анонимные пользователи**: 100 запросов/час
- **Аутентифицированные**: 1000 запросов/час
- **Premium**: 10000 запросов/час

### Headers для Rate Limiting
```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 742
X-RateLimit-Reset: 1640995200
Retry-After: 3600
```

### Размеры запросов
- **Максимальный размер тела**: 10MB
- **Максимальный размер массива**: 1000 элементов
- **Максимальная длина строки**: 1000 символов

## 🔍 Фильтрация и поиск

### Параметры фильтрации
```
GET /transactions?category=food&amount_gte=100&date_from=2024-01-01
```

### Поддерживаемые операторы
- **Равенство**: `field=value`
- **Сравнение**: `field_gte=value`, `field_lte=value`
- **Диапазон**: `field_from=value1&field_to=value2`
- **Включение**: `field_in=value1,value2,value3`
- **Поиск**: `search=query` (полнотекстовый поиск)

## 📊 Мониторинг и логирование

### Request ID
Каждый запрос должен иметь уникальный идентификатор:
```
X-Request-ID: req_abc123def456
```

### Логирование запросов
```json
{
  "timestamp": "2024-12-15T10:30:00Z",
  "level": "INFO",
  "request_id": "req_abc123",
  "method": "GET",
  "path": "/api/v1/families/123",
  "status": 200,
  "duration": 45,
  "user_id": "user_456",
  "ip": "192.168.1.1"
}
```

## ✅ Чек-лист для нового API

### Дизайн
- [ ] Следует REST принципам
- [ ] Использует правильные HTTP методы
- [ ] Корректные статус-коды
- [ ] Консистентное именование

### Документация
- [ ] OpenAPI спецификация
- [ ] Примеры запросов/ответов
- [ ] Описание ошибок
- [ ] Swagger UI доступен

### Безопасность
- [ ] Аутентификация настроена
- [ ] Авторизация проверена
- [ ] Валидация входных данных
- [ ] Rate limiting применен

### Тестирование
- [ ] Unit тесты
- [ ] Integration тесты
- [ ] Contract тесты
- [ ] Load тесты

---

*Документ создан: 2025*
*Владелец: API Team*
*Регулярность обновлений: при изменении стандартов*

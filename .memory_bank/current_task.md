# Текущая задача: Реализация веб-интерфейса Phase 1

> **Статус**: 🔄 В разработке
> **Приоритет**: 🔴 Высокий
> **Срок**: 1-2 недели
> **Дата начала**: 2025-08-15

## 📋 Цель Phase 1: Базовая инфраструктура веб-интерфейса

Создание фундаментальной инфраструктуры для веб-интерфейса с использованием HTMX и PicoCSS, включая аутентификацию, базовые layouts и middleware.

## 🎯 Основные задачи Phase 1

### 1. Настройка веб-инфраструктуры
- [ ] **Создание структуры каталогов для веб-интерфейса**
  - `internal/web/handlers/` - обработчики веб-страниц
  - `internal/web/middleware/` - middleware для аутентификации и сессий
  - `internal/web/templates/` - HTML шаблоны
  - `internal/web/static/` - статические файлы (CSS, JS, images)

- [ ] **Настройка шаблонизатора Go Templates**
  - Базовые layouts (base.html, auth.html)
  - Система компонентов (forms, tables, cards)
  - Поддержка вложенных templates

- [ ] **Интеграция статических файлов**
  - Подключение PicoCSS framework
  - Интеграция HTMX библиотеки
  - Настройка обслуживания статических файлов

### 2. Аутентификация и авторизация
- [ ] **Система сессий**
  - Session store (in-memory для начала)
  - CSRF защита
  - Session middleware

- [ ] **Аутентификация пользователей**
  - Страницы входа/регистрации
  - Обработка форм входа
  - Redirect после входа

- [ ] **Ролевая авторизация**
  - Middleware для проверки прав доступа
  - Декораторы для защищенных маршрутов
  - Поддержка ролей: admin, member, child

### 3. Базовые веб-страницы
- [ ] **Главная страница (дашборд)**
  - Layout с навигацией
  - Базовая статистика семьи
  - Responsive дизайн

- [ ] **Страницы аутентификации**
  - Форма входа с валидацией
  - Форма регистрации семьи
  - Обработка ошибок

- [ ] **Навигация и меню**
  - Responsive навигационное меню
  - Индикация текущей страницы
  - Выход из системы

### 4. HTMX интеграция
- [ ] **Базовые HTMX endpoints**
  - `/htmx/dashboard/stats` - обновление статистики
  - `/htmx/forms/validate` - валидация форм
  - Обработка HTMX запросов в handlers

- [ ] **Обработка ошибок HTMX**
  - Централизованная обработка ошибок
  - Отображение сообщений пользователю
  - Fallback для отключенного JavaScript

## 🏗️ Техническая реализация

### Анализ существующей архитектуры
**Текущая структура проекта:**
- ✅ Echo v4 HTTP server уже настроен
- ✅ Repository pattern реализован
- ✅ Domain entities готовы
- ✅ Observability и middleware настроены
- ✅ MongoDB интеграция работает

**Что нужно добавить:**
- Веб-handlers для HTML страниц (дополнительно к API)
- Template rendering система
- Session management
- Static files serving
- CSRF защита

### Структура новых файлов

```
internal/web/
├── handlers/
│   ├── auth.go              # Аутентификация (login, register, logout)
│   ├── dashboard.go         # Главная страница
│   ├── base.go              # Базовые функции для всех handlers
│   └── htmx.go              # HTMX endpoints
├── middleware/
│   ├── auth.go              # Проверка аутентификации
│   ├── session.go           # Управление сессиями
│   ├── csrf.go              # CSRF защита
│   └── template.go          # Template rendering middleware
├── templates/
│   ├── layouts/
│   │   ├── base.html        # Основной layout
│   │   └── auth.html        # Layout для страниц авторизации
│   ├── pages/
│   │   ├── dashboard.html   # Главная страница
│   │   ├── login.html       # Страница входа
│   │   └── register.html    # Регистрация семьи
│   └── components/
│       ├── nav.html         # Навигация
│       ├── header.html      # Шапка сайта
│       └── footer.html      # Подвал
└── static/
    ├── css/
    │   ├── pico.min.css     # PicoCSS framework
    │   └── custom.css       # Кастомные стили
    ├── js/
    │   ├── htmx.min.js      # HTMX библиотека
    │   └── app.js           # Дополнительная логика
    └── img/
        └── favicon.ico
```

### Интеграция с существующим кодом

**Расширение HTTP сервера:**
```go
// В internal/application/http_server.go добавить:
func (s *HTTPServer) setupWebRoutes() {
    // Статические файлы
    s.echo.Static("/static", "internal/web/static")

    // Веб-страницы
    s.echo.GET("/", webHandlers.Dashboard)
    s.echo.GET("/login", webHandlers.LoginPage)
    s.echo.POST("/login", webHandlers.Login)
    s.echo.GET("/register", webHandlers.RegisterPage)
    s.echo.POST("/register", webHandlers.Register)
    s.echo.POST("/logout", webHandlers.Logout)

    // HTMX endpoints
    htmx := s.echo.Group("/htmx")
    htmx.GET("/dashboard/stats", webHandlers.DashboardStats)
}
```

### Модели для веб-интерфейса

```go
// internal/web/models/
type SessionData struct {
    UserID   uuid.UUID  `json:"user_id"`
    FamilyID uuid.UUID  `json:"family_id"`
    Role     user.Role  `json:"role"`
    Email    string     `json:"email"`
    ExpiresAt time.Time `json:"expires_at"`
}

type DashboardData struct {
    User             *user.User
    Family           *user.Family
    TotalIncome      float64
    TotalExpenses    float64
    NetIncome        float64
    TransactionCount int
    BudgetCount      int
}

type FormErrors map[string]string
```

## 📊 План выполнения

### Week 1: Инфраструктура и аутентификация
**День 1-2: Базовая настройка**
- [ ] Создание структуры каталогов
- [ ] Настройка template renderer
- [ ] Подключение статических файлов
- [ ] Базовый layout с PicoCSS

**День 3-4: Аутентификация**
- [ ] Session middleware
- [ ] Страницы входа/регистрации
- [ ] CSRF защита
- [ ] Тестирование auth flow

**День 5: Интеграция**
- [ ] Подключение к существующему HTTP серверу
- [ ] Настройка маршрутов
- [ ] Базовое тестирование

### Week 2: Дашборд и HTMX
**День 1-3: Главная страница**
- [ ] Dashboard layout и дизайн
- [ ] Интеграция с API для получения данных
- [ ] Responsive навигация
- [ ] Статистические карточки

**День 4-5: HTMX интеграция**
- [ ] Базовые HTMX endpoints
- [ ] Динамическое обновление статистики
- [ ] Валидация форм через HTMX
- [ ] Обработка ошибок

## ✅ Критерии готовности Phase 1

### Функциональные требования
- [ ] Пользователь может зарегистрировать семью
- [ ] Пользователь может войти в систему
- [ ] Главная страница отображает базовую статистику
- [ ] Навигация работает корректно
- [ ] HTMX обновления работают без перезагрузки страницы

### Технические требования
- [ ] Responsive дизайн (мобильные устройства)
- [ ] CSRF защита активна
- [ ] Session management работает
- [ ] Обработка ошибок реализована
- [ ] Code coverage > 70%

### Безопасность
- [ ] Все формы защищены от CSRF
- [ ] Пароли хешируются
- [ ] Session cookies защищены (HttpOnly, Secure)
- [ ] Input validation на клиенте и сервере

## 🔗 Зависимости

### Внешние библиотеки
- PicoCSS v1.5+ (уже определена в спецификации)
- HTMX v1.9+ (уже определена в спецификации)

### Внутренние компоненты
- Существующие domain entities (готовы)
- Repository interfaces (готовы)
- HTTP server infrastructure (готово)
- Observability система (готова)

## 🧪 Тестирование Phase 1

### Unit тесты
- [ ] Web handlers тестирование
- [ ] Middleware тестирование
- [ ] Template rendering тесты

### Integration тесты
- [ ] Полный flow аутентификации
- [ ] Dashboard data loading
- [ ] HTMX endpoints

### E2E тесты (базовые)
- [ ] Регистрация → Вход → Дашборд
- [ ] Responsive поведение
- [ ] JavaScript включен/выключен

## 📝 Следующие этапы

**Phase 2 (будущие 2-3 недели):**
- CRUD интерфейсы для всех сущностей
- Расширенные формы с валидацией
- Фильтрация и поиск
- Улучшенный UX

**Phase 3 (будущие 1-2 недели):**
- Отчеты с графиками
- Массовые операции
- Экспорт данных
- Advanced HTMX features

---

*Обновлено: 2025-08-15*
*Ответственный: Backend Team*
*Следующий review: 2025-08-22*

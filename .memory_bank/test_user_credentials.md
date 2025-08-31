# Test User Credentials

## Тестовый пользователь для локальной разработки

**Учетные данные:**
- **Email:** `test2@example.com`
- **Password:** `test123456`
- **Family Name:** `Test Family 2`
- **User Name:** `Test User2`
- **Currency:** `RUB`

**Статус:** ✅ Успешно создан и протестирован

## Использование для тестирования

### 1. Авторизация через cURL
```bash
# Получить CSRF токен для входа
CSRF_TOKEN=$(curl -s --noproxy '*' -c /tmp/cookies.txt 127.0.0.1:8080/login | grep 'name="_token"' | sed 's/.*value="//g' | sed 's/".*//g')

# Войти в систему
curl -s --noproxy '*' -b /tmp/cookies.txt -c /tmp/cookies.txt -X POST 127.0.0.1:8080/login \
  -d "_token=$CSRF_TOKEN" \
  -d "email=test2@example.com" \
  -d "password=test123456"

# Тестировать защищенные endpoints
curl -s --noproxy '*' -b /tmp/cookies.txt 127.0.0.1:8080/budgets/new
curl -s --noproxy '*' -b /tmp/cookies.txt 127.0.0.1:8080/categories
curl -s --noproxy '*' -b /tmp/cookies.txt 127.0.0.1:8080/transactions/new
```

### 2. Быстрый тест основных страниц
```bash
# Функция для быстрого тестирования
test_pages() {
    local cookie_file="/tmp/test_session.txt"
    
    echo "Получаем токен и авторизуемся..."
    local token=$(curl -s --noproxy '*' -c $cookie_file 127.0.0.1:8080/login | grep '_token' | sed 's/.*value="//g' | sed 's/".*//g')
    
    curl -s --noproxy '*' -b $cookie_file -c $cookie_file -X POST 127.0.0.1:8080/login \
        -d "_token=$token" -d "email=test2@example.com" -d "password=test123456" > /dev/null
    
    echo "Тестируем страницы..."
    echo "Dashboard: $(curl -s --noproxy '*' -b $cookie_file 127.0.0.1:8080/ | grep '<title>' | sed 's/<title>//g' | sed 's/<\/title>//g')"
    echo "Budgets: $(curl -s --noproxy '*' -b $cookie_file 127.0.0.1:8080/budgets/new | grep '<title>' | sed 's/<title>//g' | sed 's/<\/title>//g')"
    echo "Categories: $(curl -s --noproxy '*' -b $cookie_file 127.0.0.1:8080/categories | grep '<title>' | sed 's/<title>//g' | sed 's/<\/title>//g')"
    echo "Transactions: $(curl -s --noproxy '*' -b $cookie_file 127.0.0.1:8080/transactions/new | grep '<title>' | sed 's/<title>//g' | sed 's/<\/title>//g')"
    
    rm -f $cookie_file
}

# Использование
test_pages
```

## История создания
- **Дата создания:** 2025-08-31
- **Контекст:** Создан во время исправления проблем интерфейса
- **Причина:** Первый тестовый пользователь `test@example.com` уже существовал в базе

## Примечания
- Пользователь имеет роль Admin в своей семье
- Может использоваться для тестирования всех функций приложения
- База данных: MongoDB (family_budget_local)
- Окружение: Local development (localhost:8080)
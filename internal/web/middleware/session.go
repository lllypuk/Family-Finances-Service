package middleware

import (
	"encoding/gob"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/domain/user"
)

const (
	SessionName     = "family-budget-session"
	SessionUserKey  = "user_id"
	SessionRoleKey  = "role"
	SessionEmailKey = "email"
	SessionTimeout  = 24 * time.Hour
)

// SessionData представляет данные, хранящиеся в сессии
type SessionData struct {
	UserID    uuid.UUID `json:"user_id"`
	Role      user.Role `json:"role"`
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SessionStore настраивает хранилище сессий.
//
// cookieSecure выставляет флаг Secure на cookie сессии. По умолчанию он равен
// «сборка в production», но управляется переменной COOKIE_SECURE: браузер
// выбрасывает Secure-cookie на любом http:// origin, и без такого рычага
// развёртывание без TLS (docker-compose.minimal.yml, первый запуск по IP)
// зацикливалось бы на входе.
func SessionStore(secretKey string, cookieSecure bool) echo.MiddlewareFunc {
	// Регистрируем типы для сессий
	gob.Register(uuid.UUID{})
	gob.Register(user.Role(""))

	store := sessions.NewCookieStore([]byte(secretKey))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   int(SessionTimeout.Seconds()),
		HttpOnly: true,
		Secure:   cookieSecure, // COOKIE_SECURE; по умолчанию true в production
		SameSite: http.SameSiteLaxMode,
	}
	// NewCookieStore штампует кодеки своим MaxAge по умолчанию (30 дней), и
	// присвоение Options меняет только атрибут cookie. Без этого вызова
	// перехваченная cookie оставалась бы валидной на сервере 30 дней вместо
	// SessionTimeout.
	store.MaxAge(store.Options.MaxAge)

	return session.Middleware(store)
}

// GetSessionData извлекает данные из сессии
func GetSessionData(c echo.Context) (*SessionData, error) {
	// Поддержка моков для тестирования
	if mockSessionData := c.Get("mock_session_data"); mockSessionData != nil {
		if mockData, ok := mockSessionData.(*SessionData); ok {
			return mockData, nil
		}
	}

	sess, err := session.Get(SessionName, c)
	if err != nil {
		return nil, err
	}

	// Проверяем, есть ли данные в сессии
	userID, ok := sess.Values[SessionUserKey]
	if !ok {
		return nil, echo.ErrUnauthorized
	}

	role, ok := sess.Values[SessionRoleKey]
	if !ok {
		return nil, echo.ErrUnauthorized
	}

	email, ok := sess.Values[SessionEmailKey]
	if !ok {
		return nil, echo.ErrUnauthorized
	}

	// Проверяем типы
	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		return nil, echo.ErrUnauthorized
	}

	userRole, ok := role.(user.Role)
	if !ok {
		return nil, echo.ErrUnauthorized
	}

	userEmail, ok := email.(string)
	if !ok {
		return nil, echo.ErrUnauthorized
	}

	return &SessionData{
		UserID:    userUUID,
		Role:      userRole,
		Email:     userEmail,
		ExpiresAt: time.Now().Add(SessionTimeout),
	}, nil
}

// SetSessionData сохраняет данные в сессии
func SetSessionData(c echo.Context, data *SessionData) error {
	sess, err := sessionOrNew(c)
	if sess == nil {
		return err
	}

	// ClearSession выставляет MaxAge = -1, а в рамках одного запроса session.Get
	// возвращает тот же объект сессии. Без восстановления MaxAge перевыпуск
	// сессии при входе (ClearSession -> SetSessionData) отдал бы клиенту cookie
	// на удаление, то есть вход через UI перестал бы работать.
	if sess.Options != nil {
		sess.Options.MaxAge = int(SessionTimeout.Seconds())
	}

	sess.Values[SessionUserKey] = data.UserID
	sess.Values[SessionRoleKey] = data.Role
	sess.Values[SessionEmailKey] = data.Email

	return sess.Save(c.Request(), c.Response())
}

// RotateSession перевыпускает сессию под нового пользователя: все прежние
// значения (включая CSRF-токен анонимной сессии) отбрасываются, записываются
// данные пользователя и свежий CSRF-токен, cookie сохраняется один раз.
//
// Это защита от фиксации сессии (S-02): хранилище cookie-based, серверного
// session ID нет, поэтому «новой» сессию делает именно полная замена значений
// вместе с токеном. Вызывать на каждой точке входа в аутентифицированное
// состояние — вход по паролю и регистрация по приглашению.
func RotateSession(c echo.Context, data *SessionData) error {
	sess, err := sessionOrNew(c)
	if sess == nil {
		return err
	}

	for k := range sess.Values {
		delete(sess.Values, k)
	}

	if sess.Options != nil {
		sess.Options.MaxAge = int(SessionTimeout.Seconds())
	}

	token, err := generateCSRFToken()
	if err != nil {
		return err
	}

	sess.Values[SessionUserKey] = data.UserID
	sess.Values[SessionRoleKey] = data.Role
	sess.Values[SessionEmailKey] = data.Email
	sess.Values[CSRFTokenKey] = token

	return sess.Save(c.Request(), c.Response())
}

// ClearSession очищает сессию
func ClearSession(c echo.Context) error {
	sess, err := sessionOrNew(c)
	if sess == nil {
		return err
	}

	// Очищаем все значения
	for k := range sess.Values {
		delete(sess.Values, k)
	}

	// Устанавливаем MaxAge в -1 для удаления cookie
	sess.Options.MaxAge = -1

	return sess.Save(c.Request(), c.Response())
}

// IsAuthenticated проверяет, аутентифицирован ли пользователь
func IsAuthenticated(c echo.Context) bool {
	// Поддержка моков для тестирования
	if mockAuth := c.Get("mock_is_authenticated"); mockAuth != nil {
		if mockValue, ok := mockAuth.(bool); ok {
			return mockValue
		}
		return false
	}

	_, err := GetSessionData(c)
	return err == nil
}

// sessionOrNew возвращает сессию запроса.
//
// Нечитаемая cookie (чужой или провёрнутый SESSION_SECRET) не мешает записать
// сессию заново: gorilla отдаёт вместе с ошибкой пригодную новую сессию, и
// последующий sess.Save перезаписывает cookie. Поэтому вызывающий код
// проверяет именно `sess == nil`, а не `err != nil`. Тот же хелпер используют
// csrf.go и session.go, чтобы это правило было записано ровно один раз.
func sessionOrNew(c echo.Context) (*sessions.Session, error) {
	return session.Get(SessionName, c)
}

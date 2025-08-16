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
	SessionName      = "family-budget-session"
	SessionUserKey   = "user_id"
	SessionFamilyKey = "family_id"
	SessionRoleKey   = "role"
	SessionEmailKey  = "email"
	SessionTimeout   = 24 * time.Hour
)

// SessionData представляет данные, хранящиеся в сессии
type SessionData struct {
	UserID    uuid.UUID `json:"user_id"`
	FamilyID  uuid.UUID `json:"family_id"`
	Role      user.Role `json:"role"`
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SessionStore настраивает хранилище сессий
func SessionStore(secretKey string, isProduction bool) echo.MiddlewareFunc {
	// Регистрируем типы для сессий
	gob.Register(uuid.UUID{})
	gob.Register(user.Role(""))

	store := sessions.NewCookieStore([]byte(secretKey))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   int(SessionTimeout.Seconds()),
		HttpOnly: true,
		Secure:   isProduction, // true для production с HTTPS, false для разработки
		SameSite: http.SameSiteLaxMode,
	}

	return session.Middleware(store)
}

// GetSessionData извлекает данные из сессии
func GetSessionData(c echo.Context) (*SessionData, error) {
	sess, err := session.Get(SessionName, c)
	if err != nil {
		return nil, err
	}

	// Проверяем, есть ли данные в сессии
	userID, ok := sess.Values[SessionUserKey]
	if !ok {
		return nil, echo.ErrUnauthorized
	}

	familyID, ok := sess.Values[SessionFamilyKey]
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

	familyUUID, ok := familyID.(uuid.UUID)
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
		FamilyID:  familyUUID,
		Role:      userRole,
		Email:     userEmail,
		ExpiresAt: time.Now().Add(SessionTimeout),
	}, nil
}

// SetSessionData сохраняет данные в сессии
func SetSessionData(c echo.Context, data *SessionData) error {
	sess, err := session.Get(SessionName, c)
	if err != nil {
		return err
	}

	sess.Values[SessionUserKey] = data.UserID
	sess.Values[SessionFamilyKey] = data.FamilyID
	sess.Values[SessionRoleKey] = data.Role
	sess.Values[SessionEmailKey] = data.Email

	return sess.Save(c.Request(), c.Response())
}

// ClearSession очищает сессию
func ClearSession(c echo.Context) error {
	sess, err := session.Get(SessionName, c)
	if err != nil {
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
	_, err := GetSessionData(c)
	return err == nil
}

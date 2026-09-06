package user

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrNotFound — пользователя нет в хранилище.
//
// Репозиторий оборачивает его через %w, а вызывающие проверяют через errors.Is:
// только так «строки нет» (401 на логине, 404 в API) отличается от сбоя
// инфраструктуры (таймаут контекста, SQLITE_BUSY → 500).
var ErrNotFound = errors.New("user not found")

// ErrFamilyExists — семья уже создана; единственность держит UNIQUE families.singleton.
var ErrFamilyExists = errors.New("family already exists")

// ErrLastAdmin — запись оставила бы семью без активного администратора.
// Проверяется в транзакции репозитория: два параллельных PATCH иначе понижали бы друг друга.
var ErrLastAdmin = errors.New("last active admin")

type User struct {
	ID        uuid.UUID `json:"id"         bson:"_id"`
	Email     string    `json:"email"      bson:"email"`
	Password  string    `json:"-"          bson:"password"` // Скрыт из JSON
	FirstName string    `json:"first_name" bson:"first_name"`
	LastName  string    `json:"last_name"  bson:"last_name"`
	Role      Role      `json:"role"       bson:"role"`
	// IsActive — false запрещает вход и отзывает сессии; записи пользователя остаются (A-04).
	IsActive  bool      `json:"is_active"  bson:"is_active"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

type Role string

const (
	RoleAdmin  Role = "admin"  // Главный пользователь семьи
	RoleMember Role = "member" // Обычный член семьи
)

// IsValid сообщает, что роль — одна из известных.
func (r Role) IsValid() bool {
	return r == RoleAdmin || r == RoleMember
}

type Family struct {
	ID       uuid.UUID `json:"id"       bson:"_id"`
	Name     string    `json:"name"     bson:"name"`
	Currency string    `json:"currency" bson:"currency"` // USD, RUB, EUR и т.д.
	// Timezone — IANA-зона семьи: по ней считаются границы периодов («текущий месяц»).
	Timezone  string    `json:"timezone"   bson:"timezone"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

func NewUser(email, firstName, lastName string, role Role) *User {
	return &User{
		ID:        uuid.New(),
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		Role:      role,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func NewFamily(name, currency, timezone string) *Family {
	return &Family{
		ID:        uuid.New(),
		Name:      name,
		Currency:  currency,
		Timezone:  timezone,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Location возвращает зону семьи; при пустой или неизвестной зоне — UTC,
// чтобы отчёты строились даже на записи, созданной до появления колонки.
func (f *Family) Location() *time.Location {
	loc, err := time.LoadLocation(f.Timezone)
	if err != nil {
		return time.UTC
	}

	return loc
}

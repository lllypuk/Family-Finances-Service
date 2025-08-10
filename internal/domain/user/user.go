package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id" bson:"_id"`
	Email     string    `json:"email" bson:"email"`
	Password  string    `json:"-" bson:"password"` // Скрыт из JSON
	FirstName string    `json:"first_name" bson:"first_name"`
	LastName  string    `json:"last_name" bson:"last_name"`
	Role      Role      `json:"role" bson:"role"`
	FamilyID  uuid.UUID `json:"family_id" bson:"family_id"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

type Role string

const (
	RoleAdmin  Role = "admin"  // Главный пользователь семьи
	RoleMember Role = "member" // Обычный член семьи
	RoleChild  Role = "child"  // Ребенок с ограниченными правами
)

type Family struct {
	ID        uuid.UUID `json:"id" bson:"_id"`
	Name      string    `json:"name" bson:"name"`
	Currency  string    `json:"currency" bson:"currency"` // USD, RUB, EUR и т.д.
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

func NewUser(email, firstName, lastName string, familyID uuid.UUID, role Role) *User {
	return &User{
		ID:        uuid.New(),
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		Role:      role,
		FamilyID:  familyID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func NewFamily(name, currency string) *Family {
	return &Family{
		ID:        uuid.New(),
		Name:      name,
		Currency:  currency,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

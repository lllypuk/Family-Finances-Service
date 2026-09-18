// Package account — платёжные счета семьи, по которым сверяются расходы.
package account

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// MaxNameLength — предел имени в символах.
const MaxNameLength = 50

var (
	ErrNotFound = errors.New("account not found")
	// ErrNameExists — имя занято другим счётом семьи, архивным в том числе.
	ErrNameExists = errors.New("account with this name already exists")
	// ErrInUse — на счёт ссылаются операции или сверки.
	ErrInUse     = errors.New("account is referenced by transactions or reconciliations")
	ErrNameEmpty = errors.New("account name is empty")
	ErrNameLong  = errors.New("account name is too long")
)

type Account struct {
	ID         uuid.UUID
	Name       string
	IsArchived bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// NormalizeName обрезает пробелы по краям и проверяет длину.
func NormalizeName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	switch {
	case trimmed == "":
		return "", ErrNameEmpty
	case utf8.RuneCountInString(trimmed) > MaxNameLength:
		return "", ErrNameLong
	}

	return trimmed, nil
}

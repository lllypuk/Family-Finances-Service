// Package holding — активы и пассивы семьи, из снимков стоимости которых складывается капитал.
package holding

import (
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
)

// MaxNameLength — предел имени в символах.
const MaxNameLength = 50

var (
	ErrNotFound = errors.New("holding not found")
	// ErrNameExists — имя занято другой позицией семьи, архивной в том числе.
	ErrNameExists  = errors.New("holding with this name already exists")
	ErrNameEmpty   = errors.New("holding name is empty")
	ErrNameLong    = errors.New("holding name is too long")
	ErrInvalidSide = errors.New("side must be asset or liability")
	ErrInvalidKind = errors.New("kind is not allowed for this side")
)

// Side задаёт знак вклада позиции в капитал и после создания не меняется.
type Side string

const (
	SideAsset     Side = "asset"
	SideLiability Side = "liability"
)

type Kind string

const (
	KindCash       Kind = "cash"
	KindDeposit    Kind = "deposit"
	KindInvestment Kind = "investment"
	KindProperty   Kind = "property"
	KindVehicle    Kind = "vehicle"
	KindMortgage   Kind = "mortgage"
	KindLoan       Kind = "loan"
	KindCreditCard Kind = "credit_card"
	KindOther      Kind = "other"
)

// Value — снимок стоимости позиции на дату.
type Value struct {
	Date       date.Date
	ValueMinor money.Minor
}

type Holding struct {
	ID         uuid.UUID
	Name       string
	Side       Side
	Kind       Kind
	IsArchived bool
	// Current — последний снимок не позже сегодняшнего дня семьи; nil, если снимков нет.
	Current   *Value
	CreatedAt time.Time
	UpdatedAt time.Time
}

func ValidSide(side Side) bool {
	return side == SideAsset || side == SideLiability
}

// Kinds — допустимые виды стороны; в SQL этого списка нет.
func Kinds(side Side) []Kind {
	switch side {
	case SideAsset:
		return []Kind{KindCash, KindDeposit, KindInvestment, KindProperty, KindVehicle, KindOther}
	case SideLiability:
		return []Kind{KindMortgage, KindLoan, KindCreditCard, KindOther}
	default:
		return nil
	}
}

func ValidKind(side Side, kind Kind) bool {
	return slices.Contains(Kinds(side), kind)
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

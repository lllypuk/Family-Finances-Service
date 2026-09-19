// Package holding — активы и пассивы семьи, из снимков стоимости которых складывается капитал.
package holding

import (
	"errors"
	"fmt"
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

	ErrValueNotFound   = errors.New("holding value not found")
	ErrValueOutOfRange = errors.New("value is out of range")
	ErrValueDateFuture = errors.New("value date is later than today in the family timezone")
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
	// UpdatedAt заполняют только запись и история; у Current он нулевой.
	UpdatedAt time.Time
}

// SeriesRow — снимок позиции со стороной, из которого складывается ряд капитала.
type SeriesRow struct {
	HoldingID  uuid.UUID
	Side       Side
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
	Plan      Plan
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Поля плана — json-имена: PlanError.Field уходит клиенту как есть.
const (
	FieldMonthlyIncome  = "monthly_income_minor"
	FieldMonthlyExpense = "monthly_expense_minor"
)

// Plan — плановые средние поступления и выплаты позиции в месяц; 0/0 — плана нет.
type Plan struct {
	MonthlyIncomeMinor  money.Minor
	MonthlyExpenseMinor money.Minor
	// UpdatedAt — время последней правки чисел плана; nil без плана.
	UpdatedAt *time.Time
}

func (p Plan) IsZero() bool {
	return p.MonthlyIncomeMinor == 0 && p.MonthlyExpenseMinor == 0
}

// PlanError — число плана вне 0 … MaxAmount.
type PlanError struct {
	Field string
}

func (e *PlanError) Error() string {
	return fmt.Sprintf("%s must be between 0 and %d", e.Field, money.MaxAmount)
}

func CheckPlan(income, expense money.Minor) error {
	if !ValidValue(income) {
		return &PlanError{Field: FieldMonthlyIncome}
	}
	if !ValidValue(expense) {
		return &PlanError{Field: FieldMonthlyExpense}
	}

	return nil
}

func ValidSide(side Side) bool {
	return side == SideAsset || side == SideLiability
}

// kinds — допустимые виды стороны; в SQL этого списка нет.
func kinds(side Side) []Kind {
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
	return slices.Contains(kinds(side), kind)
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

// ValidValue — 0 законен: так закрывают проданное и погашенное.
func ValidValue(v money.Minor) bool {
	return v >= 0 && v <= money.MaxAmount
}

// CheckValueDate отказывает снимку позже today — дня семьи, который считает вызывающий.
func CheckValueDate(d, today date.Date) error {
	if d.After(today) {
		return ErrValueDateFuture
	}

	return nil
}

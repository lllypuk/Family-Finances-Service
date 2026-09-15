package budget

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
)

// ErrNameExists — нарушение idx_budgets_name_period_active: живой бюджет с таким именем
// на этот период уже есть. Индекс частичный, мягко удалённые бюджеты имя не держат.
var ErrNameExists = errors.New("budget with this name already exists for this period")

// ErrIDExists — нарушение PRIMARY KEY budgets.id. Живой бюджет с этим id вернул бы 200
// по идемпотентности (A-07), поэтому сюда доходит только id мягко удалённого бюджета:
// его строка занимает ключ, но клиенту не видна.
var ErrIDExists = errors.New("budget with this id already exists")

// ErrRecurringCustom — у периода custom нет длины, следующий период вычислить не из чего.
var ErrRecurringCustom = errors.New("custom period cannot be recurring")

// ErrRecurringNotAligned — даты повторяющегося бюджета не совпадают с календарным периодом.
var ErrRecurringNotAligned = errors.New("recurring budget dates must match the calendar period")

// ErrSeriesDatesFixed — даты члена серии неизменны: перенос прошлого инстанса за хвост
// сломал бы поиск хвоста по MAX(start_date).
var ErrSeriesDatesFixed = errors.New("dates of a recurring series member cannot be changed")

// ErrNotTail — операция требует хвоста серии, а хвост уже другой (серия продвинулась
// между чтением и записью) или его нет вовсе.
var ErrNotTail = errors.New("budget is not the tail of its recurring series")

// ErrTooFarBehind — серия отстала больше чем на предел одного прохода: это ошибка в датах,
// а не пропущенные месяцы, и достраивать её молча нельзя.
var ErrTooFarBehind = errors.New("recurring series is too far behind")

// ErrOverlap — живой бюджет уже занимает эти даты (та же область или то же имя).
var ErrOverlap = errors.New("budget period overlaps with existing budget")

// OverlapError называет конфликтующий бюджет: клиенту нужно знать, что переименовать
// или подвинуть. errors.Is(err, ErrOverlap) на нём работает.
type OverlapError struct {
	Name string
}

func (e *OverlapError) Error() string {
	return fmt.Sprintf("%s: overlaps with budget %q", ErrOverlap, e.Name)
}

func (e *OverlapError) Unwrap() error {
	return ErrOverlap
}

// UpdateExpect — то, что сервис знал о серии на момент чтения; репозиторий перепроверяет
// это внутри транзакции записи, иначе между чтением и записью серию можно возобновить.
type UpdateExpect struct {
	Recurring  bool // значение флага при чтении; расхождение означает продвинувшийся хвост
	StopSeries bool // клиент прислал recurring: false
}

type Budget struct {
	ID          uuid.UUID   `json:"id"`
	Name        string      `json:"name"`
	AmountMinor money.Minor `json:"amount_minor"` // Лимит бюджета
	SpentMinor  money.Minor `json:"spent_minor"`  // Потрачено
	Period      Period      `json:"period"`
	CategoryID  *uuid.UUID  `json:"category_id"` // Для конкретной категории
	StartDate   date.Date   `json:"start_date"`
	EndDate     date.Date   `json:"end_date"`
	IsActive    bool        `json:"is_active"`
	Recurring   bool        `json:"recurring"` // Хвост серии: сервер сам создаёт следующий инстанс
	SeriesID    *uuid.UUID  `json:"series_id"` // id первого инстанса серии; nil у обычного бюджета
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

const (
	daysInWeek      = 7
	lastYearlyMonth = time.December
	lastYearlyDay   = 31
)

type Period string

const (
	PeriodWeekly  Period = "weekly"
	PeriodMonthly Period = "monthly"
	PeriodYearly  Period = "yearly"
	PeriodCustom  Period = "custom"
)

func (b *Budget) GetRemainingAmount() money.Minor {
	return b.AmountMinor - b.SpentMinor
}

// GetSpentPercentage возвращает долю потраченного в процентах; при нулевом лимите — 0.
func (b *Budget) GetSpentPercentage() float64 {
	return b.SpentMinor.Percent(b.AmountMinor)
}

// GetSpentShare — та же доля, что GetSpentPercentage, но в единице 0..1, в которой
// доли уходят в API.
func (b *Budget) GetSpentShare() float64 {
	return b.SpentMinor.Share(b.AmountMinor)
}

func (b *Budget) IsOverBudget() bool {
	return b.SpentMinor > b.AmountMinor
}

func (b *Budget) UpdateSpent(amountMinor money.Minor) {
	b.SpentMinor += amountMinor
	b.UpdatedAt = time.Now()
}

// ValidateRecurring проверяет, что даты повторяющегося бюджета совпадают с календарным
// периодом: серия шагает по календарю, а не по дате создания.
func ValidateRecurring(period Period, start, end date.Date) error {
	const weekDays = 6

	switch period {
	case PeriodCustom:
		return ErrRecurringCustom
	case PeriodWeekly:
		if date.DaysBetween(start, end) != weekDays {
			return ErrRecurringNotAligned
		}
	case PeriodMonthly:
		first, last := start.MonthBounds()
		if start != first || end != last {
			return ErrRecurringNotAligned
		}
	case PeriodYearly:
		if start != (date.Date{Year: start.Year, Month: time.January, Day: 1}) ||
			end != (date.Date{Year: start.Year, Month: lastYearlyMonth, Day: lastYearlyDay}) {
			return ErrRecurringNotAligned
		}
	default:
		return ErrRecurringNotAligned
	}

	return nil
}

// Next — следующий инстанс серии: та же область и лимит, даты следующего календарного
// периода, свой id и нулевой расход. Для custom возвращает nil.
func (b *Budget) Next() *Budget {
	var start, end date.Date

	switch b.Period {
	case PeriodWeekly:
		start, end = b.StartDate.AddDays(daysInWeek), b.EndDate.AddDays(daysInWeek)
	case PeriodMonthly:
		start, end = b.StartDate.AddMonths(1).MonthBounds()
	case PeriodYearly:
		start = date.Date{Year: b.StartDate.Year + 1, Month: time.January, Day: 1}
		end = date.Date{Year: start.Year, Month: lastYearlyMonth, Day: lastYearlyDay}
	case PeriodCustom:
		return nil
	default:
		return nil
	}

	next := *b
	next.ID = uuid.New()
	next.StartDate = start
	next.EndDate = end
	next.SpentMinor = 0
	next.Recurring = true
	next.IsActive = true

	return &next
}

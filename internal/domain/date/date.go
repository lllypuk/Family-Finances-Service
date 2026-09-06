// Package date хранит календарную дату без времени и зоны: то, что видит пользователь в операции.
package date

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"time"
)

// Layout — единственный формат даты на проводе и в БД.
const Layout = "2006-01-02"

// ErrInvalidDate возвращается при разборе строки, не являющейся календарной датой.
var ErrInvalidDate = errors.New("invalid date")

// Date — календарная дата. Сравнима оператором ==, нулевое значение невалидно.
//
//nolint:recvcheck // Scan и UnmarshalJSON обязаны быть на указателе, остальное — значение
type Date struct {
	Year  int
	Month time.Month
	Day   int
}

// New нормализует дату: New(2026, 13, 1) — это 2027-01-01.
func New(year int, month time.Month, day int) Date {
	return FromTime(time.Date(year, month, day, 0, 0, 0, 0, time.UTC))
}

// FromTime берёт календарную часть момента в его собственной зоне.
func FromTime(t time.Time) Date {
	year, month, day := t.Date()

	return Date{Year: year, Month: month, Day: day}
}

// Today возвращает сегодняшнюю дату в указанной зоне.
func Today(loc *time.Location) Date {
	return FromTime(time.Now().In(loc))
}

// Parse разбирает строку "YYYY-MM-DD"; несуществующие даты вроде 2026-02-30 отвергаются.
func Parse(s string) (Date, error) {
	t, err := time.Parse(Layout, s)
	if err != nil {
		return Date{}, fmt.Errorf("%w: %q", ErrInvalidDate, s)
	}

	return FromTime(t), nil
}

func (d Date) String() string {
	return d.In(time.UTC).Format(Layout)
}

// IsZero сообщает, что дата не заполнена.
func (d Date) IsZero() bool {
	return d == Date{}
}

// In возвращает полночь этой даты в указанной зоне.
func (d Date) In(loc *time.Location) time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, loc)
}

// AddDays сдвигает дату на n дней с переносом через границы месяцев и лет.
func (d Date) AddDays(n int) Date {
	return FromTime(d.In(time.UTC).AddDate(0, 0, n))
}

func (d Date) Before(other Date) bool {
	return d.compare(other) < 0
}

func (d Date) After(other Date) bool {
	return d.compare(other) > 0
}

// MonthBounds возвращает первый и последний день месяца, в который попадает дата.
func (d Date) MonthBounds() (Date, Date) {
	first := Date{Year: d.Year, Month: d.Month, Day: 1}
	last := New(d.Year, d.Month+1, 1).AddDays(-1)

	return first, last
}

// Value пишет дату в БД строкой; незаполненная дата даёт текст, который не пройдёт CHECK колонки.
func (d Date) Value() (driver.Value, error) {
	return d.String(), nil
}

// Scan читает колонку TEXT 'YYYY-MM-DD'; time.Time принимается на случай драйвера с parseTime.
func (d *Date) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*d = Date{}
	case string:
		return d.parseInto(v)
	case []byte:
		return d.parseInto(string(v))
	case time.Time:
		*d = FromTime(v)
	default:
		return fmt.Errorf("%w: unsupported source type %T", ErrInvalidDate, src)
	}

	return nil
}

// MarshalJSON пишет дату строкой; незаполненная дата — null: её String() дал бы
// "-0001-11-30", который не разбирается обратно.
func (d Date) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil
	}

	return []byte(`"` + d.String() + `"`), nil
}

func (d *Date) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*d = Date{}

		return nil
	}

	const quoted = 2
	if len(data) < quoted || data[0] != '"' || data[len(data)-1] != '"' {
		return fmt.Errorf("%w: expected string, got %s", ErrInvalidDate, data)
	}

	return d.parseInto(string(data[1 : len(data)-1]))
}

func (d Date) compare(other Date) int {
	switch {
	case d.Year != other.Year:
		return d.Year - other.Year
	case d.Month != other.Month:
		return int(d.Month - other.Month)
	default:
		return d.Day - other.Day
	}
}

func (d *Date) parseInto(s string) error {
	parsed, err := Parse(s)
	if err != nil {
		return err
	}

	*d = parsed

	return nil
}

package date_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/date"
)

func TestParse_Valid(t *testing.T) {
	d, err := date.Parse("2026-09-04")
	require.NoError(t, err)
	assert.Equal(t, date.Date{Year: 2026, Month: time.September, Day: 4}, d)
	assert.Equal(t, "2026-09-04", d.String())
}

func TestParse_Invalid(t *testing.T) {
	tests := []string{
		"",
		"2026-13-01",
		"2026-02-30",
		"2026-1-1",
		"04.09.2026",
		"2026-09-04T00:00:00Z",
		"not a date",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := date.Parse(input)
			require.ErrorIs(t, err, date.ErrInvalidDate)
		})
	}
}

func TestParse_LeapDay(t *testing.T) {
	_, err := date.Parse("2024-02-29")
	require.NoError(t, err)

	_, err = date.Parse("2025-02-29")
	require.ErrorIs(t, err, date.ErrInvalidDate)
}

func TestDate_New_Normalizes(t *testing.T) {
	assert.Equal(t, date.Date{Year: 2027, Month: time.January, Day: 1}, date.New(2026, 13, 1))
	assert.Equal(t, date.Date{Year: 2026, Month: time.March, Day: 1}, date.New(2026, time.February, 29))
}

func TestDate_In(t *testing.T) {
	moscow, err := time.LoadLocation("Europe/Moscow")
	require.NoError(t, err)

	midnight := date.New(2026, time.September, 4).In(moscow)
	assert.Equal(t, "2026-09-04T00:00:00+03:00", midnight.Format(time.RFC3339))
	assert.Equal(t, date.New(2026, time.September, 4), date.FromTime(midnight))
}

func TestDate_IsZero(t *testing.T) {
	assert.True(t, date.Date{}.IsZero())
	assert.False(t, date.New(2026, time.January, 1).IsZero())
}

func TestDate_BeforeAfter(t *testing.T) {
	earlier := date.New(2025, time.December, 31)
	later := date.New(2026, time.January, 1)
	sameMonth := date.New(2026, time.January, 2)
	sameYear := date.New(2026, time.February, 1)

	assert.True(t, earlier.Before(later))
	assert.True(t, later.After(earlier))
	assert.True(t, later.Before(sameMonth))
	// Разные месяцы одного года: день меньше, но дата позже.
	assert.True(t, sameMonth.Before(sameYear))
	assert.True(t, sameYear.After(sameMonth))
	assert.False(t, later.Before(later))
	assert.False(t, later.After(later))
}

func TestDate_AddDays(t *testing.T) {
	assert.Equal(t, date.New(2026, time.January, 1), date.New(2025, time.December, 31).AddDays(1))
	assert.Equal(t, date.New(2024, time.February, 29), date.New(2024, time.March, 1).AddDays(-1))
}

func TestDaysBetween(t *testing.T) {
	assert.Equal(t, 30, date.DaysBetween(date.New(2026, time.January, 1), date.New(2026, time.January, 31)))
	assert.Equal(t, -30, date.DaysBetween(date.New(2026, time.January, 31), date.New(2026, time.January, 1)))
	assert.Equal(t, 0, date.DaysBetween(date.New(2026, time.March, 5), date.New(2026, time.March, 5)))
	// Переход на летнее время не должен съедать сутки: счёт идёт в UTC.
	assert.Equal(t, 366, date.DaysBetween(date.New(2024, time.January, 1), date.New(2025, time.January, 1)))
}

func TestDate_MonthBounds(t *testing.T) {
	tests := []struct {
		name  string
		day   date.Date
		first date.Date
		last  date.Date
	}{
		{
			name:  "December crosses the year",
			day:   date.New(2026, time.December, 17),
			first: date.New(2026, time.December, 1),
			last:  date.New(2026, time.December, 31),
		},
		{
			name:  "January",
			day:   date.New(2026, time.January, 1),
			first: date.New(2026, time.January, 1),
			last:  date.New(2026, time.January, 31),
		},
		{
			name:  "Leap February",
			day:   date.New(2024, time.February, 10),
			first: date.New(2024, time.February, 1),
			last:  date.New(2024, time.February, 29),
		},
		{
			name:  "Non-leap February",
			day:   date.New(2025, time.February, 10),
			first: date.New(2025, time.February, 1),
			last:  date.New(2025, time.February, 28),
		},
		{
			name:  "30-day month",
			day:   date.New(2026, time.April, 30),
			first: date.New(2026, time.April, 1),
			last:  date.New(2026, time.April, 30),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first, last := tt.day.MonthBounds()
			assert.Equal(t, tt.first, first)
			assert.Equal(t, tt.last, last)
		})
	}
}

func TestToday(t *testing.T) {
	moscow, err := time.LoadLocation("Europe/Moscow")
	require.NoError(t, err)

	assert.Equal(t, date.FromTime(time.Now().In(moscow)), date.Today(moscow))
}

func TestDate_Value(t *testing.T) {
	value, err := date.New(2026, time.September, 4).Value()
	require.NoError(t, err)
	assert.Equal(t, "2026-09-04", value)
}

func TestDate_Scan(t *testing.T) {
	expected := date.New(2026, time.September, 4)

	tests := []struct {
		name string
		src  any
		want date.Date
	}{
		{name: "string", src: "2026-09-04", want: expected},
		{name: "bytes", src: []byte("2026-09-04"), want: expected},
		{name: "time", src: time.Date(2026, time.September, 4, 23, 59, 0, 0, time.UTC), want: expected},
		{name: "nil", src: nil, want: date.Date{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d date.Date
			require.NoError(t, d.Scan(tt.src))
			assert.Equal(t, tt.want, d)
		})
	}
}

func TestDate_Scan_Invalid(t *testing.T) {
	var d date.Date
	require.ErrorIs(t, d.Scan("2026-13-01"), date.ErrInvalidDate)
	require.ErrorIs(t, d.Scan(42), date.ErrInvalidDate)
}

func TestDate_JSON(t *testing.T) {
	payload := struct {
		Date date.Date `json:"date"`
	}{Date: date.New(2026, time.September, 4)}

	encoded, err := json.Marshal(payload)
	require.NoError(t, err)
	assert.JSONEq(t, `{"date":"2026-09-04"}`, string(encoded))

	var decoded struct {
		Date date.Date `json:"date"`
	}
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	assert.Equal(t, payload.Date, decoded.Date)
}

func TestDate_UnmarshalJSON_Invalid(t *testing.T) {
	var decoded struct {
		Date date.Date `json:"date"`
	}

	require.Error(t, json.Unmarshal([]byte(`{"date":"2026-13-01"}`), &decoded))
	require.Error(t, json.Unmarshal([]byte(`{"date":20260904}`), &decoded))
}

func TestDate_JSON_Zero(t *testing.T) {
	payload := struct {
		Date date.Date `json:"date"`
	}{}

	encoded, err := json.Marshal(payload)
	require.NoError(t, err)
	assert.JSONEq(t, `{"date":null}`, string(encoded))

	decoded := struct {
		Date date.Date `json:"date"`
	}{Date: date.New(2026, time.September, 4)}
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	assert.True(t, decoded.Date.IsZero())
}

// TestDate_JSONField — имя поля берётся из ошибки разбора, иначе клиенту подсветят чужое поле.
func TestDate_JSONField(t *testing.T) {
	var decoded struct {
		StartDate date.Date `json:"start_date"`
	}

	err := json.Unmarshal([]byte(`{"start_date":"2026-13-01"}`), &decoded)
	require.Error(t, err)

	field, ok := date.JSONField(err)
	require.True(t, ok)
	assert.Equal(t, "start_date", field)

	_, ok = date.JSONField(errors.New("unrelated"))
	assert.False(t, ok)
}

// TestDate_Value_Zero — незаполненная дата не должна доезжать до CHECK колонки.
func TestDate_Value_Zero(t *testing.T) {
	_, err := date.Date{}.Value()
	require.ErrorIs(t, err, date.ErrInvalidDate)

	value, err := date.New(2026, time.September, 4).Value()
	require.NoError(t, err)
	assert.Equal(t, "2026-09-04", value)
}

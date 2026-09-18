package services

import "family-budget-service/internal/domain/date"

// statsPeriod подставляет отсутствующие границы ряда независимо друг от друга: from — первое число
// месяца monthlyDefaultMonths-1 месяцев назад, to — today.
func statsPeriod(today date.Date, from, to *date.Date) (date.Date, date.Date, error) {
	monthStart, _ := today.MonthBounds()
	start, end := monthStart.AddMonths(-(monthlyDefaultMonths - 1)), today
	if from != nil {
		start = *from
	}
	if to != nil {
		end = *to
	}
	if end.Before(start) {
		return date.Date{}, date.Date{}, ErrInvalidStatsPeriod
	}
	if monthsBetween(start, end) > monthlyMaxMonths {
		return date.Date{}, date.Date{}, ErrStatsPeriodTooLong
	}

	return start, end, nil
}

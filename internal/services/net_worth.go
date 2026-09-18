package services

import (
	"github.com/google/uuid"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/holding"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/services/dto"
)

// foldNetWorth раскладывает снимки по корзинам месяцев [from, to]: корзина — состояние на
// min(конец месяца, to), снимок заменяет значение позиции. rows обязаны идти по возрастанию даты.
func foldNetWorth(rows []holding.SeriesRow, from, to date.Date) []dto.NetWorthMonth {
	months := make([]dto.NetWorthMonth, 0, monthlyDefaultMonths)
	current := make(map[uuid.UUID]holding.SeriesRow)
	next := 0

	last := date.Date{Year: to.Year, Month: to.Month, Day: 1}
	for m := (date.Date{Year: from.Year, Month: from.Month, Day: 1}); !m.After(last); m = m.AddMonths(1) {
		_, cutoff := m.MonthBounds()
		if cutoff.After(to) {
			cutoff = to
		}
		for ; next < len(rows) && !rows[next].Date.After(cutoff); next++ {
			current[rows[next].HoldingID] = rows[next]
		}

		var assets, liabilities money.Minor
		for _, row := range current {
			if row.Side == holding.SideLiability {
				liabilities += row.ValueMinor
			} else {
				assets += row.ValueMinor
			}
		}
		months = append(months, dto.NetWorthMonth{
			Month:            m.MonthKey(),
			AssetsMinor:      assets,
			LiabilitiesMinor: liabilities,
			NetMinor:         assets - liabilities,
		})
	}

	return months
}

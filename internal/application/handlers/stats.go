package handlers

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/services"
)

// StatsHandler отдаёт агрегаты дашборда через API.
type StatsHandler struct {
	statsService services.StatsService
}

func NewStatsHandler(statsService services.StatsService) *StatsHandler {
	return &StatsHandler{statsService: statsService}
}

// GetSummary отдаёт сводку за период [from, to]; без параметров — текущий месяц
// по часовому поясу семьи, границы считает сервис.
func (h *StatsHandler) GetSummary(c echo.Context) error {
	from, to, detail := parseStatsPeriod(c)
	if detail != nil {
		return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError,
			ErrMessageValidationFailed, *detail)
	}

	summary, err := h.statsService.Summary(c.Request().Context(), from, to)
	if err != nil {
		if errors.Is(err, services.ErrInvalidStatsPeriod) {
			return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
				ErrorDetail{Field: "from", Message: "must not be after to", Code: ErrCodeInvalidQueryParam})
		}
		return respondError(c, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}

	return respondAPI(c, http.StatusOK, summary)
}

// parseStatsPeriod разбирает from/to как календарные даты; отсутствующая граница — nil,
// её подставляет сервис по зоне семьи. Обе границы включительные.
func parseStatsPeriod(c echo.Context) (*date.Date, *date.Date, *ErrorDetail) {
	from, detail := parseStatsDate(c, "from")
	if detail != nil {
		return nil, nil, detail
	}

	to, detail := parseStatsDate(c, "to")
	if detail != nil {
		return nil, nil, detail
	}

	return from, to, nil
}

func parseStatsDate(c echo.Context, param string) (*date.Date, *ErrorDetail) {
	raw := c.QueryParam(param)
	if raw == "" {
		return nil, nil
	}

	parsed, err := date.Parse(raw)
	if err != nil {
		return nil, &ErrorDetail{
			Field: param, Message: "must be a date in YYYY-MM-DD format", Code: ErrCodeInvalidQueryParam,
		}
	}

	return &parsed, nil
}

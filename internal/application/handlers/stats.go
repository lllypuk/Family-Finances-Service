package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"family-budget-service/internal/services"
)

const (
	// statsDateLayout — формат from/to в query: календарная дата без времени.
	statsDateLayout = "2006-01-02"

	lastHourOfDay      = 23
	lastMinuteOfHour   = 59
	lastSecondOfMinute = 59
)

// StatsHandler отдаёт агрегаты дашборда через API.
type StatsHandler struct {
	statsService services.StatsService
}

func NewStatsHandler(statsService services.StatsService) *StatsHandler {
	return &StatsHandler{statsService: statsService}
}

// GetSummary отдаёт сводку за период [from, to]; без параметров — текущий месяц.
func (h *StatsHandler) GetSummary(c echo.Context) error {
	from, to, detail := parseStatsPeriod(c)
	if detail != nil {
		return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError,
			ErrMessageValidationFailed, *detail)
	}

	if h.statsService == nil {
		return respondError(c, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
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

// parseStatsPeriod разбирает from/to. Границы включительные: to расширяется до конца суток,
// иначе операции сегодняшнего дня выпадают из выборки.
func parseStatsPeriod(c echo.Context) (time.Time, time.Time, *ErrorDetail) {
	now := time.Now()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	to := endOfDay(now)

	if raw := c.QueryParam("from"); raw != "" {
		parsed, err := time.ParseInLocation(statsDateLayout, raw, now.Location())
		if err != nil {
			return time.Time{}, time.Time{}, &ErrorDetail{
				Field: "from", Message: "must be a date in YYYY-MM-DD format", Code: ErrCodeInvalidQueryParam,
			}
		}
		from = parsed
	}

	if raw := c.QueryParam("to"); raw != "" {
		parsed, err := time.ParseInLocation(statsDateLayout, raw, now.Location())
		if err != nil {
			return time.Time{}, time.Time{}, &ErrorDetail{
				Field: "to", Message: "must be a date in YYYY-MM-DD format", Code: ErrCodeInvalidQueryParam,
			}
		}
		to = endOfDay(parsed)
	}

	return from, to, nil
}

func endOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), lastHourOfDay, lastMinuteOfHour, lastSecondOfMinute, 0, t.Location())
}

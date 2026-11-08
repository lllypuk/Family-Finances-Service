package handlers

import (
	"errors"
	"net/http"

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
func (h *StatsHandler) GetSummary(w http.ResponseWriter, r *http.Request) error {
	from, to, detail := parseStatsPeriod(r)
	if detail != nil {
		return respondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError,
			ErrMessageValidationFailed, *detail)
	}

	summary, err := h.statsService.Summary(r.Context(), from, to)
	if err != nil {
		return respondStatsError(w, err)
	}

	return respondAPI(w, http.StatusOK, summary)
}

// GetMonthly отдаёт ряд по месяцам периода [from, to]; без параметров — двенадцать календарных
// месяцев по сегодняшний в часовом поясе семьи, границы считает сервис.
func (h *StatsHandler) GetMonthly(w http.ResponseWriter, r *http.Request) error {
	from, to, detail := parseStatsPeriod(r)
	if detail != nil {
		return respondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError,
			ErrMessageValidationFailed, *detail)
	}

	monthly, err := h.statsService.Monthly(r.Context(), from, to)
	if err != nil {
		return respondStatsError(w, err)
	}

	return respondAPI(w, http.StatusOK, monthly)
}

// GetNetWorth отдаёт ряд капитала за [from, to]; границы по умолчанию — как у GetMonthly.
func (h *StatsHandler) GetNetWorth(w http.ResponseWriter, r *http.Request) error {
	from, to, detail := parseStatsPeriod(r)
	if detail != nil {
		return respondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError,
			ErrMessageValidationFailed, *detail)
	}

	series, err := h.statsService.NetWorth(r.Context(), from, to)
	if err != nil {
		return respondStatsError(w, err)
	}

	return respondAPI(w, http.StatusOK, series)
}

// respondStatsError: перевёрнутый и слишком длинный период — ошибки поля from, будущий конец — поля to,
// остальное — 500.
func respondStatsError(w http.ResponseWriter, err error) error {
	switch {
	case errors.Is(err, services.ErrInvalidStatsPeriod):
		return respondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			ErrorDetail{Field: "from", Message: "must not be after to", Code: ErrCodeInvalidQueryParam})
	case errors.Is(err, services.ErrStatsPeriodTooLong):
		return respondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			ErrorDetail{Field: "from", Message: "period must not exceed 120 months", Code: ErrCodeInvalidQueryParam})
	case errors.Is(err, services.ErrStatsPeriodInFuture):
		return respondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			ErrorDetail{Field: "to", Message: "must not be later than today", Code: ErrCodeInvalidQueryParam})
	}

	return respondError(w, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
}

// parseStatsPeriod разбирает from/to как календарные даты; отсутствующая граница — nil,
// её подставляет сервис по зоне семьи. Обе границы включительные.
func parseStatsPeriod(r *http.Request) (*date.Date, *date.Date, *ErrorDetail) {
	from, detail := parseStatsDate(r, "from")
	if detail != nil {
		return nil, nil, detail
	}

	to, detail := parseStatsDate(r, "to")
	if detail != nil {
		return nil, nil, detail
	}

	return from, to, nil
}

func parseStatsDate(r *http.Request, param string) (*date.Date, *ErrorDetail) {
	raw := r.URL.Query().Get(param)
	if raw == "" {
		return nil, nil
	}

	parsed, err := date.Parse(raw)
	if err != nil {
		return nil, &ErrorDetail{
			Field: param, Message: msgInvalidDate, Code: ErrCodeInvalidQueryParam,
		}
	}

	return &parsed, nil
}

package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/infrastructure/validation"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

type FamilyHandler struct {
	familyService services.FamilyService
	validator     *validator.Validate
}

func NewFamilyHandler(familyService services.FamilyService) *FamilyHandler {
	return &FamilyHandler{
		familyService: familyService,
		validator:     newAPIValidator(),
	}
}

// GetFamily отдаёт единственную семью; доступно любой роли.
func (h *FamilyHandler) GetFamily(w http.ResponseWriter, r *http.Request) error {
	foundFamily, err := h.familyService.GetFamily(r.Context())
	if err != nil {
		return h.handleServiceError(w, err)
	}

	return respondAPI(w, http.StatusOK, toFamilyResponse(foundFamily))
}

// UpdateFamily меняет имя и валюту семьи; только admin.
func (h *FamilyHandler) UpdateFamily(w http.ResponseWriter, r *http.Request) error {
	var req UpdateFamilyRequest
	if bindErr := decodeJSON(r, &req); bindErr != nil {
		return respondBindError(w, bindErr)
	}

	// Репозиторий сохраняет уже подрезанное имя, поэтому длину проверяем по нему же.
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		req.Name = &trimmed
	}

	if validationErr := h.validator.Struct(&req); validationErr != nil {
		return respondValidationErrors(w, validationErr)
	}
	if req.Currency != nil {
		if currencyErr := validation.ValidateCurrency(*req.Currency); currencyErr != nil {
			return RespondError(
				w,
				http.StatusUnprocessableEntity,
				ErrCodeValidationError,
				ErrMessageValidationFailed,
				ErrorDetail{Field: fieldCurrency, Message: currencyErr.Error(), Code: ErrCodeValidationError},
			)
		}
	}
	if req.Name == nil && req.Currency == nil && req.Timezone == nil {
		return RespondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			bodyDetail(ErrCodeValidationError, "at least one field must be provided"))
	}

	updatedFamily, err := h.familyService.UpdateFamily(r.Context(), dto.UpdateFamilyDTO{
		Name:     req.Name,
		Currency: req.Currency,
		Timezone: req.Timezone,
	})
	if err != nil {
		return h.handleServiceError(w, err)
	}

	return respondAPI(w, http.StatusOK, toFamilyResponse(updatedFamily))
}

func (h *FamilyHandler) handleServiceError(w http.ResponseWriter, err error) error {
	switch {
	case errors.Is(err, services.ErrFamilyNotFound):
		return RespondError(w, http.StatusNotFound, ErrCodeFamilyNotFound, ErrMessageFamilyNotFound)
	case errors.Is(err, services.ErrCurrencyLocked):
		return RespondError(w, http.StatusConflict, ErrCodeCurrencyLocked, ErrMessageCurrencyLocked)
	case errors.Is(err, services.ErrValidationFailed):
		return RespondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			bodyDetail(ErrCodeValidationError, err.Error()))
	default:
		return RespondError(w, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}
}

func toFamilyResponse(f *user.Family) FamilyResponse {
	return FamilyResponse{
		ID:        f.ID,
		Name:      f.Name,
		Currency:  f.Currency,
		Timezone:  f.Timezone,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}
}

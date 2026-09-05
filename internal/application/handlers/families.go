package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"

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
func (h *FamilyHandler) GetFamily(c echo.Context) error {
	foundFamily, err := h.familyService.GetFamily(c.Request().Context())
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return respondAPI(c, http.StatusOK, toFamilyResponse(foundFamily))
}

// UpdateFamily меняет имя и валюту семьи; только admin.
func (h *FamilyHandler) UpdateFamily(c echo.Context) error {
	var req UpdateFamilyRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, ErrMessageInvalidRequest,
			bodyDetail(ErrCodeInvalidRequest, bindErr.Error()))
	}

	// Репозиторий сохраняет уже подрезанное имя, поэтому длину проверяем по нему же.
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		req.Name = &trimmed
	}

	if validationErr := h.validator.Struct(&req); validationErr != nil {
		return respondValidationErrors(c, validationErr)
	}
	if req.Currency != nil {
		if currencyErr := validation.ValidateCurrency(*req.Currency); currencyErr != nil {
			return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
				ErrorDetail{Field: fieldCurrency, Message: currencyErr.Error(), Code: ErrCodeValidationError})
		}
	}
	if req.Name == nil && req.Currency == nil {
		return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			bodyDetail(ErrCodeValidationError, "at least one field must be provided"))
	}

	updatedFamily, err := h.familyService.UpdateFamily(c.Request().Context(), dto.UpdateFamilyDTO{
		Name:     req.Name,
		Currency: req.Currency,
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return respondAPI(c, http.StatusOK, toFamilyResponse(updatedFamily))
}

func (h *FamilyHandler) handleServiceError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, services.ErrFamilyNotFound):
		return respondError(c, http.StatusNotFound, ErrCodeFamilyNotFound, ErrMessageFamilyNotFound)
	case errors.Is(err, services.ErrCurrencyLocked):
		return respondError(c, http.StatusConflict, ErrCodeCurrencyLocked, ErrMessageCurrencyLocked)
	case errors.Is(err, services.ErrValidationFailed):
		return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			bodyDetail(ErrCodeValidationError, err.Error()))
	default:
		return respondError(c, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}
}

func toFamilyResponse(f *user.Family) FamilyResponse {
	return FamilyResponse{
		ID:        f.ID,
		Name:      f.Name,
		Currency:  f.Currency,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}
}

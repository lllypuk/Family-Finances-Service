package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/domain/account"
	"family-budget-service/internal/services"
)

const fieldName = "name"

type AccountHandler struct {
	accounts  services.AccountService
	validator *validator.Validate
}

func NewAccountHandler(accounts services.AccountService) *AccountHandler {
	return &AccountHandler{accounts: accounts, validator: newAPIValidator()}
}

// ListAccounts — активные счета; ?archived=true добавляет архивные.
func (h *AccountHandler) ListAccounts(c echo.Context) error {
	page, pageErr := parsePagination(c)
	if pageErr != nil {
		return ignoreWritten(pageErr)
	}

	includeArchived := false
	if raw := c.QueryParam("archived"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return ignoreWritten(writeInvalidQueryParam(c, "archived", raw, "must be true or false"))
		}
		includeArchived = parsed
	}

	all, err := h.accounts.List(c.Request().Context(), includeArchived)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch accounts")
	}

	items := make([]AccountResponse, 0, page.Limit)
	for _, a := range pageSlice(all, page) {
		items = append(items, toAccountResponse(a))
	}

	return respondList(c, items, page, len(all))
}

func (h *AccountHandler) CreateAccount(c echo.Context) error {
	var req CreateAccountRequest
	if err := c.Bind(&req); err != nil {
		return respondBindError(c, err)
	}
	if err := h.validator.Struct(req); err != nil {
		return respondValidationErrors(c, err)
	}

	if handled, err := respondClientID(c, req.ID, h.findAccount, toAccountResponse); handled {
		return err
	}

	created, err := h.accounts.Create(c.Request().Context(), req.ID, req.Name)
	if err != nil {
		return respondAccountError(c, err)
	}

	return respondAPI(c, http.StatusCreated, toAccountResponse(created))
}

func (h *AccountHandler) findAccount(c echo.Context, id uuid.UUID) (*account.Account, bool) {
	found, err := h.accounts.GetByID(c.Request().Context(), id)

	return found, err == nil
}

func (h *AccountHandler) UpdateAccount(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidAccountID)
	}

	var req UpdateAccountRequest
	if err = c.Bind(&req); err != nil {
		return respondBindError(c, err)
	}
	if err = h.validator.Struct(req); err != nil {
		return respondValidationErrors(c, err)
	}
	if req.Name == nil && req.IsArchived == nil {
		return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			bodyDetail(ErrCodeValidationError, ErrMessageNoFields))
	}

	updated, err := h.accounts.Update(c.Request().Context(), id, req.Name, req.IsArchived)
	if err != nil {
		return respondAccountError(c, err)
	}

	return respondAPI(c, http.StatusOK, toAccountResponse(updated))
}

func (h *AccountHandler) DeleteAccount(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidAccountID)
	}

	if err = h.accounts.Delete(c.Request().Context(), id); err != nil {
		return respondAccountError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

func respondAccountError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, account.ErrNotFound):
		return respondError(c, http.StatusNotFound, ErrCodeAccountNotFound, ErrMessageAccountNotFound)
	case errors.Is(err, account.ErrNameExists):
		return respondError(c, http.StatusConflict, ErrCodeAccountNameExists, ErrMessageAccountNameExists)
	case errors.Is(err, account.ErrInUse):
		return respondError(c, http.StatusConflict, ErrCodeAccountInUse, ErrMessageAccountInUse)
	case errors.Is(err, account.ErrNameEmpty), errors.Is(err, account.ErrNameLong):
		return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			ErrorDetail{Field: fieldName, Message: err.Error(), Code: ErrCodeValidationError})
	default:
		return respondError(c, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}
}

func toAccountResponse(a *account.Account) AccountResponse {
	return AccountResponse{
		ID:         a.ID,
		Name:       a.Name,
		IsArchived: a.IsArchived,
		CreatedAt:  a.CreatedAt,
		UpdatedAt:  a.UpdatedAt,
	}
}

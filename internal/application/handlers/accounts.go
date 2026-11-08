package handlers

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

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
func (h *AccountHandler) ListAccounts(w http.ResponseWriter, r *http.Request) error {
	page, pageErr := parsePagination(w, r)
	if pageErr != nil {
		return ignoreWritten(pageErr)
	}

	includeArchived, archivedErr := parseArchivedParam(w, r)
	if archivedErr != nil {
		return ignoreWritten(archivedErr)
	}

	all, err := h.accounts.List(r.Context(), includeArchived)
	if err != nil {
		return respondError(w, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch accounts")
	}

	items := make([]AccountResponse, 0, page.Limit)
	for _, a := range pageSlice(all, page) {
		items = append(items, toAccountResponse(a))
	}

	return respondList(w, items, page, len(all))
}

func (h *AccountHandler) CreateAccount(w http.ResponseWriter, r *http.Request) error {
	var req CreateAccountRequest
	if err := decodeJSON(r, &req); err != nil {
		return respondBindError(w, err)
	}
	if err := h.validator.Struct(req); err != nil {
		return respondValidationErrors(w, err)
	}

	if handled, err := respondClientID(w, r, req.ID, h.findAccount, toAccountResponse); handled {
		return err
	}

	created, err := h.accounts.Create(r.Context(), req.ID, req.Name)
	if err != nil {
		return respondAccountError(w, err)
	}

	return respondAPI(w, http.StatusCreated, toAccountResponse(created))
}

func (h *AccountHandler) findAccount(r *http.Request, id uuid.UUID) (*account.Account, bool) {
	found, err := h.accounts.GetByID(r.Context(), id)

	return found, err == nil
}

func (h *AccountHandler) UpdateAccount(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return respondError(w, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidAccountID)
	}

	var req UpdateAccountRequest
	if err = decodeJSON(r, &req); err != nil {
		return respondBindError(w, err)
	}
	if err = h.validator.Struct(req); err != nil {
		return respondValidationErrors(w, err)
	}
	if req.Name == nil && req.IsArchived == nil {
		return respondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			bodyDetail(ErrCodeValidationError, ErrMessageNoFields))
	}

	updated, err := h.accounts.Update(r.Context(), id, req.Name, req.IsArchived)
	if err != nil {
		return respondAccountError(w, err)
	}

	return respondAPI(w, http.StatusOK, toAccountResponse(updated))
}

func (h *AccountHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return respondError(w, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidAccountID)
	}

	if err = h.accounts.Delete(r.Context(), id); err != nil {
		return respondAccountError(w, err)
	}

	return noContent(w)
}

func respondAccountError(w http.ResponseWriter, err error) error {
	switch {
	case errors.Is(err, account.ErrNotFound):
		return respondError(w, http.StatusNotFound, ErrCodeAccountNotFound, ErrMessageAccountNotFound)
	case errors.Is(err, account.ErrNameExists):
		return respondError(w, http.StatusConflict, ErrCodeAccountNameExists, ErrMessageAccountNameExists)
	case errors.Is(err, account.ErrInUse):
		return respondError(w, http.StatusConflict, ErrCodeAccountInUse, ErrMessageAccountInUse)
	case errors.Is(err, account.ErrNameEmpty), errors.Is(err, account.ErrNameLong):
		return respondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			ErrorDetail{Field: fieldName, Message: err.Error(), Code: ErrCodeValidationError})
	default:
		return respondError(w, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
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

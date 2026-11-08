package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

type CategoryHandler struct {
	categoryService services.CategoryService
	validator       *validator.Validate
}

func NewCategoryHandler(categoryService services.CategoryService) *CategoryHandler {
	return &CategoryHandler{
		categoryService: categoryService,
		validator:       newAPIValidator(),
	}
}

func (h *CategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) error {
	var req CreateCategoryRequest
	if bindErr := decodeJSON(r, &req); bindErr != nil {
		return respondBindError(w, bindErr)
	}

	if validationErr := h.validator.Struct(req); validationErr != nil {
		return respondValidationErrors(w, validationErr)
	}

	if handled, err := respondClientID(w, r, req.ID, h.findCategory, dto.ToCategoryAPIResponse); handled {
		return err
	}

	createDTO := dto.CreateCategoryDTO{
		ID:       req.ID,
		Name:     req.Name,
		Type:     category.Type(req.Type),
		Color:    req.Color,
		Icon:     req.Icon,
		ParentID: req.ParentID,
	}

	newCategory, err := h.categoryService.CreateCategory(r.Context(), createDTO)
	if err != nil {
		return handleCreateCategoryServiceError(w, err)
	}

	return respondAPI(w, http.StatusCreated, dto.ToCategoryAPIResponse(newCategory))
}

// findCategory ищет категорию по клиентскому id; ошибка означает «не найдена».
func (h *CategoryHandler) findCategory(r *http.Request, id uuid.UUID) (*category.Category, bool) {
	found, err := h.categoryService.GetCategoryByID(r.Context(), id)

	return found, err == nil
}

func (h *CategoryHandler) GetCategories(w http.ResponseWriter, r *http.Request) error {
	page, pageErr := parsePagination(w, r)
	if pageErr != nil {
		return ignoreWritten(pageErr)
	}

	typeParam := r.URL.Query().Get("type")

	var typeFilter *category.Type
	if typeParam != "" {
		categoryType := category.Type(typeParam)
		typeFilter = &categoryType
	}

	categories, err := h.categoryService.GetCategories(r.Context(), typeFilter)
	if err != nil {
		return respondError(w, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch categories")
	}

	response := make([]dto.CategoryAPIResponse, 0, page.Limit)
	for _, cat := range pageSlice(categories, page) {
		response = append(response, dto.ToCategoryAPIResponse(cat))
	}

	return respondList(w, response, page, len(categories))
}

func (h *CategoryHandler) GetCategoryByID(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return respondError(w, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidCategoryID)
	}

	foundCategory, err := h.categoryService.GetCategoryByID(r.Context(), id)
	if err != nil {
		return respondError(w, http.StatusNotFound, ErrCodeCategoryNotFound, ErrMessageCategoryNotFound)
	}

	return respondAPI(w, http.StatusOK, dto.ToCategoryAPIResponse(foundCategory))
}

func (h *CategoryHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return respondError(w, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidCategoryID)
	}

	var req UpdateCategoryRequest
	if bindErr := decodeJSON(r, &req); bindErr != nil {
		return respondBindError(w, bindErr)
	}

	if validationErr := h.validator.Struct(req); validationErr != nil {
		return respondValidationErrors(w, validationErr)
	}
	if req.Name == nil && req.Color == nil && req.Icon == nil {
		return respondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			bodyDetail(ErrCodeValidationError, ErrMessageNoFields))
	}

	updateDTO := dto.UpdateCategoryDTO{
		Name:  req.Name,
		Color: req.Color,
		Icon:  req.Icon,
	}

	updatedCategory, err := h.categoryService.UpdateCategory(r.Context(), id, updateDTO)
	if err != nil {
		return handleUpdateCategoryServiceError(w, err)
	}

	return respondAPI(w, http.StatusOK, dto.ToCategoryAPIResponse(updatedCategory))
}

func (h *CategoryHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return respondError(w, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidCategoryID)
	}

	if delErr := h.categoryService.DeleteCategory(r.Context(), id); delErr != nil {
		return handleDeleteCategoryServiceError(w, delErr)
	}

	return noContent(w)
}

func handleCreateCategoryServiceError(w http.ResponseWriter, err error) error {
	return handleCategoryServiceError(w, err, "CREATE_FAILED", "Failed to create category")
}

func handleUpdateCategoryServiceError(w http.ResponseWriter, err error) error {
	return handleCategoryServiceError(w, err, "UPDATE_FAILED", "Failed to update category")
}

func handleDeleteCategoryServiceError(w http.ResponseWriter, err error) error {
	return handleCategoryServiceError(w, err, "DELETE_FAILED", "Failed to delete category")
}

// handleCategoryServiceError переводит ошибки CategoryService в ответ API: некорректный ввод
// (несуществующий родитель, третий уровень иерархии, дубль имени) — 422, а не 500;
// всё остальное — failCode/failMessage вызывающей операции.
func handleCategoryServiceError(w http.ResponseWriter, err error, failCode, failMessage string) error {
	switch {
	case errors.Is(err, services.ErrCategoryNotFound):
		return respondError(w, http.StatusNotFound, ErrCodeCategoryNotFound, ErrMessageCategoryNotFound)
	case errors.Is(err, services.ErrParentCategoryNotFound),
		errors.Is(err, services.ErrParentCategoryWrongType),
		errors.Is(err, services.ErrCategoriesDifferentTypes),
		errors.Is(err, services.ErrMaxHierarchyLevels),
		errors.Is(err, services.ErrCategorySelfParent),
		errors.Is(err, services.ErrCategoryNameExists),
		strings.Contains(err.Error(), "validation failed"):
		return respondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			bodyDetail(ErrCodeValidationError, err.Error()))
	default:
		return respondError(w, http.StatusInternalServerError, failCode, failMessage)
	}
}

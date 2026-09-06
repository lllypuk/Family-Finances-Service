package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

type CategoryHandler struct {
	repositories    *Repositories
	categoryService services.CategoryService
	validator       *validator.Validate
}

func NewCategoryHandler(repositories *Repositories, categoryService services.CategoryService) *CategoryHandler {
	return &CategoryHandler{
		repositories:    repositories,
		categoryService: categoryService,
		validator:       newAPIValidator(),
	}
}

func (h *CategoryHandler) CreateCategory(c echo.Context) error {
	var req CreateCategoryRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return respondBindError(c, bindErr)
	}

	if validationErr := h.validator.Struct(req); validationErr != nil {
		return respondValidationErrors(c, validationErr)
	}

	if handled, err := respondClientID(c, req.ID, h.findCategory, dto.ToCategoryAPIResponse); handled {
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

	newCategory, err := h.categoryService.CreateCategory(c.Request().Context(), createDTO)
	if err != nil {
		return handleCreateCategoryServiceError(c, err)
	}

	return respondAPI(c, http.StatusCreated, dto.ToCategoryAPIResponse(newCategory))
}

// findCategory ищет категорию по клиентскому id; ошибка означает «не найдена».
func (h *CategoryHandler) findCategory(c echo.Context, id uuid.UUID) (*category.Category, bool) {
	found, err := h.categoryService.GetCategoryByID(c.Request().Context(), id)

	return found, err == nil
}

func (h *CategoryHandler) GetCategories(c echo.Context) error {
	page, pageErr := parsePagination(c)
	if pageErr != nil {
		return ignoreWritten(pageErr)
	}

	typeParam := c.QueryParam("type")

	var typeFilter *category.Type
	if typeParam != "" {
		categoryType := category.Type(typeParam)
		typeFilter = &categoryType
	}

	categories, err := h.categoryService.GetCategories(c.Request().Context(), typeFilter)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch categories")
	}

	response := make([]dto.CategoryAPIResponse, 0, page.Limit)
	for _, cat := range pageSlice(categories, page) {
		response = append(response, dto.ToCategoryAPIResponse(cat))
	}

	return respondList(c, response, page, len(categories))
}

func (h *CategoryHandler) GetCategoryByID(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidCategoryID)
	}

	foundCategory, err := h.categoryService.GetCategoryByID(c.Request().Context(), id)
	if err != nil {
		return respondError(c, http.StatusNotFound, ErrCodeCategoryNotFound, ErrMessageCategoryNotFound)
	}

	return respondAPI(c, http.StatusOK, dto.ToCategoryAPIResponse(foundCategory))
}

func (h *CategoryHandler) UpdateCategory(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidCategoryID)
	}

	var req UpdateCategoryRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return respondBindError(c, bindErr)
	}

	if validationErr := h.validator.Struct(req); validationErr != nil {
		return respondValidationErrors(c, validationErr)
	}

	updateDTO := dto.UpdateCategoryDTO{
		Name:  req.Name,
		Color: req.Color,
		Icon:  req.Icon,
	}

	updatedCategory, err := h.categoryService.UpdateCategory(c.Request().Context(), id, updateDTO)
	if err != nil {
		return handleUpdateCategoryServiceError(c, err)
	}

	return respondAPI(c, http.StatusOK, dto.ToCategoryAPIResponse(updatedCategory))
}

func (h *CategoryHandler) DeleteCategory(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidCategoryID)
	}

	if delErr := h.categoryService.DeleteCategory(c.Request().Context(), id); delErr != nil {
		return handleDeleteCategoryServiceError(c, delErr)
	}

	return c.NoContent(http.StatusNoContent)
}

func handleCreateCategoryServiceError(c echo.Context, err error) error {
	return handleCategoryServiceError(c, err, "CREATE_FAILED", "Failed to create category")
}

func handleUpdateCategoryServiceError(c echo.Context, err error) error {
	return handleCategoryServiceError(c, err, "UPDATE_FAILED", "Failed to update category")
}

func handleDeleteCategoryServiceError(c echo.Context, err error) error {
	return handleCategoryServiceError(c, err, "DELETE_FAILED", "Failed to delete category")
}

// handleCategoryServiceError переводит ошибки CategoryService в ответ API: некорректный ввод
// (несуществующий родитель, третий уровень иерархии, дубль имени) — 422, а не 500;
// всё остальное — failCode/failMessage вызывающей операции.
func handleCategoryServiceError(c echo.Context, err error, failCode, failMessage string) error {
	switch {
	case errors.Is(err, services.ErrCategoryNotFound):
		return respondError(c, http.StatusNotFound, ErrCodeCategoryNotFound, ErrMessageCategoryNotFound)
	case errors.Is(err, services.ErrParentCategoryNotFound),
		errors.Is(err, services.ErrParentCategoryWrongType),
		errors.Is(err, services.ErrCategoriesDifferentTypes),
		errors.Is(err, services.ErrMaxHierarchyLevels),
		errors.Is(err, services.ErrCategorySelfParent),
		errors.Is(err, services.ErrCategoryNameExists),
		strings.Contains(err.Error(), "validation failed"):
		return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			bodyDetail(ErrCodeValidationError, err.Error()))
	default:
		return respondError(c, http.StatusInternalServerError, failCode, failMessage)
	}
}

package handlers

import (
	"errors"
	"net/http"

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
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, ErrMessageInvalidRequest,
			bodyDetail(ErrCodeInvalidRequest, bindErr.Error()))
	}

	if validationErr := h.validator.Struct(req); validationErr != nil {
		return respondValidationErrors(c, validationErr)
	}

	createDTO := dto.CreateCategoryDTO{
		Name:     req.Name,
		Type:     category.Type(req.Type),
		Color:    req.Color,
		Icon:     req.Icon,
		ParentID: req.ParentID,
	}

	newCategory, err := h.categoryService.CreateCategory(c.Request().Context(), createDTO)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create category")
	}

	return respondAPI(c, http.StatusCreated, dto.ToCategoryAPIResponse(newCategory))
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
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, ErrMessageInvalidRequest,
			bodyDetail(ErrCodeInvalidRequest, bindErr.Error()))
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
		if errors.Is(err, services.ErrCategoryNotFound) {
			return respondError(c, http.StatusNotFound, ErrCodeCategoryNotFound, ErrMessageCategoryNotFound)
		}

		return respondError(c, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update category")
	}

	return respondAPI(c, http.StatusOK, dto.ToCategoryAPIResponse(updatedCategory))
}

func (h *CategoryHandler) DeleteCategory(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidCategoryID)
	}

	if delErr := h.categoryService.DeleteCategory(c.Request().Context(), id); delErr != nil {
		if errors.Is(delErr, services.ErrCategoryNotFound) {
			return respondError(c, http.StatusNotFound, ErrCodeCategoryNotFound, ErrMessageCategoryNotFound)
		}

		return respondError(c, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete category")
	}

	return c.NoContent(http.StatusNoContent)
}

package handlers

import (
	"errors"
	"net/http"
	"time"

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
		validator:       validator.New(),
	}
}

func (h *CategoryHandler) CreateCategory(c echo.Context) error {
	var req CreateCategoryRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request body",
				Details: bindErr.Error(),
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	if err := h.validator.Struct(req); err != nil {
		var validationErrors []ValidationError
		for _, err := range func() validator.ValidationErrors {
			var target validator.ValidationErrors
			_ = errors.As(err, &target)
			return target
		}() {
			validationErrors = append(validationErrors, ValidationError{
				Field:   err.Field(),
				Message: err.Tag(),
				Code:    "VALIDATION_ERROR",
			})
		}

		return c.JSON(http.StatusBadRequest, APIResponse[any]{
			Data: nil,
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
			Errors: validationErrors,
		})
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		var validationErrors []ValidationError
		var validationErrs validator.ValidationErrors
		if errors.As(err, &validationErrs) {
			for _, err := range validationErrs {
				validationErrors = append(validationErrors, ValidationError{
					Field:   err.Field(),
					Message: err.Tag(),
					Code:    "VALIDATION_ERROR",
				})
			}
		}

		return c.JSON(http.StatusBadRequest, APIResponse[any]{
			Data: nil,
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
			Errors: validationErrors,
		})
	}

	// Convert request to DTO
	createDTO := dto.CreateCategoryDTO{
		Name:     req.Name,
		Type:     category.Type(req.Type),
		Color:    req.Color,
		Icon:     req.Icon,
		ParentID: req.ParentID,
	}

	// Use service to create category
	newCategory, err := h.categoryService.CreateCategory(c.Request().Context(), createDTO)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorDetail{
				Code:    "CREATE_FAILED",
				Message: "Failed to create category",
				Details: err.Error(),
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	// Convert domain model to API response
	response := dto.ToCategoryAPIResponse(newCategory)

	return c.JSON(http.StatusCreated, APIResponse[dto.CategoryAPIResponse]{
		Data: response,
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

func (h *CategoryHandler) GetCategories(c echo.Context) error {
	// Получаем параметры запроса
	typeParam := c.QueryParam("type")

	var typeFilter *category.Type
	if typeParam != "" {
		categoryType := category.Type(typeParam)
		typeFilter = &categoryType
	}

	// Use service to get categories (single-family model)
	categories, err := h.categoryService.GetCategories(c.Request().Context(), typeFilter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorDetail{
				Code:    "FETCH_FAILED",
				Message: "Failed to fetch categories",
				Details: err.Error(),
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	// Convert domain models to API responses
	var response []dto.CategoryAPIResponse
	for _, cat := range categories {
		response = append(response, dto.ToCategoryAPIResponse(cat))
	}

	return c.JSON(http.StatusOK, APIResponse[[]dto.CategoryAPIResponse]{
		Data: response,
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

func (h *CategoryHandler) GetCategoryByID(c echo.Context) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid category ID format",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	foundCategory, err := h.categoryService.GetCategoryByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error: ErrorDetail{
				Code:    "CATEGORY_NOT_FOUND",
				Message: "Category not found",
				Details: err.Error(),
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	// Convert domain model to API response
	response := dto.ToCategoryAPIResponse(foundCategory)

	return c.JSON(http.StatusOK, APIResponse[dto.CategoryAPIResponse]{
		Data: response,
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

func (h *CategoryHandler) UpdateCategory(c echo.Context) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid category ID format",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	var req UpdateCategoryRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request body",
				Details: bindErr.Error(),
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	// Convert request to DTO
	updateDTO := dto.UpdateCategoryDTO{
		Name:  req.Name,
		Color: req.Color,
		Icon:  req.Icon,
	}

	// Use service to update category
	updatedCategory, err := h.categoryService.UpdateCategory(c.Request().Context(), id, updateDTO)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "UPDATE_FAILED"

		if errors.Is(err, services.ErrCategoryNotFound) {
			statusCode = http.StatusNotFound
			errorCode = "CATEGORY_NOT_FOUND"
		}

		return c.JSON(statusCode, ErrorResponse{
			Error: ErrorDetail{
				Code:    errorCode,
				Message: err.Error(),
				Details: err.Error(),
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	// Convert domain model to API response
	response := dto.ToCategoryAPIResponse(updatedCategory)

	return c.JSON(http.StatusOK, APIResponse[dto.CategoryAPIResponse]{
		Data: response,
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

func (h *CategoryHandler) DeleteCategory(c echo.Context) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid category ID format",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	// Use service to delete category (single-family model)
	err = h.categoryService.DeleteCategory(c.Request().Context(), id)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "DELETE_FAILED"

		if errors.Is(err, services.ErrCategoryNotFound) {
			statusCode = http.StatusNotFound
			errorCode = "CATEGORY_NOT_FOUND"
		}

		return c.JSON(statusCode, ErrorResponse{
			Error: ErrorDetail{
				Code:    errorCode,
				Message: err.Error(),
				Details: err.Error(),
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	return c.JSON(http.StatusNoContent, nil)
}

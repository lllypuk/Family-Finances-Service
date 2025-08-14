package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/domain/category"
)

type CategoryHandler struct {
	repositories *Repositories
	validator    *validator.Validate
}

func NewCategoryHandler(repositories *Repositories) *CategoryHandler {
	return &CategoryHandler{
		repositories: repositories,
		validator:    validator.New(),
	}
}

func (h *CategoryHandler) CreateCategory(c echo.Context) error {
	var req CreateCategoryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request body",
				Details: err.Error(),
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

		return c.JSON(http.StatusBadRequest, APIResponse[interface{}]{
			Data: nil,
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
			Errors: validationErrors,
		})
	}

	// Создаем новую категорию
	newCategory := &category.Category{
		ID:        uuid.New(),
		Name:      req.Name,
		Type:      category.Type(req.Type),
		Color:     req.Color,
		Icon:      req.Icon,
		ParentID:  req.ParentID,
		FamilyID:  req.FamilyID,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.repositories.Category.Create(c.Request().Context(), newCategory); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorDetail{
				Code:    "CREATE_FAILED",
				Message: "Failed to create category",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	response := CategoryResponse{
		ID:        newCategory.ID,
		Name:      newCategory.Name,
		Type:      string(newCategory.Type),
		Color:     newCategory.Color,
		Icon:      newCategory.Icon,
		ParentID:  newCategory.ParentID,
		FamilyID:  newCategory.FamilyID,
		IsActive:  newCategory.IsActive,
		CreatedAt: newCategory.CreatedAt,
		UpdatedAt: newCategory.UpdatedAt,
	}

	return c.JSON(http.StatusCreated, APIResponse[CategoryResponse]{
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
	familyIDParam := c.QueryParam("family_id")
	typeParam := c.QueryParam("type")

	if familyIDParam == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "MISSING_FAMILY_ID",
				Message: "family_id query parameter is required",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	familyID, err := uuid.Parse(familyIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_FAMILY_ID",
				Message: "Invalid family ID format",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	var categories []*category.Category

	if typeParam != "" {
		categories, err = h.repositories.Category.GetByType(
			c.Request().Context(),
			familyID,
			category.Type(typeParam),
		)
	} else {
		categories, err = h.repositories.Category.GetByFamilyID(c.Request().Context(), familyID)
	}

	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorDetail{
				Code:    "FETCH_FAILED",
				Message: "Failed to fetch categories",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	var response []CategoryResponse
	for _, cat := range categories {
		response = append(response, CategoryResponse{
			ID:        cat.ID,
			Name:      cat.Name,
			Type:      string(cat.Type),
			Color:     cat.Color,
			Icon:      cat.Icon,
			ParentID:  cat.ParentID,
			FamilyID:  cat.FamilyID,
			IsActive:  cat.IsActive,
			CreatedAt: cat.CreatedAt,
			UpdatedAt: cat.UpdatedAt,
		})
	}

	return c.JSON(http.StatusOK, APIResponse[[]CategoryResponse]{
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

	foundCategory, err := h.repositories.Category.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error: ErrorDetail{
				Code:    "CATEGORY_NOT_FOUND",
				Message: "Category not found",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	response := CategoryResponse{
		ID:        foundCategory.ID,
		Name:      foundCategory.Name,
		Type:      string(foundCategory.Type),
		Color:     foundCategory.Color,
		Icon:      foundCategory.Icon,
		ParentID:  foundCategory.ParentID,
		FamilyID:  foundCategory.FamilyID,
		IsActive:  foundCategory.IsActive,
		CreatedAt: foundCategory.CreatedAt,
		UpdatedAt: foundCategory.UpdatedAt,
	}

	return c.JSON(http.StatusOK, APIResponse[CategoryResponse]{
		Data: response,
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

func (h *CategoryHandler) UpdateCategory(c echo.Context) error {
	helper := NewUpdateEntityHelper(
		UpdateEntityParams[UpdateCategoryRequest, *category.Category, CategoryResponse]{
			Validator: h.validator,
			GetByID: func(c echo.Context, id uuid.UUID) (*category.Category, error) {
				return h.repositories.Category.GetByID(c.Request().Context(), id)
			},
			Update: func(c echo.Context, entity *category.Category) error {
				return h.repositories.Category.Update(c.Request().Context(), entity)
			},
			UpdateFields:  h.updateCategoryFields,
			BuildResponse: h.buildCategoryResponse,
			EntityType:    "category",
		})

	return helper.Execute(c)
}

func (h *CategoryHandler) updateCategoryFields(category *category.Category, req *UpdateCategoryRequest) {
	if req.Name != nil {
		category.Name = *req.Name
	}
	if req.Color != nil {
		category.Color = *req.Color
	}
	if req.Icon != nil {
		category.Icon = *req.Icon
	}
	category.UpdatedAt = time.Now()
}

func (h *CategoryHandler) buildCategoryResponse(cat *category.Category) CategoryResponse {
	return CategoryResponse{
		ID:        cat.ID,
		Name:      cat.Name,
		Type:      string(cat.Type),
		Color:     cat.Color,
		Icon:      cat.Icon,
		ParentID:  cat.ParentID,
		FamilyID:  cat.FamilyID,
		IsActive:  cat.IsActive,
		CreatedAt: cat.CreatedAt,
		UpdatedAt: cat.UpdatedAt,
	}
}

func (h *CategoryHandler) DeleteCategory(c echo.Context) error {
	return DeleteEntityHelper(c, func(id uuid.UUID) error {
		return h.repositories.Category.Delete(c.Request().Context(), id)
	}, "Category")
}

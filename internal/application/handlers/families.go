package handlers

import (
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type FamilyHandler struct {
	repositories *Repositories
	validator    *validator.Validate
}

func NewFamilyHandler(repositories *Repositories) *FamilyHandler {
	return &FamilyHandler{
		repositories: repositories,
		validator:    validator.New(),
	}
}

func (h *FamilyHandler) GetFamily(c echo.Context) error {
	foundFamily, err := h.repositories.Family.Get(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error: ErrorDetail{
				Code:    "FAMILY_NOT_FOUND",
				Message: "Family not found",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	response := FamilyResponse{
		ID:        foundFamily.ID,
		Name:      foundFamily.Name,
		Currency:  foundFamily.Currency,
		CreatedAt: foundFamily.CreatedAt,
		UpdatedAt: foundFamily.UpdatedAt,
	}

	return c.JSON(http.StatusOK, APIResponse[FamilyResponse]{
		Data: response,
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

func (h *FamilyHandler) GetFamilyMembers(c echo.Context) error {
	// Get the single family
	_, err := h.repositories.Family.Get(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error: ErrorDetail{
				Code:    "FAMILY_NOT_FOUND",
				Message: "Family not found",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	members, err := h.repositories.User.GetAll(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorDetail{
				Code:    "FETCH_FAILED",
				Message: "Failed to fetch family members",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	var response []UserResponse
	for _, member := range members {
		response = append(response, UserResponse{
			ID:        member.ID,
			Email:     member.Email,
			FirstName: member.FirstName,
			LastName:  member.LastName,
			Role:      string(member.Role),
			CreatedAt: member.CreatedAt,
			UpdatedAt: member.UpdatedAt,
		})
	}

	return c.JSON(http.StatusOK, APIResponse[[]UserResponse]{
		Data: response,
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
	"family-budget-service/internal/web/middleware"
	"family-budget-service/internal/web/models"
)

// AdminHandler handles admin-only operations
type AdminHandler struct {
	*BaseHandler
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(repos *handlers.Repositories, services *services.Services) *AdminHandler {
	return &AdminHandler{
		BaseHandler: NewBaseHandler(repos, services),
	}
}

// requireAdmin checks if the current user is an admin
func (h *AdminHandler) requireAdmin(c echo.Context) (*user.User, error) {
	session, err := middleware.GetSessionData(c)
	if err != nil {
		return nil, errors.New("unauthorized")
	}

	currentUser, err := h.services.User.GetUserByID(c.Request().Context(), session.UserID)
	if err != nil {
		return nil, errors.New("failed to load user")
	}

	if currentUser.Role != user.RoleAdmin {
		return nil, errors.New("admin access required")
	}

	return currentUser, nil
}

// ListUsers displays the user management page with invites
func (h *AdminHandler) ListUsers(c echo.Context) error {
	currentUser, err := h.requireAdmin(c)
	if err != nil {
		return h.redirectWithError(c, "/", "Admin access required")
	}

	// Get all users
	users, err := h.services.User.GetUsers(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load users")
	}

	// Get the single family
	family, err := h.services.Family.GetFamily(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load family")
	}

	// Get all invites for the family
	invites, err := h.services.Invite.ListFamilyInvites(c.Request().Context(), family.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load invites")
	}

	csrfToken, err := middleware.GetCSRFToken(c)
	if err != nil {
		return err
	}

	data := map[string]any{
		"Title":       "User Management",
		"Users":       users,
		"Invites":     invites,
		"Family":      family,
		"CurrentUser": currentUser,
		"CSRFToken":   csrfToken,
		"Roles":       []string{string(user.RoleAdmin), string(user.RoleMember), string(user.RoleChild)},
	}

	return c.Render(http.StatusOK, "admin/users.html", data)
}

// CreateInvite creates a new invitation
func (h *AdminHandler) CreateInvite(c echo.Context) error {
	currentUser, err := h.requireAdmin(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusForbidden, "Admin access required")
	}

	// Parse form data
	var form models.CreateInviteForm
	if bindErr := c.Bind(&form); bindErr != nil {
		return h.htmxError(c, "Invalid form data")
	}

	// Validate form
	if validateErr := form.Validate(); validateErr != nil {
		return h.htmxError(c, validateErr.Error())
	}

	// Create invite DTO
	createDTO := dto.CreateInviteDTO{
		Email: strings.ToLower(strings.TrimSpace(form.Email)),
		Role:  form.Role,
	}

	// Create invite via service
	invite, err := h.services.Invite.CreateInvite(c.Request().Context(), currentUser.ID, createDTO)
	if err != nil {
		if errors.Is(err, services.ErrEmailAlreadyExists) {
			return h.htmxError(c, "User with this email already exists")
		}
		if strings.Contains(err.Error(), "pending invite already exists") {
			return h.htmxError(c, "Pending invite already exists for this email")
		}
		return h.htmxError(c, "Failed to create invite: "+err.Error())
	}

	// Return HTMX partial with the new invite row
	data := map[string]any{
		"Invite": invite,
	}

	c.Response().Header().Set("Hx-Trigger", "inviteCreated")
	return c.Render(http.StatusOK, "admin/invite_row.html", data)
}

// RevokeInvite revokes an existing invitation
func (h *AdminHandler) RevokeInvite(c echo.Context) error {
	currentUser, err := h.requireAdmin(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusForbidden, "Admin access required")
	}

	// Parse invite ID
	inviteIDStr := c.Param("id")
	inviteID, err := uuid.Parse(inviteIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid invite ID")
	}

	// Revoke invite via service
	if revokeErr := h.services.Invite.RevokeInvite(c.Request().Context(), inviteID, currentUser.ID); revokeErr != nil {
		if errors.Is(revokeErr, services.ErrInviteNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "Invite not found")
		}
		if errors.Is(revokeErr, services.ErrUnauthorized) {
			return echo.NewHTTPError(http.StatusForbidden, "Unauthorized")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to revoke invite: "+revokeErr.Error())
	}

	// Return success for HTMX (empty response with trigger)
	c.Response().Header().Set("Hx-Trigger", "inviteRevoked")
	return c.NoContent(http.StatusOK)
}

// DeleteUser deletes a user from the family
func (h *AdminHandler) DeleteUser(c echo.Context) error {
	currentUser, err := h.requireAdmin(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusForbidden, "Admin access required")
	}

	// Parse user ID
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user ID")
	}

	// Prevent self-deletion
	if userID == currentUser.ID {
		return echo.NewHTTPError(http.StatusBadRequest, "Cannot delete yourself")
	}

	// Get user to delete to verify it exists
	_, err = h.services.User.GetUserByID(c.Request().Context(), userID)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "User not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load user")
	}

	// Single family model - no family check needed

	// Delete user via service
	if deleteErr := h.services.User.DeleteUser(c.Request().Context(), userID); deleteErr != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete user: "+deleteErr.Error())
	}

	// Return success for HTMX (empty response with trigger)
	c.Response().Header().Set("Hx-Trigger", "userDeleted")
	return c.NoContent(http.StatusOK)
}

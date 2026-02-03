package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/web/middleware"
)

// BackupHandler handles backup-related operations
type BackupHandler struct {
	*BaseHandler
}

// NewBackupHandler creates a new BackupHandler
func NewBackupHandler(repos *handlers.Repositories, services *services.Services) *BackupHandler {
	return &BackupHandler{
		BaseHandler: NewBaseHandler(repos, services),
	}
}

// requireAdmin checks if the current user is an admin
func (h *BackupHandler) requireAdmin(c echo.Context) error {
	session, err := middleware.GetSessionData(c)
	if err != nil {
		return errors.New("unauthorized")
	}

	currentUser, err := h.services.User.GetUserByID(c.Request().Context(), session.UserID)
	if err != nil {
		return errors.New("failed to load user")
	}

	if currentUser.Role != user.RoleAdmin {
		return errors.New("admin access required")
	}

	return nil
}

// BackupPage displays the backup management page
func (h *BackupHandler) BackupPage(c echo.Context) error {
	if err := h.requireAdmin(c); err != nil {
		return h.redirectWithError(c, "/", "Admin access required")
	}

	// Get list of backups
	backups, err := h.services.Backup.ListBackups(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load backups")
	}

	// Get CSRF token
	csrfToken, err := middleware.GetCSRFToken(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get CSRF token")
	}

	data := map[string]interface{}{
		"Title":     "Резервные копии",
		"Backups":   backups,
		"CSRFToken": csrfToken,
		"Messages":  h.getFlashMessages(c),
	}

	return c.Render(http.StatusOK, "admin/backup.html", data)
}

// CreateBackup creates a new backup
func (h *BackupHandler) CreateBackup(c echo.Context) error {
	if err := h.requireAdmin(c); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Admin access required")
	}

	// Create backup
	backupInfo, err := h.services.Backup.CreateBackup(c.Request().Context())
	if err != nil {
		if IsHTMXRequest(c) {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create backup")
		}
		return h.redirectWithError(c, "/admin/backup", "Failed to create backup")
	}

	// If HTMX request, return partial
	if IsHTMXRequest(c) {
		data := map[string]interface{}{
			"Backup": backupInfo,
		}
		return c.Render(http.StatusOK, "admin/backup_row.html", data)
	}

	// Otherwise redirect with success message
	return h.redirectWithSuccess(c, "/admin/backup", "Backup created successfully")
}

// DownloadBackup downloads a backup file
func (h *BackupHandler) DownloadBackup(c echo.Context) error {
	if err := h.requireAdmin(c); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Admin access required")
	}

	filename := c.Param("filename")
	if filename == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Filename is required")
	}

	// Validate and get backup info
	_, err := h.services.Backup.GetBackup(c.Request().Context(), filename)
	if err != nil {
		if errors.Is(err, services.ErrBackupNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "Backup not found")
		}
		if errors.Is(err, services.ErrInvalidBackupFilename) {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid filename")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get backup")
	}

	// Get file path
	filePath := h.services.Backup.GetBackupFilePath(filename)
	if filePath == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid filename")
	}

	// Send file for download
	// #nosec G304 -- Filename is validated by backup service safePath() method
	return c.Attachment(filePath, filename)
}

// DeleteBackup deletes a backup file
func (h *BackupHandler) DeleteBackup(c echo.Context) error {
	if err := h.requireAdmin(c); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Admin access required")
	}

	filename := c.Param("filename")
	if filename == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Filename is required")
	}

	// Delete backup
	err := h.services.Backup.DeleteBackup(c.Request().Context(), filename)
	if err != nil {
		if errors.Is(err, services.ErrBackupNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "Backup not found")
		}
		if errors.Is(err, services.ErrInvalidBackupFilename) {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid filename")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete backup")
	}

	// If HTMX request, return empty response (element will be deleted from DOM)
	if IsHTMXRequest(c) {
		return c.NoContent(http.StatusOK)
	}

	// Otherwise redirect with success message
	return h.redirectWithSuccess(c, "/admin/backup", "Backup deleted successfully")
}

// RestoreBackup restores database from a backup file
func (h *BackupHandler) RestoreBackup(c echo.Context) error {
	if err := h.requireAdmin(c); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Admin access required")
	}

	filename := c.Param("filename")
	if filename == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Filename is required")
	}

	// Restore backup
	err := h.services.Backup.RestoreBackup(c.Request().Context(), filename)
	if err != nil {
		if errors.Is(err, services.ErrBackupNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "Backup not found")
		}
		if errors.Is(err, services.ErrInvalidBackupFilename) {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid filename")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to restore backup: %v", err))
	}

	// Return success message
	if IsHTMXRequest(c) {
		data := map[string]interface{}{
			"Message": "База данных восстановлена. Перезапустите приложение.",
			"Type":    "warning",
		}
		return c.Render(http.StatusOK, "components/message.html", data)
	}

	return h.redirectWithSuccess(c, "/admin/backup", "Database restored. Please restart the application.")
}

// IsHTMXRequest checks if the request is from HTMX
func IsHTMXRequest(c echo.Context) bool {
	return c.Request().Header.Get("Hx-Request") == HTMXRequestHeader
}

package handlers

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"family-budget-service/internal/services"
)

// BackupHandler отдаёт управление файлами бэкапа через API; все роуты только для admin.
// Восстановление намеренно не публикуется — только по ssh (A-11).
type BackupHandler struct {
	backupService services.BackupService
}

func NewBackupHandler(backupService services.BackupService) *BackupHandler {
	return &BackupHandler{backupService: backupService}
}

// CreateBackup делает синхронный VACUUM INTO и отдаёт метаданные файла.
func (h *BackupHandler) CreateBackup(c echo.Context) error {
	info, err := h.backupService.CreateBackup(c.Request().Context())
	if err != nil {
		return respondError(c, http.StatusInternalServerError, ErrCodeBackupFailed, ErrMessageBackupFailed)
	}

	return respondAPI(c, http.StatusCreated, toBackupResponse(info))
}

// ListBackups отдаёт файлы каталога бэкапов, свежие первыми.
func (h *BackupHandler) ListBackups(c echo.Context) error {
	backups, err := h.backupService.ListBackups(c.Request().Context())
	if err != nil {
		return respondError(c, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}

	responses := make([]BackupResponse, 0, len(backups))
	for _, info := range backups {
		responses = append(responses, toBackupResponse(info))
	}

	return respondAPI(c, http.StatusOK, responses)
}

// DownloadBackup отдаёт файл вложением; имя проверяет сервис.
func (h *BackupHandler) DownloadBackup(c echo.Context) error {
	name := c.Param("name")

	info, err := h.backupService.GetBackup(c.Request().Context(), name)
	if err != nil {
		return h.handleBackupError(c, err)
	}

	return c.Attachment(h.backupService.GetBackupFilePath(info.Filename), info.Filename)
}

// DeleteBackup удаляет файл бэкапа.
func (h *BackupHandler) DeleteBackup(c echo.Context) error {
	if err := h.backupService.DeleteBackup(c.Request().Context(), c.Param("name")); err != nil {
		return h.handleBackupError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *BackupHandler) handleBackupError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, services.ErrInvalidBackupFilename):
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidBackupName, ErrMessageInvalidBackupName)
	case errors.Is(err, services.ErrBackupNotFound):
		return respondError(c, http.StatusNotFound, ErrCodeBackupNotFound, ErrMessageBackupNotFound)
	default:
		return respondError(c, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}
}

func toBackupResponse(info *services.BackupInfo) BackupResponse {
	return BackupResponse{
		Name:      info.Filename,
		SizeBytes: info.Size,
		CreatedAt: info.CreatedAt,
	}
}

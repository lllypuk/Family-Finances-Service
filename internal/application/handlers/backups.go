package handlers

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"family-budget-service/internal/application/router"
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
func (h *BackupHandler) CreateBackup(w http.ResponseWriter, r *http.Request) error {
	info, err := h.backupService.CreateBackup(r.Context())
	if err != nil {
		return RespondError(w, http.StatusInternalServerError, ErrCodeBackupFailed, ErrMessageBackupFailed)
	}

	return respondAPI(w, http.StatusCreated, toBackupResponse(info))
}

// ListBackups отдаёт файлы каталога бэкапов, свежие первыми.
func (h *BackupHandler) ListBackups(w http.ResponseWriter, r *http.Request) error {
	page, pageErr := parsePagination(w, r)
	if pageErr != nil {
		return ignoreWritten(pageErr)
	}

	backups, err := h.backupService.ListBackups(r.Context())
	if err != nil {
		return RespondError(w, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}

	responses := make([]BackupResponse, 0, page.Limit)
	for _, info := range pageSlice(backups, page) {
		responses = append(responses, toBackupResponse(info))
	}

	return respondList(w, responses, page, len(backups))
}

// DownloadBackup отдаёт файл вложением; имя проверяет сервис.
func (h *BackupHandler) DownloadBackup(w http.ResponseWriter, r *http.Request) error {
	name := r.PathValue("name")

	info, err := h.backupService.GetBackup(r.Context(), name)
	if err != nil {
		return h.handleBackupError(w, err)
	}

	return serveAttachment(w, r, h.backupService.GetBackupFilePath(info.Filename), info.Filename)
}

// serveAttachment отдаёт файл вложением с поддержкой Range; нет файла — 404 роутера.
func serveAttachment(w http.ResponseWriter, r *http.Request, path, name string) error {
	f, err := os.Open(path) //nolint:gosec // G703: путь строит сервис по имени, прошедшему проверку шаблона
	if err != nil {
		return &router.HTTPError{Status: http.StatusNotFound, Err: err}
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil || fi.IsDir() {
		return &router.HTTPError{Status: http.StatusNotFound, Err: err}
	}

	w.Header().
		Set("Content-Disposition", `attachment; filename="`+strings.NewReplacer("\\", "\\\\", `"`, `\"`).Replace(name)+`"`)
	http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)

	return nil
}

// DeleteBackup удаляет файл бэкапа.
func (h *BackupHandler) DeleteBackup(w http.ResponseWriter, r *http.Request) error {
	if err := h.backupService.DeleteBackup(r.Context(), r.PathValue("name")); err != nil {
		return h.handleBackupError(w, err)
	}

	return noContent(w)
}

func (h *BackupHandler) handleBackupError(w http.ResponseWriter, err error) error {
	switch {
	case errors.Is(err, services.ErrInvalidBackupFilename):
		return RespondError(w, http.StatusBadRequest, ErrCodeInvalidBackupName, ErrMessageInvalidBackupName)
	case errors.Is(err, services.ErrBackupNotFound):
		return RespondError(w, http.StatusNotFound, ErrCodeBackupNotFound, ErrMessageBackupNotFound)
	default:
		return RespondError(w, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}
}

func toBackupResponse(info *services.BackupInfo) BackupResponse {
	return BackupResponse{
		Name:      info.Filename,
		SizeBytes: info.Size,
		CreatedAt: info.CreatedAt,
	}
}

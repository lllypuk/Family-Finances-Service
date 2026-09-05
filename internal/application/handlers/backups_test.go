package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/services"
)

type MockBackupService struct {
	mock.Mock
}

func (m *MockBackupService) CreateBackup(ctx context.Context) (*services.BackupInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.BackupInfo), args.Error(1)
}

func (m *MockBackupService) ListBackups(ctx context.Context) ([]*services.BackupInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*services.BackupInfo), args.Error(1)
}

func (m *MockBackupService) GetBackup(ctx context.Context, filename string) (*services.BackupInfo, error) {
	args := m.Called(ctx, filename)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.BackupInfo), args.Error(1)
}

func (m *MockBackupService) DeleteBackup(ctx context.Context, filename string) error {
	return m.Called(ctx, filename).Error(0)
}

func (m *MockBackupService) RestoreBackup(ctx context.Context, filename string) error {
	return m.Called(ctx, filename).Error(0)
}

func (m *MockBackupService) GetBackupFilePath(filename string) string {
	return m.Called(filename).String(0)
}

const testBackupName = "backup_20260904_030000123.db"

func backupContext(method, target, name string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(method, target, nil), rec)
	if name != "" {
		c.SetParamNames("name")
		c.SetParamValues(name)
	}

	return c, rec
}

func TestBackupHandler_CreateBackup_Success(t *testing.T) {
	mockService := &MockBackupService{}
	created := time.Date(2026, 9, 4, 3, 0, 0, 0, time.UTC)
	mockService.On("CreateBackup", mock.Anything).
		Return(&services.BackupInfo{Filename: testBackupName, Size: 4096, CreatedAt: created}, nil)

	c, rec := backupContext(http.MethodPost, "/api/v1/backups", "")
	require.NoError(t, handlers.NewBackupHandler(mockService).CreateBackup(c))

	assert.Equal(t, http.StatusCreated, rec.Code)
	var response handlers.APIResponse[handlers.BackupResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, testBackupName, response.Data.Name)
	assert.Equal(t, int64(4096), response.Data.SizeBytes)
	mockService.AssertExpectations(t)
}

func TestBackupHandler_CreateBackup_ServiceError(t *testing.T) {
	mockService := &MockBackupService{}
	mockService.On("CreateBackup", mock.Anything).Return(nil, assert.AnError)

	c, rec := backupContext(http.MethodPost, "/api/v1/backups", "")
	require.NoError(t, handlers.NewBackupHandler(mockService).CreateBackup(c))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, handlers.ErrCodeBackupFailed, decodeErrorCode(t, rec))
}

func TestBackupHandler_ListBackups_Success(t *testing.T) {
	mockService := &MockBackupService{}
	mockService.On("ListBackups", mock.Anything).Return([]*services.BackupInfo{
		{Filename: testBackupName, Size: 10, CreatedAt: time.Now()},
	}, nil)

	c, rec := backupContext(http.MethodGet, "/api/v1/backups", "")
	require.NoError(t, handlers.NewBackupHandler(mockService).ListBackups(c))

	assert.Equal(t, http.StatusOK, rec.Code)
	var response handlers.APIResponse[[]handlers.BackupResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response.Data, 1)
	assert.Equal(t, testBackupName, response.Data[0].Name)
}

func TestBackupHandler_ListBackups_Empty(t *testing.T) {
	mockService := &MockBackupService{}
	mockService.On("ListBackups", mock.Anything).Return([]*services.BackupInfo{}, nil)

	c, rec := backupContext(http.MethodGet, "/api/v1/backups", "")
	require.NoError(t, handlers.NewBackupHandler(mockService).ListBackups(c))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"data":[]`)
}

func TestBackupHandler_DownloadBackup_Success(t *testing.T) {
	path := filepath.Join(t.TempDir(), testBackupName)
	require.NoError(t, os.WriteFile(path, []byte("sqlite-bytes"), 0o600))

	mockService := &MockBackupService{}
	mockService.On("GetBackup", mock.Anything, testBackupName).
		Return(&services.BackupInfo{Filename: testBackupName, Size: 12, CreatedAt: time.Now()}, nil)
	mockService.On("GetBackupFilePath", testBackupName).Return(path)

	c, rec := backupContext(http.MethodGet, "/api/v1/backups/"+testBackupName+"/download", testBackupName)
	require.NoError(t, handlers.NewBackupHandler(mockService).DownloadBackup(c))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get(echo.HeaderContentDisposition), testBackupName)
	assert.Equal(t, "sqlite-bytes", rec.Body.String())
}

func TestBackupHandler_DownloadBackup_InvalidName(t *testing.T) {
	mockService := &MockBackupService{}
	mockService.On("GetBackup", mock.Anything, "../secrets.db").Return(nil, services.ErrInvalidBackupFilename)

	c, rec := backupContext(http.MethodGet, "/api/v1/backups/x/download", "../secrets.db")
	require.NoError(t, handlers.NewBackupHandler(mockService).DownloadBackup(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, handlers.ErrCodeInvalidBackupName, decodeErrorCode(t, rec))
}

func TestBackupHandler_DownloadBackup_NotFound(t *testing.T) {
	mockService := &MockBackupService{}
	mockService.On("GetBackup", mock.Anything, testBackupName).Return(nil, services.ErrBackupNotFound)

	c, rec := backupContext(http.MethodGet, "/api/v1/backups/x/download", testBackupName)
	require.NoError(t, handlers.NewBackupHandler(mockService).DownloadBackup(c))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, handlers.ErrCodeBackupNotFound, decodeErrorCode(t, rec))
}

func TestBackupHandler_DeleteBackup_Success(t *testing.T) {
	mockService := &MockBackupService{}
	mockService.On("DeleteBackup", mock.Anything, testBackupName).Return(nil)

	c, rec := backupContext(http.MethodDelete, "/api/v1/backups/"+testBackupName, testBackupName)
	require.NoError(t, handlers.NewBackupHandler(mockService).DeleteBackup(c))

	assert.Equal(t, http.StatusNoContent, rec.Code)
	mockService.AssertExpectations(t)
}

func TestBackupHandler_DeleteBackup_InvalidName(t *testing.T) {
	mockService := &MockBackupService{}
	mockService.On("DeleteBackup", mock.Anything, "notes.txt").Return(services.ErrInvalidBackupFilename)

	c, rec := backupContext(http.MethodDelete, "/api/v1/backups/notes.txt", "notes.txt")
	require.NoError(t, handlers.NewBackupHandler(mockService).DeleteBackup(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, handlers.ErrCodeInvalidBackupName, decodeErrorCode(t, rec))
}

func TestBackupHandler_DeleteBackup_NotFound(t *testing.T) {
	mockService := &MockBackupService{}
	mockService.On("DeleteBackup", mock.Anything, testBackupName).Return(services.ErrBackupNotFound)

	c, rec := backupContext(http.MethodDelete, "/api/v1/backups/"+testBackupName, testBackupName)
	require.NoError(t, handlers.NewBackupHandler(mockService).DeleteBackup(c))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, handlers.ErrCodeBackupNotFound, decodeErrorCode(t, rec))
}

func decodeErrorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	return response.Error.Code
}

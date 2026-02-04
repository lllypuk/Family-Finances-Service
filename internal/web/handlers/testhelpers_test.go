package handlers_test

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
	"family-budget-service/internal/web/middleware"
)

// MockRenderer is a mock template renderer
type MockRenderer struct{}

func (r *MockRenderer) Render(_ io.Writer, _ string, _ interface{}, _ echo.Context) error {
	// Simple mock that just returns success
	return nil
}

// newTestContext creates a test echo context with request and response recorder
func newTestContext(method, path string, body string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	e.Renderer = &MockRenderer{}
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

// WithSession adds session data to context
func withSession(c echo.Context, userID uuid.UUID, role user.Role) {
	sessionData := &middleware.SessionData{
		UserID: userID,
		Role:   role,
		Email:  "test@example.com",
	}
	// Set both keys for compatibility with different middleware functions
	c.Set("user", sessionData)              // For GetUserFromContext
	c.Set("mock_session_data", sessionData) // For GetSessionData
	c.Set("csrf_token", "test-csrf-token")  // For GetCSRFToken
}

// WithHTMX marks the request as an HTMX request
func withHTMX(c echo.Context) {
	c.Request().Header.Set("Hx-Request", "true")
}

// MockInviteService is a mock implementation of InviteService
type MockInviteService struct {
	mock.Mock
}

func (m *MockInviteService) CreateInvite(
	ctx context.Context,
	createdByID uuid.UUID,
	createDTO dto.CreateInviteDTO,
) (*user.Invite, error) {
	args := m.Called(ctx, createdByID, createDTO)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.Invite), args.Error(1)
}

func (m *MockInviteService) GetInviteByToken(ctx context.Context, token string) (*user.Invite, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.Invite), args.Error(1)
}

func (m *MockInviteService) AcceptInvite(
	ctx context.Context,
	token string,
	acceptDTO dto.AcceptInviteDTO,
) (*user.User, error) {
	args := m.Called(ctx, token, acceptDTO)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockInviteService) RevokeInvite(ctx context.Context, inviteID uuid.UUID, revokedByID uuid.UUID) error {
	args := m.Called(ctx, inviteID, revokedByID)
	return args.Error(0)
}

func (m *MockInviteService) ListFamilyInvites(ctx context.Context, familyID uuid.UUID) ([]*user.Invite, error) {
	args := m.Called(ctx, familyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*user.Invite), args.Error(1)
}

func (m *MockInviteService) CleanupExpiredInvites(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockInviteService) DeleteExpiredInvites(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockUserService is a mock implementation of UserService
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) GetUserByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) GetUserByEmail(ctx context.Context, email string) (*user.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) GetUsers(ctx context.Context) ([]*user.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*user.User), args.Error(1)
}

func (m *MockUserService) CreateUser(ctx context.Context, createDTO dto.CreateUserDTO) (*user.User, error) {
	args := m.Called(ctx, createDTO)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) UpdateUser(
	ctx context.Context,
	id uuid.UUID,
	updateDTO dto.UpdateUserDTO,
) (*user.User, error) {
	args := m.Called(ctx, id, updateDTO)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserService) AuthenticateUser(ctx context.Context, email, password string) (*user.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) ChangeUserRole(ctx context.Context, userID uuid.UUID, role user.Role) error {
	args := m.Called(ctx, userID, role)
	return args.Error(0)
}

func (m *MockUserService) ValidateUserAccess(ctx context.Context, userID, resourceOwnerID uuid.UUID) error {
	args := m.Called(ctx, userID, resourceOwnerID)
	return args.Error(0)
}

// MockFamilyService is a mock implementation of FamilyService
type MockFamilyService struct {
	mock.Mock
}

func (m *MockFamilyService) GetFamily(ctx context.Context) (*user.Family, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.Family), args.Error(1)
}

func (m *MockFamilyService) UpdateFamily(ctx context.Context, updateDTO dto.UpdateFamilyDTO) (*user.Family, error) {
	args := m.Called(ctx, updateDTO)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.Family), args.Error(1)
}

func (m *MockFamilyService) SetupFamily(ctx context.Context, setupDTO dto.SetupFamilyDTO) (*user.Family, error) {
	args := m.Called(ctx, setupDTO)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.Family), args.Error(1)
}

func (m *MockFamilyService) IsSetupComplete(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Bool(0), args.Error(1)
}

// MockBackupService is a mock implementation of BackupService
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
	args := m.Called(ctx, filename)
	return args.Error(0)
}

func (m *MockBackupService) RestoreBackup(ctx context.Context, filename string) error {
	args := m.Called(ctx, filename)
	return args.Error(0)
}

func (m *MockBackupService) GetBackupFilePath(filename string) string {
	args := m.Called(filename)
	return args.String(0)
}

func (m *MockBackupService) CleanupOldBackups(ctx context.Context, keepCount int) (int, error) {
	args := m.Called(ctx, keepCount)
	return args.Int(0), args.Error(1)
}

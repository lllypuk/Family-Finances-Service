package handlers_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/user"
	webHandlers "family-budget-service/internal/web/handlers"
	"family-budget-service/internal/web/middleware"
)

// MockUserRepositoryWeb is a mock implementation for web auth tests
type MockUserRepositoryWeb struct {
	mock.Mock
}

func (m *MockUserRepositoryWeb) Create(ctx context.Context, user *user.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepositoryWeb) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepositoryWeb) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepositoryWeb) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*user.User, error) {
	args := m.Called(ctx, familyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*user.User), args.Error(1)
}

func (m *MockUserRepositoryWeb) Update(ctx context.Context, user *user.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepositoryWeb) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockFamilyRepositoryWeb is a mock implementation for web auth tests
type MockFamilyRepositoryWeb struct {
	mock.Mock
}

func (m *MockFamilyRepositoryWeb) Create(ctx context.Context, family *user.Family) error {
	args := m.Called(ctx, family)
	return args.Error(0)
}

func (m *MockFamilyRepositoryWeb) GetByID(ctx context.Context, id uuid.UUID) (*user.Family, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.Family), args.Error(1)
}

func (m *MockFamilyRepositoryWeb) Update(ctx context.Context, family *user.Family) error {
	args := m.Called(ctx, family)
	return args.Error(0)
}

func (m *MockFamilyRepositoryWeb) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockTemplateRenderer is a mock implementation of template renderer
type MockTemplateRenderer struct {
	mock.Mock
}

func (m *MockTemplateRenderer) Render(w io.Writer, name string, data any, c echo.Context) error {
	args := m.Called(w, name, data, c)
	return args.Error(0)
}

// setupAuthHandler creates an auth handler with mock dependencies
func setupAuthHandler() (*webHandlers.AuthHandler, *MockUserRepositoryWeb, *MockFamilyRepositoryWeb) {
	mockUserRepo := &MockUserRepositoryWeb{}
	mockFamilyRepo := &MockFamilyRepositoryWeb{}
	repositories := &handlers.Repositories{
		User:   mockUserRepo,
		Family: mockFamilyRepo,
	}
	handler := webHandlers.NewAuthHandler(repositories, nil)
	return handler, mockUserRepo, mockFamilyRepo
}

// setupEchoWithSession creates an Echo instance with session middleware for testing
func setupEchoWithSession() *echo.Echo {
	e := echo.New()

	// Initialize session middleware first
	sessionMiddleware := middleware.SessionStore("test-secret-key-for-testing-that-is-long-enough", false)
	e.Use(sessionMiddleware)

	// Add CSRF middleware mock
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Mock CSRF token
			c.Set("csrf_token", "mock-csrf-token")
			return next(c)
		}
	})

	// Add validator
	e.Validator = &mockValidator{}

	return e
}

// setupEchoForTesting creates a simplified Echo instance for testing without actual session middleware
func setupEchoForTesting() *echo.Echo {
	e := echo.New()

	// Try to setup session middleware properly
	sessionMiddleware := middleware.SessionStore("test-secret-key-for-testing-that-is-very-long-enough", false)
	e.Use(sessionMiddleware)

	// Add validator
	e.Validator = &mockValidator{}

	return e
}

func TestAuthHandler_LoginPage_Success(t *testing.T) {
	handler, _, _ := setupAuthHandler()
	e := setupEchoForTesting()

	// Setup mock renderer
	mockRenderer := &MockTemplateRenderer{}
	e.Renderer = mockRenderer
	mockRenderer.On("Render", mock.Anything, "login.html", mock.MatchedBy(func(data any) bool {
		dataMap := data.(map[string]any)
		return dataMap["Title"] == "Sign In" && dataMap["CSRFToken"] != nil
	}), mock.Anything).Return(nil)

	// Add handler as a route to ensure middleware runs
	e.GET("/login", handler.LoginPage)

	// Act - make actual HTTP request through the middleware pipeline
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)
	mockRenderer.AssertExpectations(t)
}

func TestAuthHandler_Login_Success(t *testing.T) {
	handler, mockUserRepo, _ := setupAuthHandler()
	e := setupEchoWithSession()

	// Arrange
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	expectedUser := &user.User{
		ID:       uuid.New(),
		Email:    "test@example.com",
		Password: string(hashedPassword),
		FamilyID: uuid.New(),
		Role:     user.RoleMember,
	}

	mockUserRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(expectedUser, nil)

	// Add handler as a route to ensure middleware runs
	e.POST("/login", handler.Login)

	// Prepare form data
	form := url.Values{}
	form.Add("email", "test@example.com")
	form.Add("password", "password123")

	// Act - make actual HTTP request through the middleware pipeline
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/", rec.Header().Get("Location"))
	mockUserRepo.AssertExpectations(t)
}

func TestAuthHandler_Login_HTMXRequest_Success(t *testing.T) {
	handler, mockUserRepo, _ := setupAuthHandler()
	e := setupEchoWithSession()

	// Arrange
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	expectedUser := &user.User{
		ID:       uuid.New(),
		Email:    "test@example.com",
		Password: string(hashedPassword),
		FamilyID: uuid.New(),
		Role:     user.RoleMember,
	}

	mockUserRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(expectedUser, nil)

	// Add handler as a route to ensure middleware runs
	e.POST("/login", handler.Login)

	// Prepare form data
	form := url.Values{}
	form.Add("email", "test@example.com")
	form.Add("password", "password123")

	// Act - make actual HTTP request through the middleware pipeline
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.Header.Set("Hx-Request", "true")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "/", rec.Header().Get("Hx-Redirect"))
	mockUserRepo.AssertExpectations(t)
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	handler, mockUserRepo, _ := setupAuthHandler()
	e := setupEchoWithSession()

	tests := []struct {
		name      string
		email     string
		password  string
		setupMock func()
	}{
		{
			name:     "User not found",
			email:    "nonexistent@example.com",
			password: "password123",
			setupMock: func() {
				mockUserRepo.On("GetByEmail", mock.Anything, "nonexistent@example.com").
					Return(nil, errors.New("not found"))
			},
		},
		{
			name:     "Wrong password",
			email:    "test@example.com",
			password: "wrongpassword",
			setupMock: func() {
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
				user := &user.User{
					ID:       uuid.New(),
					Email:    "test@example.com",
					Password: string(hashedPassword),
					FamilyID: uuid.New(),
					Role:     user.RoleMember,
				}
				mockUserRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(user, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock renderer for error case
			mockRenderer := &MockTemplateRenderer{}
			e.Renderer = mockRenderer
			mockRenderer.On("Render", mock.Anything, "login.html", mock.MatchedBy(func(data any) bool {
				dataMap := data.(map[string]any)
				return dataMap["Error"] == "Invalid email or password"
			}), mock.Anything).Return(nil)

			tt.setupMock()

			// Prepare form data
			form := url.Values{}
			form.Add("email", tt.email)
			form.Add("password", tt.password)

			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Mock validation

			// Act
			err := handler.Login(c)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
			mockUserRepo.AssertExpectations(t)
			mockRenderer.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_Login_WithRedirect(t *testing.T) {
	handler, mockUserRepo, _ := setupAuthHandler()
	e := setupEchoWithSession()

	// Arrange
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	expectedUser := &user.User{
		ID:       uuid.New(),
		Email:    "test@example.com",
		Password: string(hashedPassword),
		FamilyID: uuid.New(),
		Role:     user.RoleMember,
	}

	mockUserRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(expectedUser, nil)

	// Add handler as a route to ensure middleware runs
	e.POST("/login", handler.Login)

	// Prepare form data
	form := url.Values{}
	form.Add("email", "test@example.com")
	form.Add("password", "password123")

	// Act - make actual HTTP request through the middleware pipeline
	req := httptest.NewRequest(http.MethodPost, "/login?redirect=/dashboard", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/dashboard", rec.Header().Get("Location"))
	mockUserRepo.AssertExpectations(t)
}

func TestAuthHandler_Login_RedirectSecurity(t *testing.T) {
	handler, mockUserRepo, _ := setupAuthHandler()
	e := setupEchoWithSession()

	// Arrange
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	expectedUser := &user.User{
		ID:       uuid.New(),
		Email:    "test@example.com",
		Password: string(hashedPassword),
		FamilyID: uuid.New(),
		Role:     user.RoleMember,
	}

	tests := []struct {
		name             string
		redirectParam    string
		expectedRedirect string
	}{
		{
			name:             "External URL blocked",
			redirectParam:    "http://evil.com/",
			expectedRedirect: "/",
		},
		{
			name:             "Protocol relative URL blocked",
			redirectParam:    "//evil.com/",
			expectedRedirect: "/",
		},
		{
			name:             "Valid internal path allowed",
			redirectParam:    "/dashboard",
			expectedRedirect: "/dashboard",
		},
		{
			name:             "Backslash normalized",
			redirectParam:    "/dash\\board",
			expectedRedirect: "/dash/board",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(expectedUser, nil).Once()

			// Add handler as a route to ensure middleware runs
			e.POST("/login", handler.Login)

			// Prepare form data
			form := url.Values{}
			form.Add("email", "test@example.com")
			form.Add("password", "password123")

			// Act - make actual HTTP request through the middleware pipeline
			req := httptest.NewRequest(
				http.MethodPost,
				"/login?redirect="+url.QueryEscape(tt.redirectParam),
				strings.NewReader(form.Encode()),
			)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, http.StatusFound, rec.Code)
			assert.Equal(t, tt.expectedRedirect, rec.Header().Get("Location"))
		})
	}
}

func TestAuthHandler_RegisterPage_Success(t *testing.T) {
	handler, _, _ := setupAuthHandler()
	e := setupEchoForTesting()

	// Setup mock renderer
	mockRenderer := &MockTemplateRenderer{}
	e.Renderer = mockRenderer
	mockRenderer.On("Render", mock.Anything, "register.html", mock.MatchedBy(func(data any) bool {
		dataMap := data.(map[string]any)
		return dataMap["Title"] == "Create Family Account" && dataMap["CSRFToken"] != nil
	}), mock.Anything).Return(nil)

	// Add handler as a route to ensure middleware runs
	e.GET("/register", handler.RegisterPage)

	// Act - make actual HTTP request through the middleware pipeline
	req := httptest.NewRequest(http.MethodGet, "/register", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)
	mockRenderer.AssertExpectations(t)
}

func TestAuthHandler_Register_Success(t *testing.T) {
	handler, mockUserRepo, mockFamilyRepo := setupAuthHandler()
	e := setupEchoWithSession()

	// Arrange
	mockUserRepo.On("GetByEmail", mock.Anything, "newuser@example.com").Return(nil, errors.New("not found"))
	mockFamilyRepo.On("Create", mock.Anything, mock.AnythingOfType("*user.Family")).Return(nil)
	mockUserRepo.On("Create", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil)

	// Add handler as a route to ensure middleware runs
	e.POST("/register", handler.Register)

	// Prepare form data
	form := url.Values{}
	form.Add("family_name", "Test Family")
	form.Add("currency", "USD")
	form.Add("first_name", "John")
	form.Add("last_name", "Doe")
	form.Add("email", "newuser@example.com")
	form.Add("password", "password123")

	// Act - make actual HTTP request through the middleware pipeline
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/", rec.Header().Get("Location"))
	mockUserRepo.AssertExpectations(t)
	mockFamilyRepo.AssertExpectations(t)
}

func TestAuthHandler_Register_UserAlreadyExists(t *testing.T) {
	handler, mockUserRepo, _ := setupAuthHandler()
	e := setupEchoWithSession()

	// Arrange
	existingUser := &user.User{
		ID:    uuid.New(),
		Email: "existing@example.com",
	}
	mockUserRepo.On("GetByEmail", mock.Anything, "existing@example.com").Return(existingUser, nil)

	// Setup mock renderer for error case
	mockRenderer := &MockTemplateRenderer{}
	e.Renderer = mockRenderer
	mockRenderer.On("Render", mock.Anything, "register.html", mock.MatchedBy(func(data any) bool {
		dataMap := data.(map[string]any)
		fieldErrors := dataMap["FieldErrors"].(map[string]string)
		return fieldErrors["email"] == "User with this email already exists"
	}), mock.Anything).Return(nil)

	// Prepare form data
	form := url.Values{}
	form.Add("family_name", "Test Family")
	form.Add("currency", "USD")
	form.Add("first_name", "John")
	form.Add("last_name", "Doe")
	form.Add("email", "existing@example.com")
	form.Add("password", "password123")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Act
	err := handler.Register(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	mockUserRepo.AssertExpectations(t)
	mockRenderer.AssertExpectations(t)
}

func TestAuthHandler_Logout_Success(t *testing.T) {
	handler, _, _ := setupAuthHandler()
	e := setupEchoWithSession()

	// Add handler as a route to ensure middleware runs
	e.POST("/logout", handler.Logout)

	// Act - make actual HTTP request through the middleware pipeline
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
}

func TestAuthHandler_Logout_HTMXRequest(t *testing.T) {
	handler, _, _ := setupAuthHandler()
	e := setupEchoWithSession()

	// Add handler as a route to ensure middleware runs
	e.POST("/logout", handler.Logout)

	// Act - make actual HTTP request through the middleware pipeline
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.Header.Set("Hx-Request", "true")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Hx-Redirect"))
}

// mockValidator is a simple mock validator for testing
type mockValidator struct{}

func (m *mockValidator) Validate(_ any) error {
	// For testing purposes, we'll assume validation always passes
	// In real tests, you might want to implement actual validation logic
	return nil
}

// CustomValidator wraps the validator to implement echo.Validator interface
type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i any) error {
	if err := cv.validator.Struct(i); err != nil {
		return err
	}
	return nil
}

// Benchmark tests for performance validation
func BenchmarkAuthHandler_Login(b *testing.B) {
	handler, mockUserRepo, _ := setupAuthHandler()
	e := setupEchoWithSession()

	// Setup mock user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	expectedUser := &user.User{
		ID:       uuid.New(),
		Email:    "bench@example.com",
		Password: string(hashedPassword),
		FamilyID: uuid.New(),
		Role:     user.RoleMember,
	}
	mockUserRepo.On("GetByEmail", mock.Anything, "bench@example.com").Return(expectedUser, nil)

	form := url.Values{}
	form.Add("email", "bench@example.com")
	form.Add("password", "password123")

	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler.Login(c)
	}
}

func BenchmarkAuthHandler_Register(b *testing.B) {
	handler, mockUserRepo, mockFamilyRepo := setupAuthHandler()
	e := setupEchoWithSession()

	// Setup mocks
	mockUserRepo.On("GetByEmail", mock.Anything, mock.AnythingOfType("string")).Return(nil, errors.New("not found"))
	mockFamilyRepo.On("Create", mock.Anything, mock.AnythingOfType("*user.Family")).Return(nil)
	mockUserRepo.On("Create", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil)

	form := url.Values{}
	form.Add("family_name", "Bench Family")
	form.Add("currency", "USD")
	form.Add("first_name", "Bench")
	form.Add("last_name", "User")
	form.Add("email", "bench@example.com")
	form.Add("password", "password123")

	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler.Register(c)
	}
}

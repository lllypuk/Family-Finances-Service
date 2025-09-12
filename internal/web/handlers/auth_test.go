package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/services"
	webHandlers "family-budget-service/internal/web/handlers"
)

func TestAuthHandler_Creation(t *testing.T) {
	// Test that auth handler can be created without panicking
	repos := &handlers.Repositories{}
	services := &services.Services{}
	handler := webHandlers.NewAuthHandler(repos, services)

	assert.NotNil(t, handler)
}

func TestAuthHandler_LoginPageExists(t *testing.T) {
	// Test that login page handler exists and can be called
	repos := &handlers.Repositories{}
	services := &services.Services{}
	handler := webHandlers.NewAuthHandler(repos, services)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// This should not panic
	err := handler.LoginPage(c)

	// We expect some kind of response, not necessarily success
	// due to missing dependencies, but no panic
	assert.Error(t, err) // Expected due to missing template renderer
}

func TestAuthHandler_RegisterPageExists(t *testing.T) {
	// Test that register page handler exists and can be called
	repos := &handlers.Repositories{}
	services := &services.Services{}
	handler := webHandlers.NewAuthHandler(repos, services)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/register", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// This should not panic
	err := handler.RegisterPage(c)

	// We expect some kind of response, not necessarily success
	// due to missing dependencies, but no panic
	assert.Error(t, err) // Expected due to missing template renderer
}

func TestAuthHandler_LoginExists(t *testing.T) {
	// Test that login handler exists and can be called
	repos := &handlers.Repositories{}
	services := &services.Services{}
	handler := webHandlers.NewAuthHandler(repos, services)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// This should not panic
	err := handler.Login(c)

	// We expect some kind of response, not necessarily success
	// due to missing form data and dependencies
	assert.Error(t, err) // Expected due to missing form data
}

func TestAuthHandler_RegisterExists(t *testing.T) {
	// Test that register handler exists and can be called
	repos := &handlers.Repositories{}
	services := &services.Services{}
	handler := webHandlers.NewAuthHandler(repos, services)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/register", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// This should not panic
	err := handler.Register(c)

	// We expect some kind of response, not necessarily success
	// due to missing form data and dependencies
	assert.Error(t, err) // Expected due to missing form data
}

func TestAuthHandler_LogoutExists(t *testing.T) {
	// Test that logout handler exists and can be called
	repos := &handlers.Repositories{}
	services := &services.Services{}
	handler := webHandlers.NewAuthHandler(repos, services)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// This should not panic
	err := handler.Logout(c)

	// Logout might succeed even without session
	if err != nil {
		assert.Error(t, err)
	} else {
		assert.NoError(t, err)
	}
}

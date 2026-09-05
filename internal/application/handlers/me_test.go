package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/auth"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

func TestMeHandler_GetMe(t *testing.T) {
	svc := newFakeAuthService()
	users := &MockUserService{}
	users.On("GetUserByID", mock.Anything, svc.user.ID).Return(svc.user, nil)

	c, rec := principalContext(http.MethodGet, "/api/v1/me", "", principalFor(svc))
	require.NoError(t, handlers.NewMeHandler(users, svc).GetMe(c))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body handlers.APIResponse[handlers.UserResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, svc.user.ID, body.Data.ID)
	assert.Equal(t, svc.user.Email, body.Data.Email)
	users.AssertExpectations(t)
}

func TestMeHandler_GetMe_NoPrincipal(t *testing.T) {
	svc := newFakeAuthService()
	c, rec := principalContext(http.MethodGet, "/api/v1/me", "", nil)

	require.NoError(t, handlers.NewMeHandler(&MockUserService{}, svc).GetMe(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMeHandler_UpdateMe(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newFakeAuthService()
		users := &MockUserService{}
		users.On("UpdateUser", mock.Anything, svc.user.ID, mock.MatchedBy(func(req dto.UpdateUserDTO) bool {
			return req.FirstName != nil && *req.FirstName == "Janet" && req.LastName == nil && req.Email == nil
		})).Return(svc.user, nil)

		c, rec := principalContext(http.MethodPut, "/api/v1/me", `{"first_name":"Janet"}`, principalFor(svc))
		require.NoError(t, handlers.NewMeHandler(users, svc).UpdateMe(c))

		assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		users.AssertExpectations(t)
	})

	t.Run("empty body is 422", func(t *testing.T) {
		svc := newFakeAuthService()
		c, rec := principalContext(http.MethodPut, "/api/v1/me", `{}`, principalFor(svc))

		require.NoError(t, handlers.NewMeHandler(&MockUserService{}, svc).UpdateMe(c))

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Equal(t, "VALIDATION_ERROR", decodeError(t, rec).Error.Code)
	})

	t.Run("bad email is 422", func(t *testing.T) {
		svc := newFakeAuthService()
		c, rec := principalContext(http.MethodPut, "/api/v1/me", `{"email":"nope"}`, principalFor(svc))

		require.NoError(t, handlers.NewMeHandler(&MockUserService{}, svc).UpdateMe(c))

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		body := decodeError(t, rec)
		require.Len(t, body.Error.Details, 1)
		assert.Equal(t, "email", body.Error.Details[0].Field)
	})

	t.Run("taken email is 409", func(t *testing.T) {
		svc := newFakeAuthService()
		users := &MockUserService{}
		users.On("UpdateUser", mock.Anything, svc.user.ID, mock.Anything).Return(nil, services.ErrEmailAlreadyExists)

		c, rec := principalContext(http.MethodPut, "/api/v1/me", `{"email":"taken@example.com"}`, principalFor(svc))
		require.NoError(t, handlers.NewMeHandler(users, svc).UpdateMe(c))

		assert.Equal(t, http.StatusConflict, rec.Code)
	})
}

func TestMeHandler_ChangePassword(t *testing.T) {
	const body = `{"current_password":"` + fakeLoginPassword + `","new_password":"new-password-1"}`

	t.Run("success keeps current session", func(t *testing.T) {
		svc := newFakeAuthService()
		c, rec := principalContext(http.MethodPut, "/api/v1/me/password", body, principalFor(svc))

		require.NoError(t, handlers.NewMeHandler(&MockUserService{}, svc).ChangePassword(c))

		assert.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		require.NotNil(t, svc.changeCall)
		assert.Equal(t, svc.user.ID, svc.changeCall.userID)
		assert.Equal(t, fakeLoginPassword, svc.changeCall.current)
		assert.Equal(t, "new-password-1", svc.changeCall.next)
		assert.Equal(t, svc.session.ID, svc.changeCall.keep)
	})

	t.Run("wrong current is 401", func(t *testing.T) {
		svc := newFakeAuthService()
		svc.changeErr = auth.ErrInvalidCredentials
		c, rec := principalContext(http.MethodPut, "/api/v1/me/password", body, principalFor(svc))

		require.NoError(t, handlers.NewMeHandler(&MockUserService{}, svc).ChangePassword(c))

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, "INVALID_CREDENTIALS", decodeError(t, rec).Error.Code)
	})

	t.Run("weak new password is 422 before the service", func(t *testing.T) {
		svc := newFakeAuthService()
		c, rec := principalContext(http.MethodPut, "/api/v1/me/password",
			`{"current_password":"x","new_password":"short"}`, principalFor(svc))

		require.NoError(t, handlers.NewMeHandler(&MockUserService{}, svc).ChangePassword(c))

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Nil(t, svc.changeCall)
		details := decodeError(t, rec).Error.Details
		require.Len(t, details, 1)
		assert.Equal(t, "new_password", details[0].Field)
	})

	t.Run("storage failure is 500", func(t *testing.T) {
		svc := newFakeAuthService()
		svc.changeErr = errors.New("db down")
		c, rec := principalContext(http.MethodPut, "/api/v1/me/password", body, principalFor(svc))

		require.NoError(t, handlers.NewMeHandler(&MockUserService{}, svc).ChangePassword(c))

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("no principal", func(t *testing.T) {
		svc := newFakeAuthService()
		c, rec := principalContext(http.MethodPut, "/api/v1/me/password", body, nil)

		require.NoError(t, handlers.NewMeHandler(&MockUserService{}, svc).ChangePassword(c))

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

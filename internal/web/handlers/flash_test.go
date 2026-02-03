package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetFlashMessage(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set flash message
	setFlashMessage(c, "success", "Operation completed successfully")

	// Check cookies were set
	cookies := rec.Result().Cookies()
	assert.Len(t, cookies, 2, "Should set 2 cookies")

	var msgCookie, typeCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == flashCookieName {
			msgCookie = cookie
		}
		if cookie.Name == flashTypeCookieName {
			typeCookie = cookie
		}
	}

	assert.NotNil(t, msgCookie, "Message cookie should be set")
	assert.NotNil(t, typeCookie, "Type cookie should be set")

	// Check message is URL encoded
	decodedMsg, decodeErr := url.QueryUnescape(msgCookie.Value)
	require.NoError(t, decodeErr)
	assert.Equal(t, "Operation completed successfully", decodedMsg)
	assert.Equal(t, "success", typeCookie.Value)

	// Check cookie properties
	assert.Equal(t, "/", msgCookie.Path)
	assert.Equal(t, flashCookieMaxAge, msgCookie.MaxAge)
	assert.True(t, msgCookie.HttpOnly)
	assert.Equal(t, http.SameSiteStrictMode, msgCookie.SameSite)
}

func TestGetFlashMessage(t *testing.T) {
	tests := []struct {
		name          string
		setupCookies  func(*http.Request)
		expectedType  string
		expectedMsg   string
		expectCookies bool
		checkCleared  bool
	}{
		{
			name: "Valid flash message",
			setupCookies: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  flashCookieName,
					Value: url.QueryEscape("Test message"),
				})
				req.AddCookie(&http.Cookie{
					Name:  flashTypeCookieName,
					Value: "error",
				})
			},
			expectedType:  "error",
			expectedMsg:   "Test message",
			expectCookies: true,
			checkCleared:  true,
		},
		{
			name: "Message without type defaults to info",
			setupCookies: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  flashCookieName,
					Value: url.QueryEscape("Info message"),
				})
			},
			expectedType:  "info",
			expectedMsg:   "Info message",
			expectCookies: true,
			checkCleared:  true,
		},
		{
			name: "No flash message",
			setupCookies: func(_ *http.Request) {
				// No cookies
			},
			expectedType:  "",
			expectedMsg:   "",
			expectCookies: false,
			checkCleared:  false,
		},
		{
			name: "URL encoded message",
			setupCookies: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  flashCookieName,
					Value: url.QueryEscape("Message with special chars: <>&\""),
				})
				req.AddCookie(&http.Cookie{
					Name:  flashTypeCookieName,
					Value: "warning",
				})
			},
			expectedType:  "warning",
			expectedMsg:   "Message with special chars: <>&\"",
			expectCookies: true,
			checkCleared:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			tt.setupCookies(req)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Get flash message
			msgType, message := GetFlashMessage(c)

			// Assert message content
			assert.Equal(t, tt.expectedType, msgType)
			assert.Equal(t, tt.expectedMsg, message)

			// Check that cookies were cleared if message existed
			if tt.checkCleared {
				cookies := rec.Result().Cookies()
				clearedCount := 0
				for _, cookie := range cookies {
					if (cookie.Name == flashCookieName || cookie.Name == flashTypeCookieName) && cookie.MaxAge == -1 {
						clearedCount++
					}
				}
				assert.Equal(t, 2, clearedCount, "Both cookies should be cleared")
			}
		})
	}
}

func TestRedirectWithFlashMessage(t *testing.T) {
	tests := []struct {
		name           string
		redirectFunc   func(*BaseHandler, echo.Context, string, string) error
		redirectURL    string
		message        string
		expectedType   string
		expectedStatus int
	}{
		{
			name: "Redirect with error",
			redirectFunc: func(h *BaseHandler, c echo.Context, url, msg string) error {
				return h.redirectWithError(c, url, msg)
			},
			redirectURL:    "/dashboard",
			message:        "Error occurred",
			expectedType:   "error",
			expectedStatus: http.StatusSeeOther,
		},
		{
			name: "Redirect with success",
			redirectFunc: func(h *BaseHandler, c echo.Context, url, msg string) error {
				return h.redirectWithSuccess(c, url, msg)
			},
			redirectURL:    "/users",
			message:        "User created successfully",
			expectedType:   "success",
			expectedStatus: http.StatusSeeOther,
		},
		{
			name: "Redirect with empty message",
			redirectFunc: func(h *BaseHandler, c echo.Context, url, msg string) error {
				return h.redirectWithError(c, url, msg)
			},
			redirectURL:    "/",
			message:        "",
			expectedType:   "",
			expectedStatus: http.StatusSeeOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			h := &BaseHandler{}
			err := tt.redirectFunc(h, c, tt.redirectURL, tt.message)

			// Check redirect
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)
			assert.Equal(t, tt.redirectURL, rec.Header().Get("Location"))

			// Check flash message cookie if message was provided
			if tt.message != "" {
				cookies := rec.Result().Cookies()
				var msgCookie, typeCookie *http.Cookie
				for _, cookie := range cookies {
					if cookie.Name == flashCookieName {
						msgCookie = cookie
					}
					if cookie.Name == flashTypeCookieName {
						typeCookie = cookie
					}
				}

				assert.NotNil(t, msgCookie, "Message cookie should be set")
				assert.NotNil(t, typeCookie, "Type cookie should be set")

				decodedMsg, decodeErr := url.QueryUnescape(msgCookie.Value)
				require.NoError(t, decodeErr)
				assert.Equal(t, tt.message, decodedMsg)
				assert.Equal(t, tt.expectedType, typeCookie.Value)
			}
		})
	}
}

func TestGetFlashMessages(t *testing.T) {
	tests := []struct {
		name         string
		setupCookies func(*http.Request)
		expectedLen  int
		expectedType string
		expectedText string
	}{
		{
			name: "Flash message exists",
			setupCookies: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  flashCookieName,
					Value: url.QueryEscape("Success message"),
				})
				req.AddCookie(&http.Cookie{
					Name:  flashTypeCookieName,
					Value: "success",
				})
			},
			expectedLen:  1,
			expectedType: "success",
			expectedText: "Success message",
		},
		{
			name: "No flash message",
			setupCookies: func(_ *http.Request) {
				// No cookies
			},
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			tt.setupCookies(req)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			h := &BaseHandler{}
			messages := h.getFlashMessages(c)

			assert.Len(t, messages, tt.expectedLen)
			if tt.expectedLen > 0 {
				assert.Equal(t, tt.expectedType, messages[0].Type)
				assert.Equal(t, tt.expectedText, messages[0].Text)
			}
		})
	}
}

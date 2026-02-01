package middleware

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
)

// SetupChecker defines an interface for checking setup status
type SetupChecker interface {
	IsSetupComplete(ctx context.Context) (bool, error)
}

// RequireSetup middleware redirects to /setup if the family has not been created yet.
// If the family already exists and the user requests /setup, redirects to /login.
func RequireSetup(checker SetupChecker) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Path()

			complete, err := checker.IsSetupComplete(c.Request().Context())
			if err != nil {
				// On error, allow access (graceful degradation)
				return next(c)
			}

			if !complete {
				// Family not set up yet — only /setup is accessible
				if path != "/setup" {
					return c.Redirect(http.StatusFound, "/setup")
				}
				return next(c)
			}

			// Family exists — /setup is no longer accessible
			if path == "/setup" {
				return c.Redirect(http.StatusFound, "/login")
			}

			return next(c)
		}
	}
}

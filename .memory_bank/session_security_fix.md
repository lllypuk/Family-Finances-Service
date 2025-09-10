# Session Security Fix

## Problem Description

**Date**: 2025-08-16  
**Issue**: Semgrep security scanner detected that session cookies were configured without the `Secure` flag, creating a security vulnerability.

**Original Code**:
```go
store.Options = &sessions.Options{
    Path:     "/",
    MaxAge:   int(SessionTimeout.Seconds()),
    HttpOnly: true,
    Secure:   false, // TODO: установить в true для production с HTTPS
    SameSite: http.SameSiteLaxMode,
}
```

**Security Risk**: Session cookies could be transmitted over insecure HTTP connections, making them vulnerable to interception and session hijacking attacks.

## Solution Implemented

### 1. Configuration Enhancement

Added environment detection to the main configuration:

**File**: `internal/config.go`
```go
type Config struct {
    Server      ServerConfig
    Database    DatabaseConfig
    Web         WebConfig
    Environment string  // Added environment field
}

// IsProduction returns true if the application is running in production environment
func (c *Config) IsProduction() bool {
    return c.Environment == "production"
}
```

### 2. Dynamic Secure Flag Setting

Modified the session middleware to accept a production flag:

**File**: `internal/web/middleware/session.go`
```go
// SessionStore настраивает хранилище сессий
func SessionStore(secretKey string, isProduction bool) echo.MiddlewareFunc {
    // ... existing code ...
    
    store.Options = &sessions.Options{
        Path:     "/",
        MaxAge:   int(SessionTimeout.Seconds()),
        HttpOnly: true,
        Secure:   isProduction, // Dynamic based on environment
        SameSite: http.SameSiteLaxMode,
    }
    
    return session.Middleware(store)
}
```

### 3. Integration Updates

Updated the HTTP server configuration and web server initialization:

**Files Modified**:
- `internal/application/http_server.go`: Added `IsProduction` field to Config
- `internal/web/web.go`: Updated `NewWebServer` to accept production flag
- `internal/run.go`: Pass production status from main config

## Environment Variables

The fix uses the `ENVIRONMENT` environment variable:

- **Development**: `ENVIRONMENT=development` → `Secure: false` (allows HTTP)
- **Production**: `ENVIRONMENT=production` → `Secure: true` (requires HTTPS)
- **Default**: If not set, defaults to "development"

## Testing

Added comprehensive tests in `internal/web/middleware/session_test.go`:

- ✅ Production environment sets secure cookies
- ✅ Development environment allows insecure cookies  
- ✅ Basic configuration works correctly
- ✅ Handles edge cases (empty secret key)
- ✅ Validates session constants and structure

## Security Impact

### Before Fix
- **High Risk**: Session cookies transmitted over HTTP
- **Vulnerability**: Session hijacking, man-in-the-middle attacks
- **Compliance**: Failed security scanning

### After Fix
- **Secure**: Production cookies only sent over HTTPS
- **Flexible**: Development still works with HTTP
- **Compliant**: Passes Semgrep security scanning
- **Best Practice**: Environment-aware security configuration

## Deployment Notes

1. **Production**: Ensure `ENVIRONMENT=production` is set
2. **HTTPS**: Production requires HTTPS for secure cookies to work
3. **Local Dev**: Uses `ENVIRONMENT=development` by default
4. **Docker**: Environment variable can be set in docker-compose or Kubernetes

## Commands for Testing

```bash
# Development (allows HTTP)
ENVIRONMENT=development make run-local

# Production (requires HTTPS)  
ENVIRONMENT=production make build

# Check security scanning
make lint
```

## Related Files

- `internal/config.go` - Configuration with environment detection
- `internal/web/middleware/session.go` - Session middleware with security fix
- `internal/application/http_server.go` - HTTP server configuration
- `internal/web/web.go` - Web server initialization
- `internal/run.go` - Application bootstrap
- `internal/web/middleware/session_test.go` - Security tests

## References

- [OWASP Session Management](https://owasp.org/www-project-cheat-sheets/cheatsheets/Session_Management_Cheat_Sheet.html)
- [Secure Cookie Flag Documentation](https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies#restrict_access_to_cookies)
- [Gorilla Sessions Security](https://github.com/gorilla/sessions#security)
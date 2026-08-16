package handlers

// Error codes used in API responses. Exported as constants to keep them
// consistent across handlers and to satisfy the goconst linter (otherwise the
// same literal would appear in multiple handlers and trigger duplicates).
const (
	// ErrCodeInvalidRequest signals a malformed request body.
	ErrCodeInvalidRequest = "INVALID_REQUEST"
	// ErrCodeValidationError signals a request that failed domain-level validation.
	ErrCodeValidationError = "VALIDATION_ERROR"
	// ErrCodeInvalidID signals an unparseable UUID in a path/query parameter.
	ErrCodeInvalidID = "INVALID_ID"
	// ErrCodeInvalidUserID signals an unparseable user UUID.
	ErrCodeInvalidUserID = "INVALID_USER_ID"
	// ErrCodeFamilyNotFound signals that the requested family does not exist.
	ErrCodeFamilyNotFound = "FAMILY_NOT_FOUND"
	// ErrCodeCategoryNotFound signals that the requested category does not exist.
	ErrCodeCategoryNotFound = "CATEGORY_NOT_FOUND"
	// ErrCodeUnauthorized signals a request without a valid session
	// (отдаёт RequireAPIAuth, см. api_auth.go).
	ErrCodeUnauthorized = "UNAUTHORIZED"
	// ErrCodeForbidden signals a session whose role is not allowed on the route
	// (отдаёт RequireAPIRole, см. api_auth.go).
	ErrCodeForbidden = "FORBIDDEN"
	// ErrCodeInternal signals a server-side failure; подробности только в логе.
	ErrCodeInternal = "INTERNAL_ERROR"
	// ErrCodeCannotDeleteSelf signals an attempt to delete the session's own user.
	ErrCodeCannotDeleteSelf = "CANNOT_DELETE_SELF"
	// ErrCodeLastAdmin signals an attempt to delete the last remaining admin.
	ErrCodeLastAdmin = "LAST_ADMIN"

	// Standard error messages paired with the codes above. Kept as constants
	// so changes propagate to API consumers in lockstep with code updates.
	ErrMessageUnauthorized       = "Authentication required"
	ErrMessageForbidden          = "Insufficient permissions"
	ErrMessageInvalidRequest     = "Invalid request body"
	ErrMessageInvalidUserID      = "Invalid user ID format"
	ErrMessageInvalidCategoryID  = "Invalid category ID format"
	ErrMessageInvalidTransaction = "Invalid transaction data"
	ErrMessageInvalidCategoryRef = "Invalid category, user, or family ID"
	ErrMessageFamilyNotFound     = "Family not found"
	ErrMessageInternal           = "Internal server error"
	ErrMessageCannotDeleteSelf   = "Cannot delete your own account"
	ErrMessageLastAdmin          = "Cannot delete the last administrator"
)

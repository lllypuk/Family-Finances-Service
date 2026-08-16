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
	// ErrCodeUnauthorized signals a request without a valid session. Совпадает
	// с кодом, который отдаёт middleware.RequireAPIAuth.
	ErrCodeUnauthorized = "UNAUTHORIZED"

	// Standard error messages paired with the codes above. Kept as constants
	// so changes propagate to API consumers in lockstep with code updates.
	ErrMessageInvalidRequest     = "Invalid request body"
	ErrMessageInvalidUserID      = "Invalid user ID format"
	ErrMessageInvalidCategoryID  = "Invalid category ID format"
	ErrMessageInvalidTransaction = "Invalid transaction data"
	ErrMessageInvalidCategoryRef = "Invalid category, user, or family ID"
	ErrMessageFamilyNotFound     = "Family not found"
)

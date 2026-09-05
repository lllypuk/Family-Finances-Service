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
	// ErrCodeUnauthorized signals a request without a valid bearer token (auth.RequireBearer).
	ErrCodeUnauthorized = "UNAUTHORIZED"
	// ErrCodeForbidden signals a token whose role is not allowed on the route (auth.RequireRole).
	ErrCodeForbidden = "FORBIDDEN"
	// ErrCodeNotFound signals an unknown route or a missing resource (404 outside a handler).
	ErrCodeNotFound = "NOT_FOUND"
	// ErrCodeMethodNotAllowed signals a method not registered on the route.
	ErrCodeMethodNotAllowed = "METHOD_NOT_ALLOWED"
	// ErrCodeBadRequest signals a request rejected before reaching a handler.
	ErrCodeBadRequest = "BAD_REQUEST"
	// ErrCodeInternal signals a server-side failure; details stay in the log only.
	ErrCodeInternal = "INTERNAL_ERROR"
	// entityUser — имя сущности для HandleNotFoundError: USER_NOT_FOUND / "User not found".
	entityUser = "User"
	// ErrCodeEmailTaken signals that another user already has the requested email.
	ErrCodeEmailTaken = "EMAIL_TAKEN"
	// ErrCodeCannotDeactivateSelf signals an attempt to deactivate the session's own user.
	ErrCodeCannotDeactivateSelf = "CANNOT_DEACTIVATE_SELF"
	// ErrCodeLastAdmin signals an attempt to deactivate or demote the last active admin.
	ErrCodeLastAdmin = "LAST_ADMIN"
	// ErrCodeCurrencyLocked signals a currency change on a family that already has transactions.
	ErrCodeCurrencyLocked = "CURRENCY_LOCKED"
	// ErrCodeInvalidQueryParam маркирует деталь ошибки 422 по query-параметру.
	ErrCodeInvalidQueryParam = "INVALID_QUERY_PARAM"
	// ErrCodeGenerationFailed signals a failed report generation.
	ErrCodeGenerationFailed = "GENERATION_FAILED"
	// ErrCodeSaveFailed signals that a generated entity could not be persisted.
	ErrCodeSaveFailed = "SAVE_FAILED"
	// ErrCodeExportFailed signals a failed report export.
	ErrCodeExportFailed = "EXPORT_FAILED"
	// ErrCodeInvalidBackupName signals a backup filename outside the `backup_*.db` pattern.
	ErrCodeInvalidBackupName = "INVALID_BACKUP_NAME"
	// ErrCodeBackupNotFound signals that the requested backup file does not exist.
	ErrCodeBackupNotFound = "BACKUP_NOT_FOUND"
	// ErrCodeBackupFailed signals that a backup could not be created.
	ErrCodeBackupFailed = "BACKUP_FAILED"
	// ErrCodeInvalidCredentials — неверный email или пароль; ответ одинаков для обоих случаев.
	//nolint:gosec // G101: это код ошибки в ответе API, а не учётные данные.
	ErrCodeInvalidCredentials = "INVALID_CREDENTIALS"
	// ErrCodeSetupRequired — семья ещё не создана CLI `setup`, логин невозможен.
	ErrCodeSetupRequired = "SETUP_REQUIRED"
	// ErrCodeRateLimited — сработал лимитер логина; секунды до повтора — в Retry-After.
	ErrCodeRateLimited = "RATE_LIMITED"

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
	ErrMessageCannotDeactivate   = "Cannot deactivate your own account"
	ErrMessageEmailTaken         = "Email already exists"
	ErrMessageLastAdmin          = "Cannot deactivate or demote the last administrator"
	ErrMessageCurrencyLocked     = "Currency cannot be changed while transactions exist"
	ErrMessageInvalidBackupName  = "Invalid backup filename"
	ErrMessageBackupNotFound     = "Backup not found"
	ErrMessageBackupFailed       = "Failed to create backup"
	ErrMessageValidationFailed   = "Validation failed"
	ErrMessageCategoryNotFound   = "Category not found"
	ErrMessageInvalidCredentials = "Invalid email or password"
	ErrMessageSetupRequired      = "Family is not set up yet"
	ErrMessageRateLimited        = "Too many login attempts"
	// ErrMessageNoFields — деталь 422 для частичного обновления без единого поля.
	ErrMessageNoFields = "at least one field is required"

	// fieldBody — значение ErrorDetail.Field для ошибок, не привязанных к полю.
	fieldBody        = "body"
	fieldRole        = "role"
	fieldNewPassword = "new_password"
	fieldCurrency    = "currency"
)

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
	// ErrCodeCurrencyLocked signals a currency change on a family that already has transactions, reconciliations or holding values.
	ErrCodeCurrencyLocked = "CURRENCY_LOCKED"
	// ErrCodeBudgetOverlap signals a budget whose period overlaps another budget of the same scope.
	ErrCodeBudgetOverlap = "BUDGET_OVERLAP"
	// ErrCodeBudgetNameExists signals a budget name already used for the same period.
	ErrCodeBudgetNameExists = "BUDGET_NAME_EXISTS"
	// ErrCodeBudgetBelowSpent signals a budget amount below what the period has already spent.
	ErrCodeBudgetBelowSpent = "BUDGET_BELOW_SPENT"
	// ErrCodeBudgetIDExists signals a client id already taken by a deleted budget.
	ErrCodeBudgetIDExists = "BUDGET_ID_EXISTS"
	// ErrCodeBudgetNotTail signals an operation on a stale tail: the series has already advanced.
	ErrCodeBudgetNotTail = "BUDGET_NOT_TAIL"
	// ErrCodeAccountNotFound signals that the requested account does not exist.
	ErrCodeAccountNotFound = "ACCOUNT_NOT_FOUND"
	// ErrCodeAccountNameExists signals an account name already taken, by an archived account too.
	ErrCodeAccountNameExists = "ACCOUNT_NAME_EXISTS"
	// ErrCodeAccountInUse signals a delete of an account referenced by transactions or reconciliations.
	ErrCodeAccountInUse = "ACCOUNT_IN_USE"
	// ErrCodeHoldingNotFound signals that the requested holding does not exist.
	ErrCodeHoldingNotFound = "HOLDING_NOT_FOUND"
	// ErrCodeHoldingNameExists signals a holding name already taken, by an archived holding too.
	ErrCodeHoldingNameExists = "HOLDING_NAME_EXISTS"
	// ErrCodeHoldingValueNotFound signals that the holding has no value on the date.
	ErrCodeHoldingValueNotFound = "HOLDING_VALUE_NOT_FOUND"
	// ErrCodeReconciliationNotFound signals that the account has no reconciliation for the month.
	ErrCodeReconciliationNotFound = "RECONCILIATION_NOT_FOUND"
	// ErrCodeInvalidQueryParam маркирует деталь ошибки 422 по query-параметру.
	ErrCodeInvalidQueryParam = "INVALID_QUERY_PARAM"
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
	// ErrCodeRecognitionUnavailable — плечо модели выключено или не ответило; срок повтора — в Retry-After.
	ErrCodeRecognitionUnavailable = "RECOGNITION_UNAVAILABLE"
	// ErrCodeRecognitionFailed — модель ответила, но ответ не разобрался.
	ErrCodeRecognitionFailed = "RECOGNITION_FAILED"
	// ErrCodePayloadTooLarge — тело больше лимита маршрута (BodyLimit).
	ErrCodePayloadTooLarge = "PAYLOAD_TOO_LARGE"
	// ErrCodeRequestTimeout — тело не пришло за срок загрузки.
	ErrCodeRequestTimeout = "REQUEST_TIMEOUT"

	// Standard error messages paired with the codes above. Kept as constants
	// so changes propagate to API consumers in lockstep with code updates.
	ErrMessageUnauthorized           = "Authentication required"
	ErrMessageForbidden              = "Insufficient permissions"
	ErrMessageInvalidRequest         = "Invalid request body"
	ErrMessageInvalidUserID          = "Invalid user ID format"
	ErrMessageInvalidCategoryID      = "Invalid category ID format"
	ErrMessageInvalidTransaction     = "Invalid transaction data"
	ErrMessageInvalidCategoryRef     = "Invalid category, user, or family ID"
	ErrMessageFamilyNotFound         = "Family not found"
	ErrMessageInternal               = "Internal server error"
	ErrMessageCannotDeactivate       = "Cannot deactivate your own account"
	ErrMessageEmailTaken             = "Email already exists"
	ErrMessageLastAdmin              = "Cannot deactivate or demote the last administrator"
	ErrMessageCurrencyLocked         = "Currency cannot be changed while transactions, reconciliations or holding values exist"
	ErrMessageBudgetOverlap          = "Budget period overlaps with an existing budget"
	ErrMessageBudgetNameExists       = "Budget with this name already exists for this period"
	ErrMessageBudgetBelowSpent       = "Budget amount is less than already spent"
	ErrMessageBudgetIDExists         = "Budget id is already taken by a deleted budget"
	ErrMessageBudgetNotTail          = "Budget has already advanced to the next period"
	ErrMessageInvalidAccountID       = "Invalid account ID format"
	ErrMessageAccountNotFound        = "Account not found"
	ErrMessageAccountNameExists      = "Account with this name already exists"
	ErrMessageAccountInUse           = "Account has transactions or reconciliations; archive it instead"
	ErrMessageReconciliationNotFound = "Reconciliation not found"
	ErrMessageInvalidHoldingID       = "Invalid holding ID format"
	ErrMessageHoldingNotFound        = "Holding not found"
	ErrMessageHoldingNameExists      = "Holding with this name already exists"
	ErrMessageHoldingValueNotFound   = "Holding value not found"
	ErrMessageInvalidValueDate       = "Invalid date format, expected YYYY-MM-DD"
	ErrMessageInvalidBackupName      = "Invalid backup filename"
	ErrMessageBackupNotFound         = "Backup not found"
	ErrMessageBackupFailed           = "Failed to create backup"
	ErrMessageValidationFailed       = "Validation failed"
	ErrMessageCategoryNotFound       = "Category not found"
	ErrMessageInvalidCredentials     = "Invalid email or password"
	ErrMessageSetupRequired          = "Family is not set up yet"
	ErrMessageRateLimited            = "Too many login attempts"
	ErrMessageRecognitionUnavailable = "Recognition is unavailable"
	ErrMessageRecognitionFailed      = "Model answer could not be read"
	ErrMessagePayloadTooLarge        = "Request body is too large"
	ErrMessageRequestTimeout         = "Request body was not received in time"
	// ErrMessageNoFields — деталь 422 для частичного обновления без единого поля.
	ErrMessageNoFields = "at least one field is required"

	// fieldBody — значение ErrorDetail.Field для ошибок, не привязанных к полю.
	fieldBody        = "body"
	fieldID          = "id"
	fieldRole        = "role"
	fieldNewPassword = "new_password"
	fieldCurrency    = "currency"
	fieldDate        = "date"
	fieldStartDate   = "start_date"
	fieldEndDate     = "end_date"
	fieldRecurring   = "recurring"
	fieldAccountID   = "account_id"
	fieldClearAcct   = "clear_account"
	fieldUnassigned  = "unassigned"
	fieldSide        = "side"
	fieldKind        = "kind"
	fieldValueMinor  = "value_minor"
)

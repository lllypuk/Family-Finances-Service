package handlers

// Template context keys and other repeated literal values used by HTML
// handlers when rendering pages and partials. Centralized as constants so
// changes propagate together, and so goconst does not flag the same literal
// in many files as a duplicate.
//
// NOTE: key names must match what templates reference (e.g. `.CSRFToken`),
// so we keep them in PascalCase even though Go would prefer snake_case.
const (
	tplKeyCSRFToken   = "CSRFToken"
	tplKeyCurrentUser = "CurrentUser"
	tplKeyFamily      = "Family"
	tplKeyBudget      = "Budget"
	tplKeyErrors      = "Errors"
	tplKeyError       = "Error"
	tplKeyFieldErrors = "FieldErrors"
	tplKeyEmail       = "Email"
	tplKeyForm        = "form"
	tplKeyLabel       = "Label"
	tplKeyName        = "Name"
	tplKeyType        = "Type"

	// Display values rendered into the UI.
	allCategoriesOptionName = "All Categories"
)

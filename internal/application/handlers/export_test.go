package handlers

import "net/http"

// DecodeJSON открывает decodeJSON внешним тестам.
func DecodeJSON(r *http.Request, v any) error { return decodeJSON(r, v) }

// RespondBindError открывает respondBindError внешним тестам.
func RespondBindError(w http.ResponseWriter, err error) error {
	return respondBindError(w, err)
}

// ErrResponseWritten — sentinel parsePagination для внешних тестов.
func ErrResponseWritten() error { return errResponseAlreadyWritten }

// PageParams — разобранные limit/offset для внешних тестов.
type PageParams = pageParams

// ParsePagination открывает parsePagination внешним тестам.
func ParsePagination(w http.ResponseWriter, r *http.Request) (PageParams, error) {
	return parsePagination(w, r)
}

// RespondList открывает respondList внешним тестам.
func RespondList(w http.ResponseWriter, items []string, page PageParams, total int) error {
	return respondList(w, items, page, total)
}

// RespondAPI открывает respondAPI внешним тестам.
func RespondAPI(w http.ResponseWriter, status int, data map[string]string) error {
	return respondAPI(w, status, data)
}

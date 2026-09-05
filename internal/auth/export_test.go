package auth

import "net/http"

// BearerToken открывает разбор заголовка тестам внешнего пакета.
func BearerToken(r *http.Request) (string, bool) { return bearerToken(r) }

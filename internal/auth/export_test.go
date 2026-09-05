package auth

import (
	"net/http"
	"time"
)

// BearerToken открывает разбор заголовка тестам внешнего пакета.
func BearerToken(r *http.Request) (string, bool) { return bearerToken(r) }

// SetClock подменяет источник времени сервиса.
func (s *Service) SetClock(now func() time.Time) { s.now = now }

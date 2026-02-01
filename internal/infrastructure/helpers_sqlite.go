package infrastructure

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// SQLiteHelpers provides utility functions for working with SQLite database

// NullStringToUUID converts sql.NullString to UUID
func NullStringToUUID(ns sql.NullString) (uuid.UUID, error) {
	if !ns.Valid {
		return uuid.Nil, nil
	}
	return uuid.Parse(ns.String)
}

// UUIDToString converts UUID to string for SQLite storage
func UUIDToString(id uuid.UUID) string {
	return id.String()
}

// UUIDPtrToString converts *UUID to string for SQLite storage
// Returns nil if input is nil to maintain NULL semantics in database
func UUIDPtrToString(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}

// ErrNilStringValue is returned when a nil string is provided where UUID is expected
var ErrNilStringValue = errors.New("nil string value provided")

// StringToUUIDPtr converts string to *UUID
// Returns (nil, ErrNilStringValue) if input is nil to maintain NULL semantics
// Returns error if string is invalid UUID format
func StringToUUIDPtr(s *string) (*uuid.UUID, error) {
	if s == nil {
		return nil, ErrNilStringValue
	}
	id, err := uuid.Parse(*s)
	if err != nil {
		return nil, fmt.Errorf("failed to parse UUID: %w", err)
	}
	return &id, nil
}

// BoolToInt converts bool to int for SQLite storage (0 or 1)
func BoolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// IntToBool converts int to bool from SQLite storage
func IntToBool(i int) bool {
	return i != 0
}

// ConvertPlaceholders converts PostgreSQL-style placeholders ($1, $2) to SQLite-style (?, ?)
// Note: This is a simple implementation and may not work for all cases
// For production, consider using a more robust SQL parser
func ConvertPlaceholders(query string) string {
	// This is intentionally not implemented as it's better to rewrite queries manually
	// to ensure correctness. Automatic conversion can lead to subtle bugs.
	return query
}

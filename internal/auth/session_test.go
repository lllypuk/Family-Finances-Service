package auth_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/auth"
)

// Срок продлевается активностью на IdleTTL, но не дальше AbsoluteTTL от выдачи.
func TestSession_ExpiryAfter(t *testing.T) {
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s := auth.NewSession(uuid.New(), "h", "d", created)

	assert.Equal(t, created.Add(auth.IdleTTL), s.ExpiresAt)
	assert.Equal(t, created.Add(auth.IdleTTL+time.Hour), s.ExpiryAfter(created.Add(time.Hour)))
	assert.Equal(t, created.Add(auth.AbsoluteTTL), s.ExpiryAfter(created.Add(auth.AbsoluteTTL-time.Hour)))
	assert.Equal(t, created.Add(auth.AbsoluteTTL), s.ExpiryAfter(created.Add(auth.AbsoluteTTL+time.Hour)))
}

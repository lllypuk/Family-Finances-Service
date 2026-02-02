package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

// InviteStatus represents the status of an invite
type InviteStatus string

const (
	// InviteStatusPending means invite is waiting to be accepted
	InviteStatusPending InviteStatus = "pending"
	// InviteStatusAccepted means invite has been used
	InviteStatusAccepted InviteStatus = "accepted"
	// InviteStatusExpired means invite has expired
	InviteStatusExpired InviteStatus = "expired"
	// InviteStatusRevoked means invite has been cancelled by admin
	InviteStatusRevoked InviteStatus = "revoked"
)

const (
	// DefaultInviteValidityDays is the default number of days an invite is valid
	DefaultInviteValidityDays = 7
	// InviteTokenLength is the length of the invite token in bytes (32 bytes = 64 hex chars)
	InviteTokenLength = 32
)

// Invite represents an invitation to join a family
type Invite struct {
	ID         uuid.UUID    `json:"id"`
	FamilyID   uuid.UUID    `json:"family_id"`
	CreatedBy  uuid.UUID    `json:"created_by"`
	Email      string       `json:"email"`
	Role       Role         `json:"role"`
	Token      string       `json:"token"`
	Status     InviteStatus `json:"status"`
	ExpiresAt  time.Time    `json:"expires_at"`
	AcceptedAt *time.Time   `json:"accepted_at,omitempty"`
	AcceptedBy *uuid.UUID   `json:"accepted_by,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
}

// NewInvite creates a new invite
func NewInvite(familyID, createdBy uuid.UUID, email string, role Role) (*Invite, error) {
	token, err := generateInviteToken()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	expiresAt := now.AddDate(0, 0, DefaultInviteValidityDays)

	return &Invite{
		ID:        uuid.New(),
		FamilyID:  familyID,
		CreatedBy: createdBy,
		Email:     email,
		Role:      role,
		Token:     token,
		Status:    InviteStatusPending,
		ExpiresAt: expiresAt,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// IsValid checks if the invite is still valid
func (i *Invite) IsValid() bool {
	return i.Status == InviteStatusPending && time.Now().Before(i.ExpiresAt)
}

// IsExpired checks if the invite has expired
func (i *Invite) IsExpired() bool {
	return time.Now().After(i.ExpiresAt)
}

// Accept marks the invite as accepted
func (i *Invite) Accept(userID uuid.UUID) {
	now := time.Now()
	i.Status = InviteStatusAccepted
	i.AcceptedAt = &now
	i.AcceptedBy = &userID
	i.UpdatedAt = now
}

// Revoke marks the invite as revoked
func (i *Invite) Revoke() {
	i.Status = InviteStatusRevoked
	i.UpdatedAt = time.Now()
}

// MarkExpired marks the invite as expired
func (i *Invite) MarkExpired() {
	i.Status = InviteStatusExpired
	i.UpdatedAt = time.Now()
}

// generateInviteToken generates a cryptographically secure random token
func generateInviteToken() (string, error) {
	bytes := make([]byte, InviteTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// InviteRepository defines the interface for invite persistence
type InviteRepository interface {
	// Create creates a new invite
	Create(ctx context.Context, invite *Invite) error

	// GetByToken retrieves an invite by its token
	GetByToken(ctx context.Context, token string) (*Invite, error)

	// GetByID retrieves an invite by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*Invite, error)

	// GetByFamily retrieves all invites for a family
	GetByFamily(ctx context.Context, familyID uuid.UUID) ([]*Invite, error)

	// GetPendingByEmail retrieves pending invites for an email
	GetPendingByEmail(ctx context.Context, email string) ([]*Invite, error)

	// Update updates an invite
	Update(ctx context.Context, invite *Invite) error

	// Delete deletes an invite
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteExpired deletes all expired invites
	DeleteExpired(ctx context.Context) error
}

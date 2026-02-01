package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services/dto"
)

var (
	// ErrInviteNotFound is returned when invite is not found
	ErrInviteNotFound = errors.New("invite not found")
	// ErrInviteExpired is returned when invite has expired
	ErrInviteExpired = errors.New("invite has expired")
	// ErrInviteInvalid is returned when invite is not valid
	ErrInviteInvalid = errors.New("invite is not valid")
	// ErrInviteAlreadyUsed is returned when invite has already been accepted
	ErrInviteAlreadyUsed = errors.New("invite has already been used")
	// ErrInviteRevoked is returned when invite has been revoked
	ErrInviteRevoked = errors.New("invite has been revoked")
)

// InviteService defines business operations for invite management
type InviteService interface {
	// CreateInvite creates a new invite (admin only)
	CreateInvite(ctx context.Context, creatorID uuid.UUID, req dto.CreateInviteDTO) (*user.Invite, error)

	// GetInviteByToken retrieves an invite by its token
	GetInviteByToken(ctx context.Context, token string) (*user.Invite, error)

	// AcceptInvite accepts an invite and creates a new user
	AcceptInvite(ctx context.Context, token string, req dto.AcceptInviteDTO) (*user.User, error)

	// RevokeInvite revokes an invite (admin only)
	RevokeInvite(ctx context.Context, inviteID, revokerID uuid.UUID) error

	// ListFamilyInvites retrieves all invites for the family
	ListFamilyInvites(ctx context.Context, familyID uuid.UUID) ([]*user.Invite, error)

	// DeleteExpiredInvites removes all expired invites
	DeleteExpiredInvites(ctx context.Context) error
}

// inviteService implements InviteService interface
type inviteService struct {
	inviteRepo user.InviteRepository
	userRepo   UserRepository
	familyRepo FamilyRepository
}

// NewInviteService creates a new InviteService instance
func NewInviteService(
	inviteRepo user.InviteRepository,
	userRepo UserRepository,
	familyRepo FamilyRepository,
) InviteService {
	return &inviteService{
		inviteRepo: inviteRepo,
		userRepo:   userRepo,
		familyRepo: familyRepo,
	}
}

// CreateInvite creates a new invite (admin only)
func (s *inviteService) CreateInvite(
	ctx context.Context,
	creatorID uuid.UUID,
	req dto.CreateInviteDTO,
) (*user.Invite, error) {
	// Validate creator exists and is admin
	creator, err := s.userRepo.GetByID(ctx, creatorID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if creator.Role != user.RoleAdmin {
		return nil, ErrUnauthorized
	}

	// Get the single family
	family, err := s.familyRepo.Get(ctx)
	if err != nil {
		return nil, ErrFamilyNotFound
	}

	// Normalize email
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Check if user with this email already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return nil, ErrEmailAlreadyExists
	}

	// Check for existing pending invites for this email
	pendingInvites, err := s.inviteRepo.GetPendingByEmail(email)
	if err == nil && len(pendingInvites) > 0 {
		return nil, fmt.Errorf("pending invite already exists for email: %s", email)
	}

	// Validate role
	role := user.Role(req.Role)
	if role != user.RoleAdmin && role != user.RoleMember && role != user.RoleChild {
		return nil, ErrInvalidRole
	}

	// Create new invite
	invite, err := user.NewInvite(family.ID, creatorID, email, role)
	if err != nil {
		return nil, fmt.Errorf("failed to create invite: %w", err)
	}

	// Save to repository
	if createErr := s.inviteRepo.Create(invite); createErr != nil {
		return nil, fmt.Errorf("failed to save invite: %w", createErr)
	}

	return invite, nil
}

// GetInviteByToken retrieves an invite by its token
func (s *inviteService) GetInviteByToken(_ context.Context, token string) (*user.Invite, error) {
	invite, err := s.inviteRepo.GetByToken(token)
	if err != nil {
		return nil, ErrInviteNotFound
	}

	// Check if expired
	if invite.IsExpired() && invite.Status == user.InviteStatusPending {
		invite.MarkExpired()
		if updateErr := s.inviteRepo.Update(invite); updateErr != nil {
			// Log error but don't fail the request
			// In production, use proper logging instead of fmt.Printf
			_ = updateErr // Acknowledge error without printing
		}
		return nil, ErrInviteExpired
	}

	return invite, nil
}

// AcceptInvite accepts an invite and creates a new user
func (s *inviteService) AcceptInvite(ctx context.Context, token string, req dto.AcceptInviteDTO) (*user.User, error) {
	// Get invite by token
	invite, err := s.GetInviteByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Validate invite status
	if !invite.IsValid() {
		switch invite.Status {
		case user.InviteStatusAccepted:
			return nil, ErrInviteAlreadyUsed
		case user.InviteStatusRevoked:
			return nil, ErrInviteRevoked
		case user.InviteStatusExpired:
			return nil, ErrInviteExpired
		case user.InviteStatusPending:
			return nil, ErrInviteInvalid
		default:
			return nil, ErrInviteInvalid
		}
	}

	// Verify email matches
	if strings.ToLower(strings.TrimSpace(req.Email)) != invite.Email {
		return nil, errors.New("email does not match invite")
	}

	// Check if user already exists with this email
	existingUser, err := s.userRepo.GetByEmail(ctx, invite.Email)
	if err == nil && existingUser != nil {
		return nil, ErrEmailAlreadyExists
	}

	// Create the user
	newUser := user.NewUser(invite.Email, req.Name, req.Name, invite.Role)

	// Hash password before storing
	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	newUser.Password = hashedPassword

	// Save user to repository
	if createErr := s.userRepo.Create(ctx, newUser); createErr != nil {
		return nil, fmt.Errorf("failed to save user: %w", createErr)
	}

	// Mark invite as accepted
	invite.Accept(newUser.ID)
	if updateErr := s.inviteRepo.Update(invite); updateErr != nil {
		// User is created, log error but don't fail
		// In production, use proper logging instead of fmt.Printf
		_ = updateErr // Acknowledge error without printing
	}

	return newUser, nil
}

// RevokeInvite revokes an invite (admin only)
func (s *inviteService) RevokeInvite(ctx context.Context, inviteID, revokerID uuid.UUID) error {
	// Validate revoker exists and is admin
	revoker, err := s.userRepo.GetByID(ctx, revokerID)
	if err != nil {
		return ErrUserNotFound
	}

	if revoker.Role != user.RoleAdmin {
		return ErrUnauthorized
	}

	// Get the single family
	family, err := s.familyRepo.Get(ctx)
	if err != nil {
		return ErrFamilyNotFound
	}

	// Get invite
	invite, err := s.inviteRepo.GetByID(inviteID)
	if err != nil {
		return ErrInviteNotFound
	}

	// Check if invite belongs to the same family
	if invite.FamilyID != family.ID {
		return ErrUnauthorized
	}

	// Check if already accepted or revoked
	if invite.Status == user.InviteStatusAccepted {
		return errors.New("cannot revoke accepted invite")
	}

	if invite.Status == user.InviteStatusRevoked {
		return errors.New("invite already revoked")
	}

	// Revoke invite
	invite.Revoke()
	if updateErr := s.inviteRepo.Update(invite); updateErr != nil {
		return fmt.Errorf("failed to revoke invite: %w", updateErr)
	}

	return nil
}

// ListFamilyInvites retrieves all invites for the family
func (s *inviteService) ListFamilyInvites(_ context.Context, familyID uuid.UUID) ([]*user.Invite, error) {
	invites, err := s.inviteRepo.GetByFamily(familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get family invites: %w", err)
	}

	// Update expired invites
	for _, inv := range invites {
		if inv.Status == user.InviteStatusPending && inv.IsExpired() {
			inv.MarkExpired()
			if updateErr := s.inviteRepo.Update(inv); updateErr != nil {
				// Log error but continue
				// In production, use proper logging instead of fmt.Printf
				_ = updateErr // Acknowledge error without printing
			}
		}
	}

	return invites, nil
}

// DeleteExpiredInvites removes all expired invites
func (s *inviteService) DeleteExpiredInvites(_ context.Context) error {
	if err := s.inviteRepo.DeleteExpired(); err != nil {
		return fmt.Errorf("failed to delete expired invites: %w", err)
	}
	return nil
}

// hashPassword hashes a password using bcrypt
func hashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

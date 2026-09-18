package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/account"
	"family-budget-service/internal/services/dto"
)

// AccountRepository — хранилище счетов единственной семьи.
type AccountRepository interface {
	Create(ctx context.Context, a *account.Account) error
	GetByID(ctx context.Context, id uuid.UUID) (*account.Account, error)
	List(ctx context.Context, includeArchived bool) ([]*account.Account, error)
	Update(ctx context.Context, a *account.Account) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type accountService struct {
	repo AccountRepository
}

func NewAccountService(repo AccountRepository) AccountService {
	return &accountService{repo: repo}
}

func (s *accountService) Create(ctx context.Context, id *uuid.UUID, name string) (*account.Account, error) {
	normalized, err := account.NormalizeName(name)
	if err != nil {
		return nil, err
	}

	a := &account.Account{ID: dto.EntityID(id), Name: normalized}
	if err = s.repo.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	return a, nil
}

func (s *accountService) GetByID(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *accountService) List(ctx context.Context, includeArchived bool) ([]*account.Account, error) {
	return s.repo.List(ctx, includeArchived)
}

func (s *accountService) Update(
	ctx context.Context,
	id uuid.UUID,
	name *string,
	archived *bool,
) (*account.Account, error) {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if name != nil {
		if a.Name, err = account.NormalizeName(*name); err != nil {
			return nil, err
		}
	}
	if archived != nil {
		a.IsArchived = *archived
	}

	if err = s.repo.Update(ctx, a); err != nil {
		return nil, fmt.Errorf("failed to update account: %w", err)
	}

	return a, nil
}

func (s *accountService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

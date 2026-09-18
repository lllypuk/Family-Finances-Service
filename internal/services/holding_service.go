package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/holding"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/services/dto"
)

// HoldingRepository — хранилище позиций капитала единственной семьи.
type HoldingRepository interface {
	Create(ctx context.Context, h *holding.Holding) error
	GetByID(ctx context.Context, id uuid.UUID, today date.Date) (*holding.Holding, error)
	List(ctx context.Context, includeArchived bool, today date.Date) ([]*holding.Holding, error)
	Update(ctx context.Context, id uuid.UUID, name *string, kind *holding.Kind, archived *bool) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpsertValue(ctx context.Context, holdingID uuid.UUID, v *holding.Value) error
	DeleteValue(ctx context.Context, holdingID uuid.UUID, day date.Date) error
	ListValues(ctx context.Context, holdingID uuid.UUID, limit, offset int) ([]*holding.Value, int, error)
	SeriesValues(ctx context.Context, from, to date.Date) ([]holding.SeriesRow, error)
}

type holdingService struct {
	repo     HoldingRepository
	families FamilyRepository
}

func NewHoldingService(repo HoldingRepository, families FamilyRepository) HoldingService {
	return &holdingService{repo: repo, families: families}
}

func (s *holdingService) Create(
	ctx context.Context,
	id *uuid.UUID,
	name string,
	side holding.Side,
	kind holding.Kind,
) (*holding.Holding, error) {
	normalized, err := holding.NormalizeName(name)
	if err != nil {
		return nil, err
	}
	if !holding.ValidSide(side) {
		return nil, holding.ErrInvalidSide
	}
	if !holding.ValidKind(side, kind) {
		return nil, holding.ErrInvalidKind
	}

	h := &holding.Holding{ID: dto.EntityID(id), Name: normalized, Side: side, Kind: kind}
	if err = s.repo.Create(ctx, h); err != nil {
		return nil, fmt.Errorf("failed to create holding: %w", err)
	}

	return h, nil
}

func (s *holdingService) GetByID(ctx context.Context, id uuid.UUID) (*holding.Holding, error) {
	today, err := s.today(ctx)
	if err != nil {
		return nil, err
	}

	return s.repo.GetByID(ctx, id, today)
}

func (s *holdingService) List(ctx context.Context, includeArchived bool) ([]*holding.Holding, error) {
	today, err := s.today(ctx)
	if err != nil {
		return nil, err
	}

	return s.repo.List(ctx, includeArchived, today)
}

// Update проверяет новый вид по стороне из базы: side не меняется, поэтому чтение до записи безопасно.
func (s *holdingService) Update(
	ctx context.Context,
	id uuid.UUID,
	name *string,
	kind *holding.Kind,
	archived *bool,
) (*holding.Holding, error) {
	if name != nil {
		normalized, err := holding.NormalizeName(*name)
		if err != nil {
			return nil, err
		}
		name = &normalized
	}

	today, err := s.today(ctx)
	if err != nil {
		return nil, err
	}

	if kind != nil {
		current, getErr := s.repo.GetByID(ctx, id, today)
		if getErr != nil {
			return nil, getErr
		}
		if !holding.ValidKind(current.Side, *kind) {
			return nil, holding.ErrInvalidKind
		}
	}

	if err = s.repo.Update(ctx, id, name, kind, archived); err != nil {
		return nil, fmt.Errorf("failed to update holding: %w", err)
	}

	return s.repo.GetByID(ctx, id, today)
}

func (s *holdingService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// PutValue пишет снимок и архивной позиции: история правится независимо от видимости.
func (s *holdingService) PutValue(
	ctx context.Context,
	id uuid.UUID,
	day date.Date,
	value money.Minor,
) (*holding.Value, error) {
	if !holding.ValidValue(value) {
		return nil, holding.ErrValueOutOfRange
	}
	today, err := s.today(ctx)
	if err != nil {
		return nil, err
	}
	if err = holding.CheckValueDate(day, today); err != nil {
		return nil, err
	}
	if _, err = s.repo.GetByID(ctx, id, today); err != nil {
		return nil, err
	}

	v := &holding.Value{Date: day, ValueMinor: value}
	if err = s.repo.UpsertValue(ctx, id, v); err != nil {
		return nil, fmt.Errorf("failed to save holding value: %w", err)
	}

	return v, nil
}

func (s *holdingService) DeleteValue(ctx context.Context, id uuid.UUID, day date.Date) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}

	return s.repo.DeleteValue(ctx, id, day)
}

func (s *holdingService) ListValues(
	ctx context.Context,
	id uuid.UUID,
	limit, offset int,
) ([]*holding.Value, int, error) {
	if _, err := s.GetByID(ctx, id); err != nil {
		return nil, 0, err
	}

	return s.repo.ListValues(ctx, id, limit, offset)
}

func (s *holdingService) today(ctx context.Context) (date.Date, error) {
	family, err := s.families.Get(ctx)
	if err != nil {
		return date.Date{}, fmt.Errorf("failed to get family: %w", err)
	}

	return date.Today(family.Location()), nil
}

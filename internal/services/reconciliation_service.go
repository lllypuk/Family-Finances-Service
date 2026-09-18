package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/reconciliation"
	"family-budget-service/internal/services/dto"
)

// ReconciliationRepository — цифры банка по счетам и месяцам `YYYY-MM`.
type ReconciliationRepository interface {
	Upsert(ctx context.Context, rec *reconciliation.Reconciliation) error
	Delete(ctx context.Context, accountID uuid.UUID, month string) error
	ListByMonth(ctx context.Context, month string) ([]*reconciliation.Reconciliation, error)
}

type reconciliationService struct {
	repo         ReconciliationRepository
	accounts     AccountRepository
	transactions TransactionRepository
	families     FamilyRepository
}

func NewReconciliationService(
	repo ReconciliationRepository,
	accounts AccountRepository,
	transactions TransactionRepository,
	families FamilyRepository,
) ReconciliationService {
	return &reconciliationService{repo: repo, accounts: accounts, transactions: transactions, families: families}
}

func (s *reconciliationService) Put(
	ctx context.Context,
	accountID uuid.UUID,
	month date.Date,
	bankExpense money.Minor,
	note string,
) (*reconciliation.Reconciliation, error) {
	if !reconciliation.ValidAmount(bankExpense) {
		return nil, reconciliation.ErrAmountOutOfRange
	}
	if _, err := s.accounts.GetByID(ctx, accountID); err != nil {
		return nil, err
	}

	rec := &reconciliation.Reconciliation{
		AccountID:        accountID,
		Month:            month.MonthKey(),
		BankExpenseMinor: bankExpense,
		Note:             note,
	}
	if err := s.repo.Upsert(ctx, rec); err != nil {
		return nil, fmt.Errorf("failed to save reconciliation: %w", err)
	}

	return rec, nil
}

func (s *reconciliationService) Delete(ctx context.Context, accountID uuid.UUID, month date.Date) error {
	if _, err := s.accounts.GetByID(ctx, accountID); err != nil {
		return err
	}

	return s.repo.Delete(ctx, accountID, month.MonthKey())
}

func (s *reconciliationService) Summary(ctx context.Context, month *date.Date) (*dto.ReconciliationStats, error) {
	var day date.Date
	if month != nil {
		day = *month
	} else {
		family, err := s.families.Get(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get family: %w", err)
		}
		day = date.Today(family.Location())
	}
	first, last := day.MonthBounds()
	key := first.MonthKey()

	accounts, err := s.accounts.List(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	totals, err := s.transactions.RecordedByAccount(ctx, first, last)
	if err != nil {
		return nil, fmt.Errorf("failed to sum recorded expenses: %w", err)
	}
	recs, err := s.repo.ListByMonth(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to list reconciliations: %w", err)
	}

	stats := &dto.ReconciliationStats{Month: key, Accounts: make([]dto.ReconciliationRow, 0, len(accounts))}
	recorded := make(map[uuid.UUID]money.Minor, len(totals))
	for _, t := range totals {
		if t.AccountID == nil {
			stats.UnassignedMinor = t.AmountMinor
			continue
		}
		recorded[*t.AccountID] = t.AmountMinor
	}
	byAccount := make(map[uuid.UUID]*reconciliation.Reconciliation, len(recs))
	for _, rec := range recs {
		byAccount[rec.AccountID] = rec
	}

	for _, a := range accounts {
		spent, hasSpent := recorded[a.ID]
		rec, hasRec := byAccount[a.ID]
		if a.IsArchived && !hasSpent && !hasRec {
			continue
		}

		row := dto.ReconciliationRow{
			Account: dto.ReconciliationAccount{
				ID: a.ID, Name: a.Name, IsArchived: a.IsArchived, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
			},
			RecordedMinor: spent,
		}
		if hasRec {
			diff := rec.BankExpenseMinor - spent
			row.BankExpenseMinor = &rec.BankExpenseMinor
			row.DiffMinor = &diff
			row.Note = &rec.Note
			row.UpdatedAt = &rec.UpdatedAt
		}
		stats.Accounts = append(stats.Accounts, row)
	}

	return stats, nil
}

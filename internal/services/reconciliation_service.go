package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/account"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/reconciliation"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/services/dto"
)

// ErrReconciliationMonthInFuture — месяц позже текущего в поясе семьи.
var ErrReconciliationMonthInFuture = errors.New("reconciliation month is in the future")

// BalanceRepository — остатки счетов на конец месяца `YYYY-MM`.
type BalanceRepository interface {
	Upsert(ctx context.Context, b *reconciliation.Balance) error
	Delete(ctx context.Context, accountID uuid.UUID, month string) error
	ByMonths(ctx context.Context, prev, month string) ([]*reconciliation.Balance, error)
}

type reconciliationService struct {
	balances     BalanceRepository
	accounts     AccountRepository
	transactions TransactionRepository
	families     FamilyRepository
}

func NewReconciliationService(
	balances BalanceRepository,
	accounts AccountRepository,
	transactions TransactionRepository,
	families FamilyRepository,
) ReconciliationService {
	return &reconciliationService{
		balances:     balances,
		accounts:     accounts,
		transactions: transactions,
		families:     families,
	}
}

func (s *reconciliationService) PutBalance(
	ctx context.Context,
	accountID uuid.UUID,
	month date.Date,
	balance money.Minor,
) (*reconciliation.Balance, error) {
	if !reconciliation.ValidBalance(balance) {
		return nil, reconciliation.ErrBalanceOutOfRange
	}
	if err := s.checkMonth(ctx, month); err != nil {
		return nil, err
	}
	if _, err := s.accounts.GetByID(ctx, accountID); err != nil {
		return nil, err
	}

	b := &reconciliation.Balance{AccountID: accountID, Month: month.MonthKey(), BalanceMinor: balance}
	if err := s.balances.Upsert(ctx, b); err != nil {
		return nil, fmt.Errorf("failed to save account balance: %w", err)
	}

	return b, nil
}

func (s *reconciliationService) DeleteBalance(ctx context.Context, accountID uuid.UUID, month date.Date) error {
	if err := s.checkMonth(ctx, month); err != nil {
		return err
	}
	if _, err := s.accounts.GetByID(ctx, accountID); err != nil {
		return err
	}

	return s.balances.Delete(ctx, accountID, month.MonthKey())
}

func (s *reconciliationService) checkMonth(ctx context.Context, month date.Date) error {
	loc, err := s.location(ctx)
	if err != nil {
		return err
	}
	current, _ := date.Today(loc).MonthBounds()
	if first, _ := month.MonthBounds(); current.Before(first) {
		return ErrReconciliationMonthInFuture
	}

	return nil
}

func (s *reconciliationService) location(ctx context.Context) (*time.Location, error) {
	family, err := s.families.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get family: %w", err)
	}

	return family.Location(), nil
}

func (s *reconciliationService) Summary(ctx context.Context, month *date.Date) (*dto.ReconciliationStats, error) {
	loc, err := s.location(ctx)
	if err != nil {
		return nil, err
	}
	current, _ := date.Today(loc).MonthBounds()
	first := current
	if month != nil {
		first, _ = month.MonthBounds()
		if current.Before(first) {
			return nil, ErrReconciliationMonthInFuture
		}
	}
	_, last := first.MonthBounds()
	key, prevKey := first.MonthKey(), first.AddMonths(-1).MonthKey()

	accounts, err := s.accounts.List(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	balances, err := s.balances.ByMonths(ctx, prevKey, key)
	if err != nil {
		return nil, fmt.Errorf("failed to list account balances: %w", err)
	}
	totals, err := s.transactions.GetTotalsByMonth(ctx, first, last)
	if err != nil {
		return nil, fmt.Errorf("failed to sum transactions: %w", err)
	}

	stats := &dto.ReconciliationStats{Month: key, Accounts: make([]dto.ReconciliationRow, 0, len(accounts))}
	stats.IncomeMinor, stats.ExpenseMinor = incomeExpense(totals)

	opening := make(map[uuid.UUID]*reconciliation.Balance)
	closing := make(map[uuid.UUID]*reconciliation.Balance)
	for _, b := range balances {
		if b.Month == key {
			closing[b.AccountID] = b
		} else {
			opening[b.AccountID] = b
		}
	}
	for _, a := range accounts {
		created := date.FromTime(a.CreatedAt.In(loc))
		if row, ok := reconciliationRow(a, created, first, last, opening[a.ID], closing[a.ID]); ok {
			stats.Accounts = append(stats.Accounts, row)
		}
	}
	fillEdges(stats)

	return stats, nil
}

func incomeExpense(totals []transaction.MonthTotal) (money.Minor, money.Minor) {
	var income, expense money.Minor
	for _, t := range totals {
		switch t.Type {
		case transaction.TypeIncome:
			income += t.AmountMinor
		case transaction.TypeExpense:
			expense += t.AmountMinor
		}
	}

	return income, expense
}

// fillEdges суммирует края, заполненные у каждого счёта, и считает gap, когда заполнены оба.
func fillEdges(stats *dto.ReconciliationStats) {
	// Пустой список — не «все края заполнены»: иначе gap = −(приход − расход).
	if len(stats.Accounts) == 0 {
		return
	}

	var openingSum, closingSum money.Minor
	openingFilled, closingFilled := true, true
	for _, row := range stats.Accounts {
		if row.OpeningMinor == nil {
			openingFilled = false
		} else {
			openingSum += *row.OpeningMinor
		}
		if row.ClosingMinor == nil {
			closingFilled = false
		} else {
			closingSum += *row.ClosingMinor
		}
	}

	if openingFilled {
		stats.OpeningMinor = &openingSum
	}
	if closingFilled {
		stats.ClosingMinor = &closingSum
	}
	if openingFilled && closingFilled {
		gap := (closingSum - openingSum) - (stats.IncomeMinor - stats.ExpenseMinor)
		stats.GapMinor = &gap
		stats.Complete = true
	}
}

// reconciliationRow — строка счёта за месяц [first, last]; false — счёт в сверку месяца не входит.
func reconciliationRow(
	a *account.Account,
	created, first, last date.Date,
	opening, closing *reconciliation.Balance,
) (dto.ReconciliationRow, bool) {
	if a.IsArchived {
		if closing == nil && (opening == nil || opening.BalanceMinor == 0) {
			return dto.ReconciliationRow{}, false
		}
	} else if created.After(last) {
		return dto.ReconciliationRow{}, false
	}

	row := dto.ReconciliationRow{Account: dto.ReconciliationAccount{
		ID: a.ID, Name: a.Name, IsArchived: a.IsArchived, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}}
	switch {
	case opening != nil:
		row.OpeningMinor = &opening.BalanceMinor
	case !created.Before(first) && !created.After(last):
		var zero money.Minor
		row.OpeningMinor = &zero
	}
	switch {
	case closing != nil:
		row.ClosingMinor = &closing.BalanceMinor
		row.UpdatedAt = &closing.UpdatedAt
	case a.IsArchived:
		// Карта закрыта, деньги ушли в closing другого счёта.
		var zero money.Minor
		row.ClosingMinor = &zero
	}

	return row, true
}

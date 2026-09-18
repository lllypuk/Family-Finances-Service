package services_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/account"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

func incomeRequest(accountID *uuid.UUID) dto.CreateTransactionDTO {
	return dto.CreateTransactionDTO{
		AmountMinor: 1000,
		Type:        transaction.TypeIncome,
		Description: "Зарплата",
		CategoryID:  uuid.New(),
		AccountID:   accountID,
		UserID:      uuid.New(),
		Date:        date.Today(time.UTC),
	}
}

func TestTransactionService_CreateTransaction_AccountChecks(t *testing.T) {
	archived := &account.Account{ID: uuid.New(), Name: "Старая карта", IsArchived: true}
	active := &account.Account{ID: uuid.New(), Name: "Карта"}
	unknownID := uuid.New()

	tests := []struct {
		name      string
		accountID uuid.UUID
		found     *account.Account
		wantErr   error
	}{
		{"active", active.ID, active, nil},
		{"archived", archived.ID, archived, services.ErrTransactionAccountInvalid},
		{"unknown", unknownID, nil, services.ErrTransactionAccountInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, txRepo, _, categoryRepo, userRepo, accountRepo := setupTransactionServiceWithAccounts()
			req := incomeRequest(&tt.accountID)
			userRepo.On("GetByID", mock.Anything, req.UserID).Return(createTestUser(uuid.Nil), nil)
			categoryRepo.On("GetByID", mock.Anything, req.CategoryID).
				Return(createTestCategory(req.CategoryID, category.TypeIncome), nil)
			if tt.found != nil {
				accountRepo.On("GetByID", mock.Anything, tt.accountID).Return(tt.found, nil)
			} else {
				accountRepo.On("GetByID", mock.Anything, tt.accountID).
					Return(nil, fmt.Errorf("%w: %s", account.ErrNotFound, tt.accountID))
			}
			txRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()

			created, err := service.CreateTransaction(t.Context(), req)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				txRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, created.AccountID)
			assert.Equal(t, tt.accountID, *created.AccountID)
		})
	}
}

func TestTransactionService_UpdateTransaction_AccountChange(t *testing.T) {
	archivedID := uuid.New()
	archived := &account.Account{ID: archivedID, Name: "Старая карта", IsArchived: true}

	tests := []struct {
		name    string
		req     dto.UpdateTransactionDTO
		lookup  bool
		want    *uuid.UUID
		wantErr error
	}{
		{name: "no field keeps the account", req: dto.UpdateTransactionDTO{}, want: &archivedID},
		{name: "same archived account is not checked",
			req: dto.UpdateTransactionDTO{AccountID: new(archivedID)}, want: &archivedID},
		{name: "clear unlinks", req: dto.UpdateTransactionDTO{ClearAccount: true}},
		{name: "other archived account is refused", lookup: true,
			req:     dto.UpdateTransactionDTO{AccountID: new(uuid.New())},
			wantErr: services.ErrTransactionAccountInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, txRepo, _, _, _, accountRepo := setupTransactionServiceWithAccounts()
			existing := createTestTransactionWithCategory(
				uuid.New(), uuid.New(), 500, transaction.TypeIncome, date.Today(time.UTC))
			existing.AccountID = new(archivedID)
			txRepo.On("GetByID", mock.Anything, existing.ID).Return(existing, nil)
			txRepo.On("Update", mock.Anything, mock.Anything).Return(nil).Maybe()
			if tt.lookup {
				accountRepo.On("GetByID", mock.Anything, *tt.req.AccountID).Return(archived, nil)
			}
			description := "Новое описание"
			tt.req.Description = &description

			updated, err := service.UpdateTransaction(t.Context(), existing.ID, tt.req)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				txRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, updated.AccountID)
			accountRepo.AssertExpectations(t)
		})
	}
}

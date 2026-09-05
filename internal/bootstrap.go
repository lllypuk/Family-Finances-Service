package internal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/infrastructure"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

// migrationsDir — относительно CWD: сервер и CLI запускаются из корня репозитория или образа.
const migrationsDir = "./migrations"

// OpenDatabase открывает SQLite и применяет миграции; общая точка входа сервера и CLI.
func OpenDatabase(cfg *Config) (*sql.DB, error) {
	conn, err := infrastructure.NewSQLiteConnection(cfg.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SQLite: %w", err)
	}

	dbURL := fmt.Sprintf("sqlite://%s", cfg.Database.Path)
	if err = infrastructure.NewMigrationManager(dbURL, migrationsDir).Up(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return conn.DB(), nil
}

// SetupParams — входные данные CLI `setup`.
type SetupParams struct {
	FamilyName string
	Currency   string
	Timezone   string
	Email      string
	FirstName  string
	LastName   string
	Password   string
}

// Setup создаёт семью, категории и админа одной транзакцией.
// Повтор → services.ErrFamilyAlreadyExists.
func Setup(ctx context.Context, db *sql.DB, p SetupParams) (*user.Family, error) {
	repos := infrastructure.NewRepositoriesSQLite(db)
	familyService := services.NewFamilyService(repos.Family, repos.Transaction)

	return familyService.SetupFamily(ctx, dto.SetupFamilyDTO{
		FamilyName: p.FamilyName,
		Currency:   p.Currency,
		Timezone:   p.Timezone,
		Email:      p.Email,
		FirstName:  p.FirstName,
		LastName:   p.LastName,
		Password:   p.Password,
	})
}

// ResetPassword ставит новый пароль по email и отзывает все сессии пользователя.
// Неизвестный email → user.ErrNotFound, слабый пароль → auth.ErrInvalidPassword.
func ResetPassword(ctx context.Context, db *sql.DB, email, password string) error {
	if err := auth.ValidatePassword(password); err != nil {
		return err
	}

	repos := infrastructure.NewRepositoriesSQLite(db)
	authService, err := auth.NewService(repos.Session, repos.User, repos.Family)
	if err != nil {
		return fmt.Errorf("failed to initialize auth service: %w", err)
	}

	u, err := repos.User.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return user.ErrNotFound
		}
		return fmt.Errorf("failed to look up user: %w", err)
	}

	return authService.AdminSetPassword(ctx, u.ID, password)
}

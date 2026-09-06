package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"

	"family-budget-service/internal"
	"family-budget-service/internal/observability"
	"family-budget-service/internal/services"
)

const cmdBackup = "backup"

// runBackup — подкоманда `backup`: VACUUM INTO в BACKUP_DIR и удаление лишних файлов.
// Запускается по cron хоста через `docker compose exec`, поэтому пишет и в stdout, и в лог.
func runBackup(ctx context.Context, args []string, _ io.Reader, stdout io.Writer) error {
	keep, err := parseBackupArgs(args)
	if err != nil {
		return err
	}

	cfg := internal.LoadConfig()
	if keep <= 0 {
		keep = cfg.Database.BackupKeep
	}

	db, err := internal.OpenDatabase(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	logger := observability.NewLogger(observability.LogConfig{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
	})

	service := services.NewBackupService(db, cfg.Database.Path, cfg.GetBackupDir(), keep, logger)
	info, err := service.CreateBackup(ctx)
	if err != nil {
		return err
	}

	logger.InfoContext(ctx, "backup created",
		slog.String("file", info.Filename),
		slog.Int64("size", info.Size),
		slog.Int("keep", keep),
	)

	_, err = fmt.Fprintf(stdout, "backup %s created (%d bytes)\n", info.Filename, info.Size)
	return err
}

func parseBackupArgs(args []string) (int, error) {
	fs := flag.NewFlagSet(cmdBackup, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	keep := fs.Int("keep", 0, "how many newest backups to keep (default: BACKUP_KEEP)")

	if err := fs.Parse(args); err != nil {
		return 0, fmt.Errorf("%s: %w", cmdBackup, err)
	}

	return *keep, nil
}

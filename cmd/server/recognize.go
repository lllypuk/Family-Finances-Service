package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"family-budget-service/internal"
	"family-budget-service/internal/infrastructure"
	"family-budget-service/internal/infrastructure/llmengine"
	"family-budget-service/internal/recognize"
	"family-budget-service/internal/services"
)

const cmdRecognize = "recognize"

// errRecognizeDisabled — вызывать некого; main выходит с кодом 2, как на неверной командной строке.
var errRecognizeDisabled = errors.New("LLM_OLLAMA_HOST is not set, recognition is disabled")

// recognizeCommand — подкоманда `recognize <file>...`: Result в stdout, отчёт вызова модели в stderr.
func recognizeCommand(stderr io.Writer) command {
	return func(ctx context.Context, args []string, _ io.Reader, stdout io.Writer) error {
		paths, err := parseRecognizeArgs(args)
		if err != nil {
			return err
		}

		cfg := internal.LoadConfig()
		if cfg.LLM.OllamaHost == "" {
			return fmt.Errorf("%s: %w", cmdRecognize, errRecognizeDisabled)
		}
		if err = cfg.Validate(); err != nil {
			return fmt.Errorf("%s: %w", cmdRecognize, err)
		}

		images, err := readImages(paths)
		if err != nil {
			return err
		}

		// Открытие создало бы пустую БД по неверному пути.
		if _, statErr := os.Stat(cfg.Database.Path); statErr != nil {
			return fmt.Errorf("%s: database %s: %w", cmdRecognize, cfg.Database.Path, statErr)
		}

		db, err := internal.OpenDatabaseNoMigrate(cfg)
		if err != nil {
			return err
		}
		defer db.Close()

		logger := slog.New(slog.NewTextHandler(stderr, nil))
		repos := infrastructure.NewRepositoriesSQLite(db)
		service := services.NewRecognizeService(
			llmengine.New(cfg.LLM.OllamaHost, cfg.LLM.Model, cfg.LLM.Timeout, nil, logger),
			repos.Family,
			repos.Category,
			repos.Transaction,
			services.NopRecognizeObserver{},
		)

		result, err := service.Recognize(ctx, images)
		if err != nil {
			return err
		}

		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}
}

func parseRecognizeArgs(args []string) ([]string, error) {
	fs := flag.NewFlagSet(cmdRecognize, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("%s: %w", cmdRecognize, err)
	}

	if fs.NArg() == 0 || fs.NArg() > recognize.MaxImages {
		return nil, fmt.Errorf("%s: want 1…%d PNG or JPEG files, got %d", cmdRecognize, recognize.MaxImages, fs.NArg())
	}

	return fs.Args(), nil
}

func readImages(paths []string) ([]recognize.Image, error) {
	images := make([]recognize.Image, 0, len(paths))

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", cmdRecognize, err)
		}

		mime, err := recognize.CheckImage(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %s: %w", cmdRecognize, path, err)
		}

		images = append(images, recognize.Image{MIME: mime, Data: data})
	}

	return images, nil
}

package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"family-budget-service/internal"
)

const cmdMigrate = "migrate"

// runMigrate — подкоманда `migrate`: без флагов печатает версию схемы, с `--to N` двигает её.
// Нужна для отката релиза: старый образ не стартует на версии, которой нет в его ./migrations.
func runMigrate(_ context.Context, args []string, _ io.Reader, stdout io.Writer) error {
	target, targetSet, err := parseMigrateArgs(args)
	if err != nil {
		return err
	}

	cfg := internal.LoadConfig()
	if targetSet {
		if err = internal.MigrateTo(cfg, target); err != nil {
			return err
		}
	}

	version, dirty, err := internal.SchemaVersion(cfg)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "schema version %d (dirty: %t)\n", version, dirty)
	return err
}

func parseMigrateArgs(args []string) (uint, bool, error) {
	fs := flag.NewFlagSet(cmdMigrate, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	to := fs.Uint("to", 0, "target schema version (default: only print the current one)")

	if err := fs.Parse(args); err != nil {
		return 0, false, fmt.Errorf("%s: %w", cmdMigrate, err)
	}

	var set bool
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "to" {
			set = true
		}
	})

	return *to, set, nil
}

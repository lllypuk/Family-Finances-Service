package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"

	"family-budget-service/internal"
	"family-budget-service/internal/auth"
	"family-budget-service/internal/services/dto"
)

const (
	cmdSetup         = "setup"
	cmdResetPassword = "reset-password"
)

// errPasswordStdinRequired — пароль не принимается через argv, только через stdin.
var errPasswordStdinRequired = errors.New("password is read from stdin only: pass --password-stdin")

// runSetup — подкоманда `setup`: семья, категории и админ одной транзакцией.
func runSetup(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer) error {
	params, err := parseSetupArgs(args, stdin)
	if err != nil {
		return err
	}

	db, err := internal.OpenDatabase(internal.LoadConfig())
	if err != nil {
		return err
	}
	defer db.Close()

	family, err := internal.Setup(ctx, db, params)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "family %q created, admin %s\n", family.Name, params.Email)
	return err
}

// runResetPassword — подкоманда `reset-password`: новый пароль и отзыв всех сессий пользователя.
func runResetPassword(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer) error {
	email, password, err := parseResetPasswordArgs(args, stdin)
	if err != nil {
		return err
	}

	db, err := internal.OpenDatabase(internal.LoadConfig())
	if err != nil {
		return err
	}
	defer db.Close()

	if err = internal.ResetPassword(ctx, db, email, password); err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "password for %s reset, all sessions revoked\n", email)
	return err
}

func parseSetupArgs(args []string, stdin io.Reader) (dto.SetupFamilyDTO, error) {
	fs := flag.NewFlagSet(cmdSetup, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var p dto.SetupFamilyDTO
	var passwordStdin bool
	fs.StringVar(&p.FamilyName, "family", "", "family name (required)")
	fs.StringVar(&p.Currency, "currency", "", "ISO 4217 currency code, e.g. RUB (required)")
	fs.StringVar(&p.Timezone, "timezone", "", "IANA timezone, e.g. Europe/Moscow (required)")
	fs.StringVar(&p.Email, "email", "", "admin email (required)")
	fs.StringVar(&p.FirstName, "first-name", "", "admin first name (required)")
	fs.StringVar(&p.LastName, "last-name", "", "admin last name (required)")
	fs.BoolVar(&passwordStdin, "password-stdin", false, "read admin password from the first line of stdin")

	if err := fs.Parse(args); err != nil {
		return dto.SetupFamilyDTO{}, fmt.Errorf("%s: %w", cmdSetup, err)
	}

	required := map[string]string{
		"family":     p.FamilyName,
		"currency":   p.Currency,
		"timezone":   p.Timezone,
		"email":      p.Email,
		"first-name": p.FirstName,
		"last-name":  p.LastName,
	}
	if err := requireFlags(cmdSetup, required); err != nil {
		return dto.SetupFamilyDTO{}, err
	}

	password, err := readPassword(passwordStdin, stdin)
	if err != nil {
		return dto.SetupFamilyDTO{}, fmt.Errorf("%s: %w", cmdSetup, err)
	}
	p.Password = password

	return p, nil
}

func parseResetPasswordArgs(args []string, stdin io.Reader) (string, string, error) {
	fs := flag.NewFlagSet(cmdResetPassword, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var email string
	var passwordStdin bool
	fs.StringVar(&email, "email", "", "user email (required)")
	fs.BoolVar(&passwordStdin, "password-stdin", false, "read the new password from the first line of stdin")

	if err := fs.Parse(args); err != nil {
		return "", "", fmt.Errorf("%s: %w", cmdResetPassword, err)
	}
	if err := requireFlags(cmdResetPassword, map[string]string{"email": email}); err != nil {
		return "", "", err
	}

	password, err := readPassword(passwordStdin, stdin)
	if err != nil {
		return "", "", fmt.Errorf("%s: %w", cmdResetPassword, err)
	}

	return email, password, nil
}

func requireFlags(cmd string, values map[string]string) error {
	var missing []string
	for name, v := range values {
		if strings.TrimSpace(v) == "" {
			missing = append(missing, "--"+name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	slices.Sort(missing)
	return fmt.Errorf("%s: missing required flags: %s", cmd, strings.Join(missing, ", "))
}

// readPassword — первая строка stdin без завершающего перевода строки; политика — auth.ValidatePassword.
func readPassword(fromStdin bool, stdin io.Reader) (string, error) {
	if !fromStdin {
		return "", errPasswordStdinRequired
	}

	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("failed to read password from stdin: %w", err)
	}
	password := strings.TrimRight(line, "\r\n")
	if err = auth.ValidatePassword(password); err != nil {
		return "", err
	}

	return password, nil
}

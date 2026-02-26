package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"family-budget-service/internal"
)

const (
	// HealthCheckTimeout timeout for health check requests
	HealthCheckTimeout = 2 * time.Second
)

func main() {
	// Проверка флага healthcheck
	if len(os.Args) > 1 && os.Args[1] == "-health-check" {
		healthCheck()
		return
	}

	app, err := internal.NewApplication()
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	err = app.Run()
	if err != nil {
		log.Fatalf("Failed to run application: %v", err)
	}
}

func healthCheck() {
	exitCode := doHealthCheck()
	os.Exit(exitCode)
}

func doHealthCheck() int {
	cfg := internal.LoadConfig()
	return doHealthCheckWithURL(&http.Client{}, buildHealthCheckURL(cfg.Server.Host, cfg.Server.Port))
}

func doHealthCheckWithURL(client *http.Client, healthURL string) int {
	ctx, cancel := context.WithTimeout(context.Background(), HealthCheckTimeout)
	defer cancel()

	if client == nil {
		client = &http.Client{}
	}

	if err := checkHealth(ctx, client, healthURL); err != nil {
		log.Printf("Health check failed: %v", err)
		return 1
	}

	log.Println("Health check passed")
	return 0
}

func checkHealth(ctx context.Context, client *http.Client, healthURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func buildHealthCheckURL(host, port string) string {
	safeHost := normalizeHealthCheckHost(host)
	safePort := strings.TrimSpace(port)
	if safePort == "" {
		safePort = "8080"
	}

	return (&url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(safeHost, safePort),
		Path:   "/health",
	}).String()
}

func normalizeHealthCheckHost(host string) string {
	trimmed := strings.TrimSpace(host)
	if trimmed == "" {
		return "localhost"
	}

	switch trimmed {
	case "0.0.0.0", "::", "[::]":
		return "localhost"
	}

	// net.JoinHostPort expects raw IPv6 without square brackets.
	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		return strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]")
	}

	return trimmed
}

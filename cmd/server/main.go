package main

import (
	"context"
	"log"
	"net/http"
	"os"
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
	ctx, cancel := context.WithTimeout(context.Background(), HealthCheckTimeout)
	defer cancel()

	client := &http.Client{}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:8080/health", nil)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return 1
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Health check failed: %v", err)
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Health check failed with status: %d", resp.StatusCode)
		return 1
	}

	log.Println("Health check passed")
	return 0
}

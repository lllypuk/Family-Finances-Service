package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"family-budget-service/internal"
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

	if err := app.Run(); err != nil {
		log.Fatalf("Failed to run application: %v", err)
	}
}

func healthCheck() {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get("http://localhost:8080/health")
	if err != nil {
		log.Printf("Health check failed: %v", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Health check failed with status: %d", resp.StatusCode)
		os.Exit(1)
	}

	log.Println("Health check passed")
	os.Exit(0)
}

package main

import (
	"log"

	"family-budget-service/internal"
)

func main() {
	app, err := internal.NewApplication()
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	if err := app.Run(); err != nil {
		log.Fatalf("Failed to run application: %v", err)
	}
}

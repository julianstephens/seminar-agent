package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/julianstephens/formation/internal/app"
	"github.com/julianstephens/formation/internal/config"
)

func main() {
	// Load configuration from Infisical
	cfg, err := config.LoadFromInfisical()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
		os.Exit(1)
	}

	// Create app instance
	ctx := context.Background()
	application, err := app.New(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	// Run the application (blocks until shutdown signal)
	if err := application.Run(); err != nil {
		slog.Error("application error", "error", err)
		os.Exit(1)
	}
}

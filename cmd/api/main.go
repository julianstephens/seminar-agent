package main

import (
	"context"
	"log"

	"github.com/joho/godotenv"

	"github.com/julianstephens/formation/internal/app"
	"github.com/julianstephens/formation/internal/config"
)

func main() {
	// Load .env file when present — non-fatal so production (env injected
	// via the process environment) works without a .env file.
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found; relying on process environment")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	a, err := app.New(context.Background(), cfg)
	if err != nil {
		log.Fatalf("startup error: %v", err)
	}

	if err := a.Run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

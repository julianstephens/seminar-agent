package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Seminar module
	// Tutorial module
	// Shared infrastructure

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/julianstephens/formation/internal/agent"
	"github.com/julianstephens/formation/internal/agent/providers"
	"github.com/julianstephens/formation/internal/auth"
	"github.com/julianstephens/formation/internal/config"
	"github.com/julianstephens/formation/internal/db"
	apphttp "github.com/julianstephens/formation/internal/http"
	"github.com/julianstephens/formation/internal/http/handlers"
	seminarHandlers "github.com/julianstephens/formation/internal/modules/seminar/handlers"
	seminarRepo "github.com/julianstephens/formation/internal/modules/seminar/repo"
	seminarService "github.com/julianstephens/formation/internal/modules/seminar/service"
	tutorialHandlers "github.com/julianstephens/formation/internal/modules/tutorial/handlers"
	tutorialRepo "github.com/julianstephens/formation/internal/modules/tutorial/repo"
	tutorialService "github.com/julianstephens/formation/internal/modules/tutorial/service"
	"github.com/julianstephens/formation/internal/observability"
	sharedRepo "github.com/julianstephens/formation/internal/repo"
	"github.com/julianstephens/formation/internal/scheduler"
	"github.com/julianstephens/formation/internal/service"
	"github.com/julianstephens/formation/internal/sse"
	"github.com/julianstephens/formation/internal/storage"
)

// App is the top-level application container.
// It owns the HTTP server, DB pool, Scheduler, and SSE hub.
type App struct {
	Config    *config.Config
	DB        *pgxpool.Pool
	JWKS      *auth.JWKS
	Scheduler *scheduler.Scheduler
	Hub       *sse.Hub
	server    *http.Server
}

// New constructs the App: runs DB migrations then wires the HTTP router.
// It returns an error if the database is unreachable or migration fails.
func New(ctx context.Context, cfg *config.Config) (*App, error) {
	logger := observability.NewLogger(cfg.Env)
	slog.SetDefault(logger)

	// 1. Open connection pool and verify connectivity.
	pool, err := db.Open(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// 2. Run pending migrations (idempotent).
	if err := db.Migrate(cfg); err != nil {
		pool.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	logger.Info("database ready")

	// 3. Fetch and warm Auth0 JWKS (starts background refresh goroutine).
	jwks, err := auth.NewJWKS(ctx, cfg)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("init jwks: %w", err)
	}

	logger.Info("jwks ready")

	// 4. Wire repositories, services, and handlers.
	base := sharedRepo.Base{Pool: pool}

	// Seminar module wiring
	semRepo := seminarRepo.NewSeminarRepo(base)
	semSvc := seminarService.NewSeminarService(semRepo)
	seminarHandler := seminarHandlers.NewSeminarHandler(semSvc)

	sessRepo := seminarRepo.NewSessionRepo(base)
	sessSvc := seminarService.NewSessionService(sessRepo, semRepo)
	sessionHandler := seminarHandlers.NewSessionHandler(sessSvc)

	// 5. Start the phase scheduler and recover any in-progress sessions.
	sched := scheduler.New(sessRepo, logger)
	if err := sched.RecoverInProgress(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("scheduler recovery: %w", err)
	}
	logger.Info("scheduler ready")

	// 6. Create the SSE hub and connect it to the scheduler.
	hub := sse.New(logger)
	sched.SetOnPhaseChanged(hub.PublishPhaseChanged)
	logger.Info("sse hub ready")

	// 7. Build the prompt assembler (parses embedded YAML files once at startup).
	assembler, err := agent.NewAssembler()
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("init prompt assembler: %w", err)
	}
	logger.Info("prompt assembler ready")

	// 7b. Build the tutorial prompt assembler.
	tutorialAssembler, err := agent.NewTutorialAssembler()
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("init tutorial prompt assembler: %w", err)
	}
	logger.Info("tutorial prompt assembler ready")

	// 8. Create the OpenAI provider and turn service.
	openaiProvider := providers.New(cfg.OpenAIAPIKey, cfg.OpenAIModel)
	turnSvc := seminarService.NewTurnService(sessRepo, semRepo, assembler, hub, openaiProvider)
	turnHandler := seminarHandlers.NewTurnHandler(turnSvc)

	eventsHandler := seminarHandlers.NewEventsHandler(hub, sessSvc)

	// 11. Wire tutorial repositories, services, and handlers.
	tutRepo := tutorialRepo.NewTutorialRepo(base)
	diagnosticLedgerSvc := tutorialService.NewDiagnosticLedgerService(tutRepo)
	tutorialSvc := tutorialService.NewTutorialService(tutRepo)
	tutorialSessionSvc := tutorialService.NewTutorialSessionService(tutRepo)
	artifactSvc := tutorialService.NewArtifactService(tutRepo)
	tutorialTurnSvc := tutorialService.NewTutorialTurnService(
		tutRepo,
		tutorialAssembler,
		hub,
		openaiProvider,
		diagnosticLedgerSvc,
		cfg.EnableStreaming,
	)
	tutorialHandler := tutorialHandlers.NewTutorialHandler(tutorialSvc, tutorialSessionSvc)
	tutorialSessionHandler := tutorialHandlers.NewTutorialSessionHandler(
		tutorialSessionSvc,
		artifactSvc,
		tutorialTurnSvc,
	)
	tutorialSessionEventsHandler := tutorialHandlers.NewTutorialSessionEventsHandler(hub, tutorialSessionSvc)
	tutorialDiagnosticsHandler := tutorialHandlers.NewTutorialDiagnosticsHandler(diagnosticLedgerSvc, tutRepo)

	// 10. Create the export service and handler using module repos.
	exportSvc := service.NewExportService(semRepo, sessRepo, tutRepo).
		WithS3(storage.NewS3Client(cfg), logger)
	exportHandler := handlers.NewExportHandler(exportSvc)

	// 9. Build HTTP server.
	router := apphttp.NewRouter(apphttp.RouterDeps{
		Config:                cfg,
		JWKS:                  jwks,
		Logger:                logger,
		Seminars:              seminarHandler,
		Sessions:              sessionHandler,
		Events:                eventsHandler,
		Turns:                 turnHandler,
		Exports:               exportHandler,
		Tutorials:             tutorialHandler,
		TutorialSessions:      tutorialSessionHandler,
		TutorialSessionEvents: tutorialSessionEventsHandler,
		TutorialDiagnostics:   tutorialDiagnosticsHandler,
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // disable write timeout to allow indefinite SSE streams
		IdleTimeout:  120 * time.Second,
	}

	return &App{
		Config:    cfg,
		DB:        pool,
		JWKS:      jwks,
		Scheduler: sched,
		Hub:       hub,
		server:    srv,
	}, nil
}

// Run starts the HTTP server and blocks until SIGINT/SIGTERM is received,
// then performs a graceful shutdown with a 30-second deadline.
func (a *App) Run() error {
	// Start server in background
	l := slog.Default()

	// Start server in background
	errCh := make(chan error, 1)
	go func() {
		l.Info("server listening", "addr", a.server.Addr, "env", a.Config.Env)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// Wait for OS signal or server error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		l.Info("shutdown signal received", "signal", sig.String())
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	l.Info("server stopped cleanly")

	// Release DB pool after HTTP connections are all drained.
	a.DB.Close()
	l.Info("database pool closed")

	// Stop phase-advance timers.
	a.Scheduler.Stop()
	l.Info("scheduler stopped")

	// Stop JWKS background refresh goroutine.
	a.JWKS.Stop()
	l.Info("jwks stopped")

	return nil
}

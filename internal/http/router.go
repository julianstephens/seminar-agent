package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/julianstephens/formation/internal/auth"
	"github.com/julianstephens/formation/internal/config"
	"github.com/julianstephens/formation/internal/observability"
)

// RouteRegistrar is implemented by any handler group that can mount its routes
// onto a Gin router group. Using an interface here breaks the import cycle that
// would arise if internal/http imported internal/http/handlers directly.
type RouteRegistrar interface {
	Register(rg *gin.RouterGroup)
}

// SeminarSessionRouteRegistrar extends RouteRegistrar with a second mount-point used
// to register the session-creation route nested under /seminars/:id.
type SeminarSessionRouteRegistrar interface {
	RouteRegistrar
	RegisterUnderSeminar(rg *gin.RouterGroup)
}

// ExportRouteRegistrar knows how to mount export routes under both the
// seminars and sessions resource groups.
type ExportRouteRegistrar interface {
	RegisterUnderSeminars(rg *gin.RouterGroup)
	RegisterUnderSessions(rg *gin.RouterGroup)
	RegisterUnderTutorials(rg *gin.RouterGroup)
	RegisterUnderTutorialSessions(rg *gin.RouterGroup)
	RegisterUnderProblemSets(rg *gin.RouterGroup)
}

// TutorialRouteRegistrar extends RouteRegistrar with a second mount-point used
// to register the session sub-routes nested under /tutorials/:id.
type TutorialRouteRegistrar interface {
	RouteRegistrar
	RegisterSessionsUnderTutorial(rg *gin.RouterGroup)
}

// RouterDeps holds all handler dependencies injected into the router.
// Adding a new handler in a later phase only requires extending this struct
// and wiring the new group below.
type RouterDeps struct {
	Config                *config.Config
	JWKS                  *auth.JWKS
	Logger                *slog.Logger // optional; falls back to slog.Default()
	Seminars              RouteRegistrar
	Sessions              SeminarSessionRouteRegistrar
	Events                RouteRegistrar
	Turns                 RouteRegistrar
	Exports               ExportRouteRegistrar
	Tutorials             TutorialRouteRegistrar
	TutorialSessions      RouteRegistrar
	TutorialSessionEvents RouteRegistrar
	TutorialDiagnostics   RouteRegistrar
}

// NewRouter builds and returns the configured Gin engine.
func NewRouter(deps RouterDeps) *gin.Engine {
	if deps.Config.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	// Structured request logging with per-request ID.
	baseLogger := deps.Logger
	if baseLogger == nil {
		baseLogger = slog.Default()
	}
	r.Use(observability.RequestIDMiddleware(baseLogger))

	// CORS — tighten origins in production via env
	r.Use(cors.New(cors.Config{
		AllowOrigins:     deps.Config.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	baseLogger.Debug(fmt.Sprintf("allowed origins: %s", deps.Config.AllowedOrigins))

	// ── Health ────────────────────────────────────────────────────────────────
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().UTC()})
	})

	// ── Metrics ───────────────────────────────────────────────────────────────
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// ── API v1 ────────────────────────────────────────────────────────────────
	v1 := r.Group("/v1")
	v1.Use(auth.JWTMiddleware(deps.Config, deps.JWKS))

	// Seminars
	seminarsGroup := v1.Group("/seminars")
	deps.Seminars.Register(seminarsGroup)
	// Seminar export.
	if deps.Exports != nil {
		deps.Exports.RegisterUnderSeminars(seminarsGroup)
	} else {
		seminarsGroup.GET("/:id/export", placeholder("export seminar"))
	}
	// Session creation is nested under seminars.
	if deps.Sessions != nil {
		deps.Sessions.RegisterUnderSeminar(seminarsGroup)
	} else {
		seminarsGroup.POST("/:id/sessions", placeholder("create session"))
	}

	// Sessions (top-level operations)
	seminarSessionsGroup := v1.Group("/seminar-sessions")
	if deps.Sessions != nil {
		deps.Sessions.Register(seminarSessionsGroup)
	}
	// Turns (submit a user turn and receive the agent response).
	if deps.Turns != nil {
		deps.Turns.Register(seminarSessionsGroup)
	} else {
		seminarSessionsGroup.POST("/:id/turns", placeholder("submit turn"))
	}
	// Session export.
	if deps.Exports != nil {
		deps.Exports.RegisterUnderSessions(seminarSessionsGroup)
	} else {
		seminarSessionsGroup.GET("/:id/export", placeholder("export session"))
	}
	// SSE event stream.
	if deps.Events != nil {
		deps.Events.Register(seminarSessionsGroup)
	} else {
		seminarSessionsGroup.GET("/:id/events", placeholder("SSE stream"))
	}

	// ── Tutorials ─────────────────────────────────────────────────────────────
	tutorialsGroup := v1.Group("/tutorials")
	if deps.Tutorials != nil {
		deps.Tutorials.Register(tutorialsGroup)
		deps.Tutorials.RegisterSessionsUnderTutorial(tutorialsGroup)
	} else {
		tutorialsGroup.GET("", placeholder("list tutorials"))
		tutorialsGroup.POST("", placeholder("create tutorial"))
		tutorialsGroup.GET("/:id", placeholder("get tutorial"))
		tutorialsGroup.PATCH("/:id", placeholder("update tutorial"))
		tutorialsGroup.DELETE("/:id", placeholder("delete tutorial"))
		tutorialsGroup.GET("/:id/sessions", placeholder("list tutorial sessions"))
		tutorialsGroup.POST("/:id/sessions", placeholder("create tutorial session"))
	}
	// Tutorial export.
	if deps.Exports != nil {
		deps.Exports.RegisterUnderTutorials(tutorialsGroup)
	} else {
		tutorialsGroup.GET("/:id/export", placeholder("export tutorial"))
	}
	// Tutorial diagnostics and problem sets
	if deps.TutorialDiagnostics != nil {
		deps.TutorialDiagnostics.Register(tutorialsGroup)
	} else {
		tutorialsGroup.GET("/:id/diagnostics", placeholder("list diagnostics"))
		tutorialsGroup.GET("/:id/diagnostics/summary", placeholder("get diagnostic summary"))
		tutorialsGroup.GET("/:id/problem-sets", placeholder("list problem sets"))
	}

	// Tutorial Sessions (top-level operations)
	tutorialSessionsGroup := v1.Group("/tutorial-sessions")
	if deps.TutorialSessions != nil {
		deps.TutorialSessions.Register(tutorialSessionsGroup)
	} else {
		tutorialSessionsGroup.GET("/:id", placeholder("get tutorial session"))
		tutorialSessionsGroup.DELETE("/:id", placeholder("delete tutorial session"))
		tutorialSessionsGroup.POST("/:id/complete", placeholder("complete tutorial session"))
		tutorialSessionsGroup.POST("/:id/abandon", placeholder("abandon tutorial session"))
		tutorialSessionsGroup.GET("/:id/artifacts", placeholder("list artifacts"))
		tutorialSessionsGroup.POST("/:id/artifacts", placeholder("create artifact"))
		tutorialSessionsGroup.DELETE("/:id/artifacts/:artifactId", placeholder("delete artifact"))
	}
	// Tutorial Session export.
	if deps.Exports != nil {
		deps.Exports.RegisterUnderTutorialSessions(tutorialSessionsGroup)
	} else {
		tutorialSessionsGroup.GET("/:id/export", placeholder("export tutorial session"))
	}
	// Tutorial Session SSE stream.
	if deps.TutorialSessionEvents != nil {
		deps.TutorialSessionEvents.Register(tutorialSessionsGroup)
	} else {
		tutorialSessionsGroup.GET("/:id/events", placeholder("SSE stream"))
	}

	// ── Problem Sets ──────────────────────────────────────────────────────────
	problemSetsGroup := v1.Group("/problem-sets")
	// Problem Set export.
	if deps.Exports != nil {
		deps.Exports.RegisterUnderProblemSets(problemSetsGroup)
	} else {
		problemSetsGroup.GET("/:id/export", placeholder("export problem set"))
	}

	return r
}

// placeholder returns a stub handler that responds 501 until a route is implemented.
func placeholder(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error":   "not_implemented",
			"message": name + " not yet implemented",
		})
	}
}

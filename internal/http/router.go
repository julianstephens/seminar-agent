package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

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

// SessionRouteRegistrar extends RouteRegistrar with a second mount-point used
// to register the session-creation route nested under /seminars/:id.
type SessionRouteRegistrar interface {
	RouteRegistrar
	RegisterUnderSeminar(rg *gin.RouterGroup)
}

// ExportRouteRegistrar knows how to mount export routes under both the
// seminars and sessions resource groups.
type ExportRouteRegistrar interface {
	RegisterUnderSeminars(rg *gin.RouterGroup)
	RegisterUnderSessions(rg *gin.RouterGroup)
}

// RouterDeps holds all handler dependencies injected into the router.
// Adding a new handler in a later phase only requires extending this struct
// and wiring the new group below.
type RouterDeps struct {
	Config   *config.Config
	JWKS     *auth.JWKS
	Logger   *slog.Logger // optional; falls back to slog.Default()
	Seminars RouteRegistrar
	Sessions SessionRouteRegistrar
	Events   RouteRegistrar
	Turns    RouteRegistrar
	Exports  ExportRouteRegistrar
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
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// ── Health ────────────────────────────────────────────────────────────────
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().UTC()})
	})

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
	sessionsGroup := v1.Group("/sessions")
	if deps.Sessions != nil {
		deps.Sessions.Register(sessionsGroup)
	}
	// Turns (submit a user turn and receive the agent response).
	if deps.Turns != nil {
		deps.Turns.Register(sessionsGroup)
	} else {
		sessionsGroup.POST("/:id/turns", placeholder("submit turn"))
	}
	// Session export.
	if deps.Exports != nil {
		deps.Exports.RegisterUnderSessions(sessionsGroup)
	} else {
		sessionsGroup.GET("/:id/export", placeholder("export session"))
	}
	// SSE event stream.
	if deps.Events != nil {
		deps.Events.Register(sessionsGroup)
	} else {
		sessionsGroup.GET("/:id/events", placeholder("SSE stream"))
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

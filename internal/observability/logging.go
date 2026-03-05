// Package observability provides structured logging helpers and HTTP middleware
// for the Seminar backend.
//
// Design
// ──────
// Every incoming HTTP request is assigned a unique request ID (8-byte random
// hex string). The ID is:
//   - returned to the caller in the X-Request-ID response header
//   - embedded in all slog log entries emitted during the request via a child
//     logger stored in the Gin context
//   - available from a standard library context for use in non-Gin code paths
//     (e.g. service/repo calls made from a handler)
//
// NewLogger constructs the application's root JSON slog.Logger. Pass it to
// the scheduler, SSE hub, and any other component that holds a *slog.Logger.
// RequestIDMiddleware mints per-request loggers from this root logger.
package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// ── Key types ─────────────────────────────────────────────────────────────────

// ginLoggerKey is the key under which the per-request *slog.Logger is stored
// in the Gin context.
const ginLoggerKey = "formation_req_logger"

// ctxLoggerKey is the key used to propagate a *slog.Logger through a standard
// library context.Context (e.g. when a handler passes ctx into a service).
type ctxLoggerKey struct{}

// ── Constructor ───────────────────────────────────────────────────────────────

// NewLogger builds the application root logger.
//
//   - env == "production"  →  JSON output, Info level
//   - anything else        →  JSON output, Debug level
//
// Pass the result to New(app.go) and hand it to every component that needs
// a *slog.Logger (scheduler, SSE hub, etc.).
func NewLogger(env string) *slog.Logger {
	level := slog.LevelDebug
	if env == "production" {
		level = slog.LevelInfo
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: env == "production",
	})
	return slog.New(h)
}

// NewTestLogger returns a logger that discards all output. Use it in unit
// tests wherever a *slog.Logger is required.
func NewTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// ── Gin middleware ────────────────────────────────────────────────────────────

// RequestIDMiddleware is a Gin middleware that:
//  1. Generates a random 16-hex-character request ID.
//  2. Writes it to the X-Request-ID response header.
//  3. Stores a child logger (root logger + request_id attribute) in the Gin
//     context under ginLoggerKey.
//  4. After the handler chain completes, logs a structured "request completed"
//     entry with method, path, status, latency, and client IP.
//
// Usage:
//
//	r.Use(observability.RequestIDMiddleware(logger))
func RequestIDMiddleware(base *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := newRequestID()
		c.Header("X-Request-ID", reqID)

		// Build a child logger for this request.
		reqLogger := base.With(
			slog.String("request_id", reqID),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
		)
		c.Set(ginLoggerKey, reqLogger)

		// Propagate into the standard context so service/repo layers can log
		// with the same request_id without importing Gin.
		ctx := WithLogger(c.Request.Context(), reqLogger)
		c.Request = c.Request.WithContext(ctx)

		start := time.Now()
		c.Next()

		reqLogger.Info("request completed",
			slog.Int("status", c.Writer.Status()),
			slog.Int64("latency_ms", time.Since(start).Milliseconds()),
			slog.String("client_ip", c.ClientIP()),
		)
	}
}

// ── Gin context helpers ───────────────────────────────────────────────────────

// FromGinCtx returns the per-request logger stored by RequestIDMiddleware.
// Falls back to slog.Default() when no logger is present (e.g. in tests that
// bypass the middleware).
func FromGinCtx(c *gin.Context) *slog.Logger {
	v, ok := c.Get(ginLoggerKey)
	if !ok {
		return slog.Default()
	}
	l, ok := v.(*slog.Logger)
	if !ok {
		return slog.Default()
	}
	return l
}

// ── stdlib context helpers ────────────────────────────────────────────────────

// WithLogger returns a copy of ctx that carries l. The logger can be retrieved
// later with LoggerFromContext.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxLoggerKey{}, l)
}

// LoggerFromContext retrieves the logger stored by WithLogger.
// Returns slog.Default() when no logger is found.
func LoggerFromContext(ctx context.Context) *slog.Logger {
	l, ok := ctx.Value(ctxLoggerKey{}).(*slog.Logger)
	if !ok || l == nil {
		return slog.Default()
	}
	return l
}

// ── internal ──────────────────────────────────────────────────────────────────

// newRequestID generates a cryptographically random 16-hex-character string.
// In the unlikely event that rand.Read fails it falls back to a timestamp-based
// ID so the application never blocks.
func newRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Extremely unlikely on any modern OS; use timestamp as fallback.
		return fmt.Sprintf("%016x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

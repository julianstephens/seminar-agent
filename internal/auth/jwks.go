// Package auth provides Auth0 JWT validation and request context helpers.
package auth

import (
	"context"
	"fmt"

	"github.com/MicahParks/keyfunc/v3"

	"github.com/julianstephens/formation/internal/config"
)

// JWKS wraps a keyfunc.Keyfunc and provides a Stop method to cancel its
// background refresh goroutine. Call Stop during graceful shutdown.
type JWKS struct {
	inner keyfunc.Keyfunc
	stop  context.CancelFunc
}

// NewJWKS creates a JWKS client that fetches and automatically refreshes the
// Auth0 JWKS keyset. It starts a background goroutine; call Stop() to cancel it.
func NewJWKS(parentCtx context.Context, cfg *config.Config) (*JWKS, error) {
	jwksURL := fmt.Sprintf("https://%s/.well-known/jwks.json", cfg.Auth0Domain)

	// A cancellable context controls the background refresh goroutine lifetime.
	refreshCtx, cancel := context.WithCancel(parentCtx)

	k, err := keyfunc.NewDefaultCtx(refreshCtx, []string{jwksURL})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("fetch JWKS from %s: %w", jwksURL, err)
	}

	return &JWKS{inner: k, stop: cancel}, nil
}

// Stop cancels the background JWKS refresh goroutine.
func (j *JWKS) Stop() {
	j.stop()
}

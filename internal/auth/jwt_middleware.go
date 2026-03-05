package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/julianstephens/formation/internal/config"
)

// Claims are the JWT payload fields we care about.
// RegisteredClaims carries Subject (sub), Issuer (iss), Audience (aud), ExpiresAt, etc.
type Claims struct {
	jwt.RegisteredClaims
}

// JWTMiddleware returns a Gin middleware that:
//  1. Requires a valid Bearer token in the Authorization header.
//  2. Validates signature via the JWKS key set.
//  3. Enforces issuer (https://{domain}/) and audience.
//  4. Stores the Auth0 sub in the Gin context (use OwnerSubFromCtx to retrieve).
func JWTMiddleware(cfg *config.Config, jwks *JWKS) gin.HandlerFunc {
	expectedIssuer := fmt.Sprintf("https://%s/", cfg.Auth0Domain)
	expectedAudience := cfg.Auth0Audience

	return func(c *gin.Context) {
		raw, err := extractBearerToken(c.GetHeader("Authorization"))
		if err != nil {
			abortUnauthorized(c, err.Error())
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(raw, claims, jwks.inner.Keyfunc,
			jwt.WithIssuedAt(),
			jwt.WithExpirationRequired(),
		)
		if err != nil || !token.Valid {
			abortUnauthorized(c, "invalid token")
			return
		}

		// Validate issuer.
		if iss, _ := claims.GetIssuer(); iss != expectedIssuer {
			abortUnauthorized(c, "invalid issuer")
			return
		}

		// Validate audience — token aud may be a string or array.
		aud, _ := claims.GetAudience()
		if !containsAudience(aud, expectedAudience) {
			abortUnauthorized(c, "invalid audience")
			return
		}

		// Validate sub is non-empty.
		sub, _ := claims.GetSubject()
		if sub == "" {
			abortUnauthorized(c, "missing sub claim")
			return
		}

		setOwnerSub(c, sub)
		c.Next()
	}
}

// extractBearerToken parses "Bearer <token>" and returns the raw token string.
func extractBearerToken(header string) (string, error) {
	if header == "" {
		return "", errors.New("missing Authorization header")
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", errors.New("Authorization header must be 'Bearer <token>'")
	}
	return parts[1], nil
}

func containsAudience(aud jwt.ClaimStrings, target string) bool {
	for _, a := range aud {
		if a == target {
			return true
		}
	}
	return false
}

func abortUnauthorized(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error":   "unauthorized",
		"message": msg,
	})
}

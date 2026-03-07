package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	// Server
	Port string
	Env  string // "development" | "production"

	// Postgres
	DatabaseURL string

	// Auth0
	Auth0Domain   string
	Auth0Audience string

	// OpenAI
	OpenAIAPIKey    string
	OpenAIModel     string
	EnableStreaming bool

	// SSE
	TimerTickSeconds int

	// S3 (Garage / AWS compatible)
	S3EndpointURL string
	S3AccessKeyID string
	S3SecretKey   string
	S3BucketName  string
	S3Region      string
}

// Load reads configuration from environment variables, returning an error
// if any required variable is missing.
func Load() (*Config, error) {
	cfg := &Config{
		Port:             getEnv("PORT", "8080"),
		Env:              getEnv("APP_ENV", "development"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		Auth0Domain:      os.Getenv("AUTH0_DOMAIN"),
		Auth0Audience:    os.Getenv("AUTH0_AUDIENCE"),
		OpenAIAPIKey:     os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:      getEnv("OPENAI_MODEL", "gpt-4o"),
		EnableStreaming:  getEnv("ENABLE_STREAMING", true),
		TimerTickSeconds: getEnv("TIMER_TICK_SECONDS", 2),
		S3EndpointURL:    os.Getenv("S3_ENDPOINT_URL"),
		S3AccessKeyID:    os.Getenv("S3_ACCESS_KEY_ID"),
		S3SecretKey:      os.Getenv("S3_SECRET_KEY"),
		S3BucketName:     os.Getenv("S3_BUCKET_NAME"),
		S3Region:         getEnv("S3_REGION", "us-east-1"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	required := map[string]string{
		"DATABASE_URL":     c.DatabaseURL,
		"AUTH0_DOMAIN":     c.Auth0Domain,
		"AUTH0_AUDIENCE":   c.Auth0Audience,
		"OPENAI_API_KEY":   c.OpenAIAPIKey,
		"S3_ENDPOINT_URL":  c.S3EndpointURL,
		"S3_ACCESS_KEY_ID": c.S3AccessKeyID,
		"S3_SECRET_KEY":    c.S3SecretKey,
		"S3_BUCKET_NAME":   c.S3BucketName,
	}

	var missing []string
	for k, v := range required {
		if v == "" {
			missing = append(missing, k)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

func getEnv[T any](key string, fallback T) T {
	if v := os.Getenv(key); v != "" {
		switch any(fallback).(type) {
		case string:
			return any(v).(T)
		case int:
			if i, err := strconv.Atoi(v); err == nil {
				return any(i).(T)
			}
		case bool:
			if b, err := strconv.ParseBool(v); err == nil {
				return any(b).(T)
			}
		}
	}
	return fallback
}

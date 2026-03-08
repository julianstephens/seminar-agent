package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	infisical "github.com/infisical/go-sdk"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	// Server
	Port           string
	Env            string // "development" | "production"
	AllowedOrigins []string

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
		AllowedOrigins:   strings.Split(getEnv("ALLOWED_ORIGINS", ""), ","),
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

// LoadFromInfisical loads configuration from Infisical secrets.
// It requires the following environment variables to be set for Infisical authentication:
// - APP_ENV (used to determine which environment's secrets to load, e.g. "development" or "production")
// - INFISICAL_CLIENT_ID
// - INFISICAL_CLIENT_SECRET
// - INFISICAL_PROJECT_ID
// - INFISICAL_SECRET_PATH
func LoadFromInfisical() (*Config, error) {
	client := infisical.NewInfisicalClient(context.Background(), infisical.Config{
		SiteUrl:          "https://infisical.cyborgdev.cloud", // Optional, default is https://app.infisical.com
		AutoTokenRefresh: true,                                // Wether or not to let the SDK handle the access token lifecycle. Defaults to true if not specified.
	})

	client_id := os.Getenv("INFISICAL_CLIENT_ID")
	client_secret := os.Getenv("INFISICAL_CLIENT_SECRET")
	if client_id == "" || client_secret == "" {
		return nil, fmt.Errorf(
			"infisical: INFISICAL_CLIENT_ID and INFISICAL_CLIENT_SECRET environment variables are required",
		)
	}
	if _, err := client.Auth().UniversalAuthLogin(client_id, client_secret); err != nil {
		return nil, fmt.Errorf("infisical auth: %w", err)
	}

	secret_path := os.Getenv("INFISICAL_SECRET_PATH")
	if secret_path == "" {
		return nil, fmt.Errorf("infisical: INFISICAL_SECRET_PATH environment variable is required")
	}

	project_id := os.Getenv("INFISICAL_PROJECT_ID")
	if project_id == "" {
		return nil, fmt.Errorf("infisical: INFISICAL_PROJECT_ID environment variable is required")
	}
	secrets, err := client.Secrets().List(infisical.ListSecretsOptions{
		ProjectID:          project_id,
		Environment:        getEnv("APP_ENV", "dev"),
		SecretPath:         secret_path,
		AttachToProcessEnv: false,
	})
	if err != nil {
		return nil, fmt.Errorf("infisical list secrets: %w", err)
	}

	var cfg Config
	for _, s := range secrets {
		switch s.SecretKey {
		case "PORT":
			cfg.Port = s.SecretValue
		case "APP_ENV":
			cfg.Env = s.SecretValue
		case "ALLOWED_ORIGINS":
			cfg.AllowedOrigins = strings.Split(s.SecretValue, ",")
		case "DATABASE_URL":
			cfg.DatabaseURL = s.SecretValue
		case "AUTH0_DOMAIN":
			cfg.Auth0Domain = s.SecretValue
		case "AUTH0_AUDIENCE":
			cfg.Auth0Audience = s.SecretValue
		case "OPENAI_API_KEY":
			cfg.OpenAIAPIKey = s.SecretValue
		case "OPENAI_MODEL":
			cfg.OpenAIModel = s.SecretValue
		case "ENABLE_STREAMING":
			cfg.EnableStreaming, _ = strconv.ParseBool(s.SecretValue)
		case "TIMER_TICK_SECONDS":
			cfg.TimerTickSeconds, _ = strconv.Atoi(s.SecretValue)
		case "S3_ENDPOINT_URL":
			cfg.S3EndpointURL = s.SecretValue
		case "S3_ACCESS_KEY_ID":
			cfg.S3AccessKeyID = s.SecretValue
		case "S3_SECRET_KEY":
			cfg.S3SecretKey = s.SecretValue
		case "S3_BUCKET_NAME":
			cfg.S3BucketName = s.SecretValue
		case "S3_REGION":
			cfg.S3Region = s.SecretValue
		}
	}
	return &cfg, nil
}

func (c *Config) validate() error {
	required := map[string]string{
		"PORT":             c.Port,
		"APP_ENV":          c.Env,
		"ALLOWED_ORIGINS":  strings.Join(c.AllowedOrigins, ","),
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

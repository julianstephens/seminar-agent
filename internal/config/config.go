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
	Env            string // "dev" | "prod"
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

	// Redis
	RedisUser     string
	RedisPassword string
	RedisPort     int
	RedisHost     string
	RedisPrefix   string
}

// LoadFromInfisical loads configuration from Infisical secrets.
// It requires the following environment variables to be set for Infisical authentication:
// - APP_ENV (used to determine which environment's secrets to load, e.g. "dev" or "prod")
// - INFISICAL_CLIENT_ID
// - INFISICAL_CLIENT_SECRET
// - INFISICAL_PROJECT_ID
// - INFISICAL_SECRET_PATH
func LoadFromInfisical() (*Config, error) {
	client := infisical.NewInfisicalClient(context.Background(), infisical.Config{
		SiteUrl:          "https://infisical.cyborgdev.cloud",
		AutoTokenRefresh: true,
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

	// Initialize config with default values
	cfg := &Config{
		Port:             "8080",
		Env:              "dev",
		OpenAIModel:      "gpt-5",
		EnableStreaming:  true,
		TimerTickSeconds: 2,
		S3Region:         "us-east-1",
		RedisPrefix:      "formation:",
	}

	// Override defaults with values from Infisical
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
			cfg.EnableStreaming, err = strconv.ParseBool(s.SecretValue)
			if err != nil {
				return nil, fmt.Errorf("invalid ENABLE_STREAMING: %w", err)
			}
		case "TIMER_TICK_SECONDS":
			cfg.TimerTickSeconds, err = strconv.Atoi(s.SecretValue)
			if err != nil {
				return nil, fmt.Errorf("invalid TIMER_TICK_SECONDS: %w", err)
			}
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
		case "REDIS_USER":
			cfg.RedisUser = s.SecretValue
		case "REDIS_PASSWORD":
			cfg.RedisPassword = s.SecretValue
		case "REDIS_PORT":
			cfg.RedisPort, err = strconv.Atoi(s.SecretValue)
			if err != nil {
				return nil, fmt.Errorf("invalid REDIS_PORT: %w", err)
			}
		case "REDIS_HOST":
			cfg.RedisHost = s.SecretValue
		case "REDIS_PREFIX":
			cfg.RedisPrefix = s.SecretValue
		}
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	fmt.Println("Configuration loaded successfully from Infisical")
	fmt.Printf("%+v\n", cfg)

	return cfg, nil
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
		"REDIS_USER":       c.RedisUser,
		"REDIS_PASSWORD":   c.RedisPassword,
		"REDIS_PORT":       strconv.Itoa(c.RedisPort),
		"REDIS_HOST":       c.RedisHost,
		"REDIS_PREFIX":     c.RedisPrefix,
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

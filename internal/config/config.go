package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr          string
	DatabaseURL       string
	MigrationDBURL    string
	RedisURL          string
	JWTSecret         string
	JWTTTL            time.Duration
	CORSOrigins       []string
	AdminEmail        string
	AdminPassword     string
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
}

func Load() (Config, error) {
	return load(true)
}

// LoadDatabase loads configuration for migration and seed commands, which do
// not need the HTTP server's JWT secret.
func LoadDatabase() (Config, error) {
	return load(false)
}

func load(requireJWT bool) (Config, error) {
	cfg := Config{
		HTTPAddr:       env("HTTP_ADDR", ":8080"),
		DatabaseURL:    strings.TrimSpace(os.Getenv("DATABASE_URL")),
		MigrationDBURL: strings.TrimSpace(os.Getenv("MIGRATION_DATABASE_URL")),
		RedisURL:       strings.TrimSpace(env("REDIS_URL", "redis://localhost:6379/0")),
		JWTSecret:      strings.TrimSpace(os.Getenv("JWT_SECRET")),
		AdminEmail:     strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))),
		AdminPassword:  strings.TrimSpace(os.Getenv("ADMIN_PASSWORD")),
	}
	var problems []string
	var err error
	if cfg.DBMaxOpenConns, err = envInt("DB_MAX_OPEN_CONNS", 10); err != nil {
		problems = append(problems, err.Error())
	}
	if cfg.DBMaxIdleConns, err = envInt("DB_MAX_IDLE_CONNS", 5); err != nil {
		problems = append(problems, err.Error())
	}
	if cfg.DBConnMaxLifetime, err = envDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute); err != nil {
		problems = append(problems, err.Error())
	}
	if cfg.JWTTTL, err = envDuration("JWT_TTL", time.Hour); err != nil {
		problems = append(problems, err.Error())
	}
	for _, origin := range strings.Split(env("CORS_ORIGINS", "http://localhost:3000"), ",") {
		if value := strings.TrimSpace(origin); value != "" {
			cfg.CORSOrigins = append(cfg.CORSOrigins, value)
		}
	}
	if strings.ReplaceAll(cfg.DatabaseURL, " ", "") == "" {
		problems = append(problems, "DATABASE_URL is required")
	}
	if requireJWT && len(cfg.JWTSecret) < 32 {
		problems = append(problems, "JWT_SECRET must contain at least 32 characters")
	}
	if requireJWT && cfg.JWTTTL <= 0 {
		problems = append(problems, "JWT_TTL must be positive")
	}
	if len(problems) > 0 {
		return Config{}, errors.New(strings.Join(problems, "; "))
	}
	return cfg, nil
}

func (c Config) MigrationURL() string {
	if c.MigrationDBURL != "" {
		return c.MigrationDBURL
	}
	return c.DatabaseURL
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", key)
	}
	return n, nil
}

func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid Go duration", key)
	}
	return d, nil
}

func (c Config) String() string {
	return fmt.Sprintf("addr=%s", c.HTTPAddr)
}

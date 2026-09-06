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
	Environment       string
	HTTPAddr          string
	DatabaseURL       string
	MigrationDBURL    string
	JWTSecret         string
	JWTTTL            time.Duration
	CORSOrigins       []string
	AdminEmail        string
	AdminPassword     string
	AWSRegion         string
	S3Bucket          string
	S3PublicBaseURL   string
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
		Environment:     env("APP_ENV", "development"),
		HTTPAddr:        env("HTTP_ADDR", ":8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		MigrationDBURL:  os.Getenv("MIGRATION_DATABASE_URL"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		AdminEmail:      strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))),
		AdminPassword:   os.Getenv("ADMIN_PASSWORD"),
		AWSRegion:       env("AWS_REGION", "ap-northeast-2"),
		S3Bucket:        os.Getenv("S3_BUCKET"),
		S3PublicBaseURL: strings.TrimRight(os.Getenv("S3_PUBLIC_BASE_URL"), "/"),
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
	if cfg.DatabaseURL == "" {
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

func (c Config) UploadsEnabled() bool { return c.S3Bucket != "" }

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
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
	return fmt.Sprintf("env=%s addr=%s uploads=%t", c.Environment, c.HTTPAddr, c.UploadsEnabled())
}

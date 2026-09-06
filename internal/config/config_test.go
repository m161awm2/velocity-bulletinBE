package config

import "testing"

func TestLoadRequiresDatabaseAndStrongJWTSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "short")
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid configuration")
	}
}

func TestMigrationURLFallsBackToRuntimeURL(t *testing.T) {
	cfg := Config{DatabaseURL: "runtime", MigrationDBURL: ""}
	if got := cfg.MigrationURL(); got != "runtime" {
		t.Fatalf("got %q", got)
	}
	cfg.MigrationDBURL = "direct"
	if got := cfg.MigrationURL(); got != "direct" {
		t.Fatalf("got %q", got)
	}
}

func TestLoadRejectsMalformedDurations(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://example")
	t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
	t.Setenv("JWT_TTL", "tomorrow")
	if _, err := Load(); err == nil {
		t.Fatal("expected malformed JWT_TTL to be rejected")
	}
}

func TestDatabaseCommandsDoNotRequireJWT(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://example")
	t.Setenv("JWT_SECRET", "")
	if _, err := LoadDatabase(); err != nil {
		t.Fatalf("unexpected database-only configuration error: %v", err)
	}
}

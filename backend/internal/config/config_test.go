package config

import (
	"os"
	"testing"
)

func TestLoadConfig_ProductionRejectsEmptyJWTSecret(t *testing.T) {
	// Unset all relevant env vars, then set production
	os.Unsetenv("JWT_SECRET")
	t.Setenv("APP_ENV", "production")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error when JWT_SECRET is missing in production, got nil")
	}
}

func TestLoadConfig_ProductionRejectsShortJWTSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "short")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error when JWT_SECRET is shorter than MinJWTSecretLength in production")
	}
}

func TestLoadConfig_ProductionRejectsLegacyDefault(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", LegacyInsecureJWTSecret)

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error when JWT_SECRET is the legacy insecure default in production")
	}
}

func TestLoadConfig_ProductionRejectsDevFallback(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", DevFallbackJWTSecret)

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error when JWT_SECRET is the dev fallback in production")
	}
}

func TestLoadConfig_ProductionAcceptsValidSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "a-real-production-secret-that-is-long-enough-yes!")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error with valid JWT_SECRET in production, got: %v", err)
	}
	if cfg.JWTSecret != "a-real-production-secret-that-is-long-enough-yes!" {
		t.Fatalf("expected cfg.JWTSecret to match env var")
	}
}

func TestLoadConfig_DevelopmentFallbackSecret(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	os.Unsetenv("JWT_SECRET")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error in development with fallback, got: %v", err)
	}
	if cfg.JWTSecret != DevFallbackJWTSecret {
		t.Fatalf("expected dev fallback JWT secret, got: %s", cfg.JWTSecret)
	}
}

func TestLoadConfig_AllowedOriginsParsed(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("JWT_SECRET", DevFallbackJWTSecret)
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com, https://admin.example.com")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.AllowedOrigins) != 2 {
		t.Fatalf("expected 2 allowed origins, got %d: %v", len(cfg.AllowedOrigins), cfg.AllowedOrigins)
	}
	if cfg.AllowedOrigins[0] != "https://app.example.com" {
		t.Fatalf("expected first origin https://app.example.com, got %s", cfg.AllowedOrigins[0])
	}
}

func TestLoadConfig_IsProductionHelper(t *testing.T) {
	tests := []struct {
		env    string
		expect bool
	}{
		{"production", true},
		{"Production", true},
		{"prod", true},
		{"PROD", true},
		{"development", false},
		{"staging", false},
	}

	for _, tc := range tests {
		cfg := &Config{AppEnv: tc.env}
		if cfg.IsProduction() != tc.expect {
			t.Errorf("IsProduction(%q) = %v, want %v", tc.env, cfg.IsProduction(), tc.expect)
		}
	}
}

func TestLoadConfig_AdminBootstrap_MismatchedCredentials(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("JWT_SECRET", DevFallbackJWTSecret)

	// Email set, password empty
	t.Setenv("ADMIN_BOOTSTRAP_EMAIL", "admin@example.com")
	os.Unsetenv("ADMIN_BOOTSTRAP_PASSWORD")
	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error when ADMIN_BOOTSTRAP_PASSWORD is unset but email is provided")
	}

	// Password set, email empty
	os.Unsetenv("ADMIN_BOOTSTRAP_EMAIL")
	t.Setenv("ADMIN_BOOTSTRAP_PASSWORD", "some-password-1234")
	_, err = LoadConfig()
	if err == nil {
		t.Fatal("expected error when ADMIN_BOOTSTRAP_EMAIL is unset but password is provided")
	}
}

func TestLoadConfig_AdminBootstrap_ShortPassword(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("JWT_SECRET", DevFallbackJWTSecret)
	t.Setenv("ADMIN_BOOTSTRAP_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_BOOTSTRAP_PASSWORD", "short")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error when ADMIN_BOOTSTRAP_PASSWORD is shorter than 8 chars")
	}
}

func TestLoadConfig_AdminBootstrap_RejectsLegacyPassword(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("JWT_SECRET", DevFallbackJWTSecret)
	t.Setenv("ADMIN_BOOTSTRAP_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_BOOTSTRAP_PASSWORD", LegacyAdminPassword)

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error when ADMIN_BOOTSTRAP_PASSWORD is the legacy default password")
	}
}

func TestLoadConfig_AdminBootstrap_ValidCredentials(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("JWT_SECRET", DevFallbackJWTSecret)
	t.Setenv("ADMIN_BOOTSTRAP_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_BOOTSTRAP_PASSWORD", "a-secure-valid-password-1234")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error with valid admin bootstrap credentials: %v", err)
	}
	if cfg.AdminBootstrapEmail != "admin@example.com" {
		t.Fatalf("expected admin email admin@example.com, got %s", cfg.AdminBootstrapEmail)
	}
	if cfg.AdminBootstrapPassword != "a-secure-valid-password-1234" {
		t.Fatalf("expected admin password to match")
	}
}


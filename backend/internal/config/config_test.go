package config

import (
	"strings"
	"testing"
)

func TestValidateRejectsProductionWithMissingRequiredSecrets(t *testing.T) {
	cfg := Config{AppEnv: "production"}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}

	msg := err.Error()
	for _, want := range []string{"JWT_SECRET", "MYSQL_DSN", "AGENT_INTERNAL_TOKEN"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected error to mention %s, got %q", want, msg)
		}
	}
}

func TestValidateAllowsDevelopmentDefaults(t *testing.T) {
	cfg := Config{
		AppEnv:                   "development",
		AppHost:                  "127.0.0.1",
		AppPort:                  "8080",
		MySQLDSN:                 "ppk_dev:ppk_dev_password@tcp(127.0.0.1:3306)/ppk?charset=utf8mb4&parseTime=True&loc=Local",
		JWTSecret:                "dev-jwt-secret-change-me-32-bytes",
		AgentServiceURL:          "http://127.0.0.1:8090",
		AgentInternalToken:       "dev-agent-internal-token-change-me",
		AllowedOrigins:           []string{"http://localhost:5173"},
		MaxReviewGenerateCount:   50,
		DefaultReviewTargetCount: 10,
		UploadDir:                "./uploads",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected development config to validate, got %v", err)
	}
}

func TestValidateRejectsAllowedOriginWithPath(t *testing.T) {
	cfg := Config{
		AppEnv:                   "production",
		AppHost:                  "127.0.0.1",
		AppPort:                  "8080",
		MySQLDSN:                 "ppk_user:secret@tcp(127.0.0.1:3306)/ppk?charset=utf8mb4&parseTime=True&loc=Local",
		JWTSecret:                "prod-jwt-secret-with-at-least-32-bytes",
		AgentServiceURL:          "http://127.0.0.1:8090",
		AgentInternalToken:       "prod-agent-token-with-at-least-32-bytes",
		AllowedOrigins:           []string{"https://frontend.example.com/app"},
		MaxReviewGenerateCount:   50,
		DefaultReviewTargetCount: 10,
		UploadDir:                "./uploads",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "ALLOWED_ORIGINS") {
		t.Fatalf("expected ALLOWED_ORIGINS error, got %q", err.Error())
	}
}

func TestLoadInvalidIntegerEnvFailsValidation(t *testing.T) {
	t.Setenv("MAX_REVIEW_GENERATE_COUNT", "not-a-number")
	t.Setenv("DEFAULT_REVIEW_TARGET_COUNT", "10")
	t.Setenv("APP_ENV", "development")

	cfg := Load()
	if cfg.MaxReviewGenerateCount != 0 {
		t.Fatalf("got max count %d, want 0 sentinel for invalid env", cfg.MaxReviewGenerateCount)
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid integer env to fail validation")
	}
}

func TestLoadNormalizesPublicBasePath(t *testing.T) {
	t.Setenv("PUBLIC_BASE_PATH", "ppk/")
	t.Setenv("APP_ENV", "development")

	cfg := Load()
	if got, want := cfg.PublicBasePath, "/ppk"; got != want {
		t.Fatalf("got public base path %q, want %q", got, want)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected normalized public base path to validate, got %v", err)
	}
}

func TestValidateRejectsInvalidPublicBasePath(t *testing.T) {
	cfg := Config{
		AppEnv:                   "development",
		AppHost:                  "127.0.0.1",
		AppPort:                  "8080",
		MySQLDSN:                 "ppk_dev:ppk_dev_password@tcp(127.0.0.1:3306)/ppk?charset=utf8mb4&parseTime=True&loc=Local",
		JWTSecret:                "dev-jwt-secret-change-me-32-bytes",
		AgentServiceURL:          "http://127.0.0.1:8090",
		AgentInternalToken:       "dev-agent-internal-token-change-me",
		AllowedOrigins:           []string{"http://localhost:5173"},
		MaxReviewGenerateCount:   50,
		DefaultReviewTargetCount: 10,
		UploadDir:                "./uploads",
		PublicBasePath:           "/bad path",
	}

	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "PUBLIC_BASE_PATH") {
		t.Fatalf("expected PUBLIC_BASE_PATH validation error, got %v", err)
	}
}

func TestListenAddressUsesConfiguredHostAndPort(t *testing.T) {
	cfg := Config{AppHost: "127.0.0.1", AppPort: "18989"}

	if got, want := cfg.ListenAddress(), "127.0.0.1:18989"; got != want {
		t.Fatalf("got listen address %q, want %q", got, want)
	}
}

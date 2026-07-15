package shared

import (
	"strings"
	"testing"
	"time"
)

func validProductionConfig() Config {
	return Config{
		Environment:      "production",
		JWTSecret:        strings.Repeat("u", 32),
		AdminJWTSecret:   strings.Repeat("a", 32),
		JWTExpire:        time.Hour,
		AdminJWTExpire:   time.Hour,
		AdminInitialPass: "a-strong-admin-password",
		AgentTimeout:     20 * time.Second,
	}
}

func TestConfigValidate(t *testing.T) {
	if err := validProductionConfig().Validate(); err != nil {
		t.Fatalf("valid production config rejected: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{"short user secret", func(c *Config) { c.JWTSecret = "short" }},
		{"shared admin secret", func(c *Config) { c.AdminJWTSecret = c.JWTSecret }},
		{"default admin password", func(c *Config) { c.AdminInitialPass = "Admin@123" }},
		{"invalid expiration", func(c *Config) { c.JWTExpire = 0 }},
		{"invalid AI timeout", func(c *Config) { c.AgentTimeout = 0 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validProductionConfig()
			tt.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("invalid config was accepted")
			}
		})
	}
}

func TestDevelopmentAllowsDocumentedPlaceholders(t *testing.T) {
	cfg := Config{Environment: "development", JWTExpire: time.Hour, AdminJWTExpire: time.Hour, AgentTimeout: 20 * time.Second}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("development config rejected: %v", err)
	}
}

func TestLoadConfigAcceptsOpenAIEnvironmentAliases(t *testing.T) {
	t.Setenv("AI_API_KEY", "")
	t.Setenv("AI_BASE_URL", "")
	t.Setenv("AI_MODEL", "")
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", "https://example.invalid/v1")
	t.Setenv("OPENAI_MODEL", "test-model")

	cfg := LoadConfig()
	if cfg.AgentAPIKey != "test-key" || cfg.AgentBaseURL != "https://example.invalid/v1" || cfg.AgentModel != "test-model" {
		t.Fatalf("OpenAI aliases were not loaded: key=%t base=%q model=%q", cfg.AgentAPIKey != "", cfg.AgentBaseURL, cfg.AgentModel)
	}
}

func TestLoadConfigPrefersAIEnvironmentNames(t *testing.T) {
	t.Setenv("AI_API_KEY", "preferred-key")
	t.Setenv("OPENAI_API_KEY", "fallback-key")

	if got := LoadConfig().AgentAPIKey; got != "preferred-key" {
		t.Fatalf("AgentAPIKey = %q, want AI_API_KEY value", got)
	}
}

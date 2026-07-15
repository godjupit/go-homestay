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
	cfg := Config{Environment: "development", JWTExpire: time.Hour, AdminJWTExpire: time.Hour}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("development config rejected: %v", err)
	}
}

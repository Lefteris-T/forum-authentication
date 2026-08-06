package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("FORUM_ADDRESS", "")
	t.Setenv("FORUM_DATABASE_PATH", "")
	t.Setenv("FORUM_SESSION_DURATION", "")
	t.Setenv("FORUM_COOKIE_NAME", "")
	t.Setenv("FORUM_SECURE_COOKIE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if cfg.Address != ":8080" {
		t.Errorf("Address = %q, want %q", cfg.Address, ":8080")
	}

	if cfg.DatabasePath != "data/forum.db" {
		t.Errorf(
			"DatabasePath = %q, want %q",
			cfg.DatabasePath,
			"data/forum.db",
		)
	}

	if cfg.SessionDuration != 24*time.Hour {
		t.Errorf(
			"SessionDuration = %v, want %v",
			cfg.SessionDuration,
			24*time.Hour,
		)
	}

	if cfg.CookieName != "forum_session" {
		t.Errorf(
			"CookieName = %q, want %q",
			cfg.CookieName,
			"forum_session",
		)
	}

	if cfg.SecureCookie {
		t.Error("SecureCookie = true, want false")
	}
}
func TestLoadEnvironmentOverrides(t *testing.T) {
	t.Setenv("FORUM_ADDRESS", "127.0.0.1:9090")
	t.Setenv("FORUM_DATABASE_PATH", "testdata/forum.db")
	t.Setenv("FORUM_SESSION_DURATION", "2h")
	t.Setenv("FORUM_COOKIE_NAME", "custom_session")
	t.Setenv("FORUM_SECURE_COOKIE", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if cfg.Address != "127.0.0.1:9090" {
		t.Errorf(
			"Address = %q, want %q",
			cfg.Address,
			"127.0.0.1:9090",
		)
	}

	if cfg.DatabasePath != "testdata/forum.db" {
		t.Errorf(
			"DatabasePath = %q, want %q",
			cfg.DatabasePath,
			"testdata/forum.db",
		)
	}

	if cfg.SessionDuration != 2*time.Hour {
		t.Errorf(
			"SessionDuration = %v, want %v",
			cfg.SessionDuration,
			2*time.Hour,
		)
	}

	if cfg.CookieName != "custom_session" {
		t.Errorf(
			"CookieName = %q, want %q",
			cfg.CookieName,
			"custom_session",
		)
	}

	if !cfg.SecureCookie {
		t.Error("SecureCookie = false, want true")
	}
}
func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
	}{
		{
			name:     "invalid address",
			envKey:   "FORUM_ADDRESS",
			envValue: "invalid-address",
		},
		{
			name:     "empty database path",
			envKey:   "FORUM_DATABASE_PATH",
			envValue: "   ",
		},
		{
			name:     "invalid session duration",
			envKey:   "FORUM_SESSION_DURATION",
			envValue: "tomorrow",
		},
		{
			name:     "zero session duration",
			envKey:   "FORUM_SESSION_DURATION",
			envValue: "0s",
		},
		{
			name:     "invalid cookie name",
			envKey:   "FORUM_COOKIE_NAME",
			envValue: "forum session",
		},
		{
			name:     "invalid secure cookie",
			envKey:   "FORUM_SECURE_COOKIE",
			envValue: "sometimes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("FORUM_ADDRESS", ":8080")
			t.Setenv("FORUM_DATABASE_PATH", "data/forum.db")
			t.Setenv("FORUM_SESSION_DURATION", "24h")
			t.Setenv("FORUM_COOKIE_NAME", "forum_session")
			t.Setenv("FORUM_SECURE_COOKIE", "false")

			t.Setenv(tt.envKey, tt.envValue)

			_, err := Load()
			if err == nil {
				t.Fatalf("Load() error = nil, want an error")
			}
		})
	}
}

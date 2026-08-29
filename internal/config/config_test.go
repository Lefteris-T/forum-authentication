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
func TestLoadOAuthProvidersDisabledByDefault(t *testing.T) {
	t.Setenv("GITHUB_CLIENT_ID", "")
	t.Setenv("GITHUB_CLIENT_SECRET", "")
	t.Setenv("GITHUB_REDIRECT_URL", "")
	t.Setenv("GOOGLE_CLIENT_ID", "")
	t.Setenv("GOOGLE_CLIENT_SECRET", "")
	t.Setenv("GOOGLE_REDIRECT_URL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if cfg.GitHub.Enabled {
		t.Error("GitHub.Enabled = true, want false")
	}

	if cfg.Google.Enabled {
		t.Error("Google.Enabled = true, want false")
	}
}

func TestLoadGitHubOAuthConfiguration(t *testing.T) {
	t.Setenv("GITHUB_CLIENT_ID", "github-client")
	t.Setenv("GITHUB_CLIENT_SECRET", "github-secret")
	t.Setenv(
		"GITHUB_REDIRECT_URL",
		"http://localhost:8080/auth/github/callback",
	)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if !cfg.GitHub.Enabled {
		t.Fatal("GitHub.Enabled = false, want true")
	}

	if cfg.GitHub.ClientID != "github-client" {
		t.Errorf(
			"GitHub.ClientID = %q, want %q",
			cfg.GitHub.ClientID,
			"github-client",
		)
	}

	if cfg.GitHub.ClientSecret != "github-secret" {
		t.Errorf(
			"GitHub.ClientSecret = %q, want %q",
			cfg.GitHub.ClientSecret,
			"github-secret",
		)
	}

	wantRedirect := "http://localhost:8080/auth/github/callback"
	if cfg.GitHub.RedirectURL != wantRedirect {
		t.Errorf(
			"GitHub.RedirectURL = %q, want %q",
			cfg.GitHub.RedirectURL,
			wantRedirect,
		)
	}
}

func TestLoadGoogleOAuthConfiguration(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "google-client")
	t.Setenv("GOOGLE_CLIENT_SECRET", "google-secret")
	t.Setenv(
		"GOOGLE_REDIRECT_URL",
		"http://localhost:8080/auth/google/callback",
	)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if !cfg.Google.Enabled {
		t.Fatal("Google.Enabled = false, want true")
	}

	if cfg.Google.ClientID != "google-client" {
		t.Errorf(
			"Google.ClientID = %q, want %q",
			cfg.Google.ClientID,
			"google-client",
		)
	}

	if cfg.Google.ClientSecret != "google-secret" {
		t.Errorf(
			"Google.ClientSecret = %q, want %q",
			cfg.Google.ClientSecret,
			"google-secret",
		)
	}

	wantRedirect := "http://localhost:8080/auth/google/callback"
	if cfg.Google.RedirectURL != wantRedirect {
		t.Errorf(
			"Google.RedirectURL = %q, want %q",
			cfg.Google.RedirectURL,
			wantRedirect,
		)
	}
}
func TestLoadRejectsPartialOAuthConfiguration(t *testing.T) {
	tests := []struct {
		name string
		set  func(t *testing.T)
	}{
		{
			name: "github missing client secret",
			set: func(t *testing.T) {
				t.Setenv("GITHUB_CLIENT_ID", "github-client")
				t.Setenv(
					"GITHUB_REDIRECT_URL",
					"http://localhost:8080/auth/github/callback",
				)
			},
		},
		{
			name: "google missing redirect url",
			set: func(t *testing.T) {
				t.Setenv("GOOGLE_CLIENT_ID", "google-client")
				t.Setenv("GOOGLE_CLIENT_SECRET", "google-secret")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_CLIENT_ID", "")
			t.Setenv("GITHUB_CLIENT_SECRET", "")
			t.Setenv("GITHUB_REDIRECT_URL", "")
			t.Setenv("GOOGLE_CLIENT_ID", "")
			t.Setenv("GOOGLE_CLIENT_SECRET", "")
			t.Setenv("GOOGLE_REDIRECT_URL", "")

			tt.set(t)

			_, err := Load()
			if err == nil {
				t.Fatal("Load() error = nil, want an error")
			}
		})
	}
}

func TestLoadRejectsInvalidOAuthRedirectURL(t *testing.T) {
	t.Setenv("GITHUB_CLIENT_ID", "github-client")
	t.Setenv("GITHUB_CLIENT_SECRET", "github-secret")
	t.Setenv("GITHUB_REDIRECT_URL", "not-a-url")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want an error")
	}
}

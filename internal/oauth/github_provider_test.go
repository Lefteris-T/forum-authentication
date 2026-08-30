package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGitHubProviderExchangeCodeAndFetchUser(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("token method = %s, want POST", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error: %v", err)
		}

		if r.Form.Get("client_id") != "client-id" {
			t.Errorf(
				"client_id = %q, want %q",
				r.Form.Get("client_id"),
				"client-id",
			)
		}

		if r.Form.Get("client_secret") != "client-secret" {
			t.Errorf(
				"client_secret = %q, want %q",
				r.Form.Get("client_secret"),
				"client-secret",
			)
		}

		if r.Form.Get("code") != "authorization-code" {
			t.Errorf(
				"code = %q, want %q",
				r.Form.Get("code"),
				"authorization-code",
			)
		}

		if r.Form.Get("code_verifier") != "pkce-verifier" {
			t.Errorf(
				"code_verifier = %q, want %q",
				r.Form.Get("code_verifier"),
				"pkce-verifier",
			)
		}

		w.Header().Set("Content-Type", "application/json")

		_ = json.NewEncoder(w).Encode(map[string]string{
			"access_token": "github-access-token",
			"token_type":   "bearer",
		})
	})

	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer github-access-token" {
			t.Errorf(
				"Authorization = %q, want %q",
				got,
				"Bearer github-access-token",
			)
		}

		w.Header().Set("Content-Type", "application/json")

		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    123456,
			"login": "octocat",
		})
	})

	mux.HandleFunc("/emails", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer github-access-token" {
			t.Errorf(
				"Authorization = %q, want %q",
				got,
				"Bearer github-access-token",
			)
		}

		w.Header().Set("Content-Type", "application/json")

		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"email":    "other@example.com",
				"primary":  false,
				"verified": true,
			},
			{
				"email":    "oauth@example.com",
				"primary":  true,
				"verified": true,
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	provider := NewGitHubProvider(ProviderConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "http://localhost:8080/auth/github/callback",

		TokenEndpoint: server.URL + "/token",
		UserEndpoint:  server.URL + "/user",

		Client: server.Client(),
	})

	provider.emailEndpoint = server.URL + "/emails"

	token, err := provider.ExchangeCode(
		context.Background(),
		"authorization-code",
		"pkce-verifier",
	)
	if err != nil {
		t.Fatalf("ExchangeCode() error: %v", err)
	}

	if token != "github-access-token" {
		t.Fatalf(
			"token = %q, want %q",
			token,
			"github-access-token",
		)
	}

	user, err := provider.FetchUser(
		context.Background(),
		token,
	)
	if err != nil {
		t.Fatalf("FetchUser() error: %v", err)
	}

	if user.Provider != "github" {
		t.Errorf(
			"user.Provider = %q, want %q",
			user.Provider,
			"github",
		)
	}

	if user.ProviderUserID != "123456" {
		t.Errorf(
			"user.ProviderUserID = %q, want %q",
			user.ProviderUserID,
			"123456",
		)
	}

	if user.VerifiedEmail != "oauth@example.com" {
		t.Errorf(
			"user.VerifiedEmail = %q, want %q",
			user.VerifiedEmail,
			"oauth@example.com",
		)
	}

	if user.SuggestedUsername != "octocat" {
		t.Errorf(
			"user.SuggestedUsername = %q, want %q",
			user.SuggestedUsername,
			"octocat",
		)
	}
}
func TestGitHubProviderRejectsNonSuccessTokenResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream failure", http.StatusBadGateway)
	}))
	defer server.Close()

	provider := NewGitHubProvider(ProviderConfig{
		ClientID:      "client-id",
		ClientSecret:  "client-secret",
		RedirectURL:   "http://localhost:8080/auth/github/callback",
		TokenEndpoint: server.URL,
		UserEndpoint:  server.URL + "/user",
		Client:        server.Client(),
	})

	_, err := provider.ExchangeCode(
		context.Background(),
		"authorization-code",
		"pkce-verifier",
	)

	if err == nil {
		t.Fatal("ExchangeCode() error = nil, want error")
	}
}
func TestGitHubProviderRejectsMalformedTokenJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":`))
	}))
	defer server.Close()

	provider := NewGitHubProvider(ProviderConfig{
		ClientID:      "client-id",
		ClientSecret:  "client-secret",
		RedirectURL:   "http://localhost:8080/auth/github/callback",
		TokenEndpoint: server.URL,
		UserEndpoint:  server.URL + "/user",
		Client:        server.Client(),
	})

	_, err := provider.ExchangeCode(
		context.Background(),
		"authorization-code",
		"pkce-verifier",
	)

	if err == nil {
		t.Fatal("ExchangeCode() error = nil, want error")
	}
}
func TestGitHubProviderRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("a", maxProviderResponseSize+1)))
	}))
	defer server.Close()

	provider := NewGitHubProvider(ProviderConfig{
		ClientID:      "client-id",
		ClientSecret:  "client-secret",
		RedirectURL:   "http://localhost:8080/auth/github/callback",
		TokenEndpoint: server.URL,
		UserEndpoint:  server.URL + "/user",
		Client:        server.Client(),
	})

	_, err := provider.ExchangeCode(
		context.Background(),
		"authorization-code",
		"pkce-verifier",
	)

	if err == nil {
		t.Fatal("ExchangeCode() error = nil, want error")
	}
}
func TestGitHubProviderRejectsMissingUserID(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"login": "octocat",
		})
	})

	mux.HandleFunc("/emails", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"email":    "oauth@example.com",
				"primary":  true,
				"verified": true,
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	provider := NewGitHubProvider(ProviderConfig{
		UserEndpoint: server.URL + "/user",
		Client:       server.Client(),
	})

	provider.emailEndpoint = server.URL + "/emails"

	_, err := provider.FetchUser(
		context.Background(),
		"access-token",
	)

	if err == nil {
		t.Fatal("FetchUser() error = nil, want error")
	}
}
func TestGitHubProviderRejectsMissingVerifiedPrimaryEmail(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    123456,
			"login": "octocat",
		})
	})

	mux.HandleFunc("/emails", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"email":    "oauth@example.com",
				"primary":  true,
				"verified": false,
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	provider := NewGitHubProvider(ProviderConfig{
		UserEndpoint: server.URL + "/user",
		Client:       server.Client(),
	})

	provider.emailEndpoint = server.URL + "/emails"

	_, err := provider.FetchUser(
		context.Background(),
		"access-token",
	)

	if err == nil {
		t.Fatal("FetchUser() error = nil, want error")
	}
}
func TestGitHubProviderReturnsErrorOnTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)

		_ = json.NewEncoder(w).Encode(map[string]string{
			"access_token": "token",
		})
	}))
	defer server.Close()

	client := server.Client()
	client.Timeout = 10 * time.Millisecond

	provider := NewGitHubProvider(ProviderConfig{
		ClientID:      "client-id",
		ClientSecret:  "client-secret",
		RedirectURL:   "http://localhost:8080/auth/github/callback",
		TokenEndpoint: server.URL,
		UserEndpoint:  server.URL + "/user",
		Client:        client,
	})

	_, err := provider.ExchangeCode(
		context.Background(),
		"authorization-code",
		"pkce-verifier",
	)

	if err == nil {
		t.Fatal("ExchangeCode() error = nil, want timeout error")
	}
}

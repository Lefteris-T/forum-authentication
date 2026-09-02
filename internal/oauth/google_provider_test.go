package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestGoogleProviderAuthorizationURL(t *testing.T) {
	provider := NewGoogleProvider(ProviderConfig{
		ClientID:              "google-client-id",
		ClientSecret:          "google-client-secret",
		RedirectURL:           "http://localhost:8080/auth/google/callback",
		AuthorizationEndpoint: "https://accounts.example/authorize",
		TokenEndpoint:         "https://oauth.example/token",
		UserEndpoint:          "https://openidconnect.example/userinfo",
		Client:                DefaultHTTPClient(),
	})

	location, err := provider.AuthorizationURL(
		"state-value",
		"pkce-challenge",
	)
	if err != nil {
		t.Fatalf("AuthorizationURL() error: %v", err)
	}

	redirectURL, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse authorization URL: %v", err)
	}

	if redirectURL.Scheme != "https" {
		t.Errorf("scheme = %q, want %q", redirectURL.Scheme, "https")
	}

	if redirectURL.Host != "accounts.example" {
		t.Errorf(
			"host = %q, want %q",
			redirectURL.Host,
			"accounts.example",
		)
	}

	query := redirectURL.Query()

	wantParameters := map[string]string{
		"client_id":             "google-client-id",
		"redirect_uri":          "http://localhost:8080/auth/google/callback",
		"response_type":         "code",
		"state":                 "state-value",
		"code_challenge":        "pkce-challenge",
		"code_challenge_method": "S256",
	}

	for name, want := range wantParameters {
		if got := query.Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}

	gotScopes := strings.Fields(query.Get("scope"))
	wantScopes := []string{"openid", "email", "profile"}

	if !reflect.DeepEqual(gotScopes, wantScopes) {
		t.Errorf("scopes = %q, want %q", gotScopes, wantScopes)
	}
}

func TestGoogleProviderExchangesAuthorizationCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want %q", r.Method, http.MethodPost)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error: %v", err)
		}

		wantForm := map[string]string{
			"client_id":     "google-client-id",
			"client_secret": "google-client-secret",
			"code":          "authorization-code",
			"redirect_uri":  "http://localhost:8080/auth/google/callback",
			"grant_type":    "authorization_code",
			"code_verifier": "pkce-verifier",
		}

		for name, want := range wantForm {
			if got := r.Form.Get(name); got != want {
				t.Errorf("%s = %q, want %q", name, got, want)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"access_token": "google-access-token",
			"token_type":   "Bearer",
		})
	}))
	defer server.Close()

	provider := NewGoogleProvider(ProviderConfig{
		ClientID:      "google-client-id",
		ClientSecret:  "google-client-secret",
		RedirectURL:   "http://localhost:8080/auth/google/callback",
		TokenEndpoint: server.URL,
		Client:        server.Client(),
	})

	token, err := provider.ExchangeCode(
		context.Background(),
		"authorization-code",
		"pkce-verifier",
	)
	if err != nil {
		t.Fatalf("ExchangeCode() error: %v", err)
	}

	if token != "google-access-token" {
		t.Fatalf("token = %q, want %q", token, "google-access-token")
	}
}

func TestGoogleProviderRejectsInvalidTokenResponse(t *testing.T) {
	tests := []struct {
		name    string
		handler http.Handler
	}{
		{
			name: "non-success status",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "upstream failure", http.StatusBadGateway)
			}),
		},
		{
			name: "malformed json",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"access_token":`))
			}),
		},
		{
			name: "missing access token",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{}`))
			}),
		},
		{
			name: "oversized response",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(strings.Repeat("a", maxProviderResponseSize+1)))
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			provider := NewGoogleProvider(ProviderConfig{
				TokenEndpoint: server.URL,
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
		})
	}
}

func TestGoogleProviderTokenExchangeTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		time.Sleep(100 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"access_token": "google-access-token",
		})
	}))
	defer server.Close()

	client := server.Client()
	client.Timeout = 10 * time.Millisecond

	provider := NewGoogleProvider(ProviderConfig{
		TokenEndpoint: server.URL,
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

func TestGoogleProviderFetchesVerifiedUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want %q", r.Method, http.MethodGet)
		}

		if got := r.Header.Get("Authorization"); got != "Bearer google-access-token" {
			t.Errorf(
				"Authorization = %q, want %q",
				got,
				"Bearer google-access-token",
			)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"sub":            "google-subject-123",
			"email":          " oauth@example.com ",
			"email_verified": true,
			"name":           "OAuth User",
		})
	}))
	defer server.Close()

	provider := NewGoogleProvider(ProviderConfig{
		UserEndpoint: server.URL,
		Client:       server.Client(),
	})

	user, err := provider.FetchUser(
		context.Background(),
		"google-access-token",
	)
	if err != nil {
		t.Fatalf("FetchUser() error: %v", err)
	}

	want := User{
		Provider:          "google",
		ProviderUserID:    "google-subject-123",
		VerifiedEmail:     "oauth@example.com",
		SuggestedUsername: "OAuth User",
	}

	if !reflect.DeepEqual(user, want) {
		t.Fatalf("user = %#v, want %#v", user, want)
	}
}

func TestGoogleProviderRejectsInvalidIdentity(t *testing.T) {
	tests := []struct {
		name    string
		profile map[string]any
	}{
		{
			name: "missing subject",
			profile: map[string]any{
				"email":          "oauth@example.com",
				"email_verified": true,
			},
		},
		{
			name: "missing email",
			profile: map[string]any{
				"sub":            "google-subject-123",
				"email_verified": true,
			},
		},
		{
			name: "blank email",
			profile: map[string]any{
				"sub":            "google-subject-123",
				"email":          "   ",
				"email_verified": true,
			},
		},
		{
			name: "unverified email",
			profile: map[string]any{
				"sub":            "google-subject-123",
				"email":          "oauth@example.com",
				"email_verified": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				_ = json.NewEncoder(w).Encode(tt.profile)
			}))
			defer server.Close()

			provider := NewGoogleProvider(ProviderConfig{
				UserEndpoint: server.URL,
				Client:       server.Client(),
			})

			_, err := provider.FetchUser(context.Background(), "access-token")
			if err == nil {
				t.Fatal("FetchUser() error = nil, want error")
			}
		})
	}
}

func TestGoogleProviderRejectsInvalidProfileResponse(t *testing.T) {
	tests := []struct {
		name    string
		handler http.Handler
	}{
		{
			name: "non-success status",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "upstream failure", http.StatusBadGateway)
			}),
		},
		{
			name: "malformed json",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"sub":`))
			}),
		},
		{
			name: "oversized response",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(strings.Repeat("a", maxProviderResponseSize+1)))
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			provider := NewGoogleProvider(ProviderConfig{
				UserEndpoint: server.URL,
				Client:       server.Client(),
			})

			_, err := provider.FetchUser(context.Background(), "access-token")
			if err == nil {
				t.Fatal("FetchUser() error = nil, want error")
			}
		})
	}
}

func TestGoogleProviderProfileRequestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		time.Sleep(100 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sub":            "google-subject-123",
			"email":          "oauth@example.com",
			"email_verified": true,
		})
	}))
	defer server.Close()

	client := server.Client()
	client.Timeout = 10 * time.Millisecond

	provider := NewGoogleProvider(ProviderConfig{
		UserEndpoint: server.URL,
		Client:       client,
	})

	_, err := provider.FetchUser(context.Background(), "access-token")
	if err == nil {
		t.Fatal("FetchUser() error = nil, want timeout error")
	}
}

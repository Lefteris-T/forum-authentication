package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type recordingAuthorizationProvider struct {
	state     string
	challenge string
	called    bool
}

func (p *recordingAuthorizationProvider) AuthorizationURL(
	state string,
	challenge string,
) (string, error) {
	p.called = true
	p.state = state
	p.challenge = challenge

	return "https://provider.example/authorize", nil
}

func (p *recordingAuthorizationProvider) ExchangeCode(
	ctx context.Context,
	code string,
	verifier string,
) (string, error) {
	return "", nil
}

func (p *recordingAuthorizationProvider) FetchUser(
	ctx context.Context,
	accessToken string,
) (User, error) {
	return User{}, nil
}

func TestAuthorizationHandlerStartsProviderFlow(t *testing.T) {
	store := NewOAuthStateStore()
	provider := &recordingAuthorizationProvider{}

	handler := NewAuthorizationHandler(
		provider,
		"google",
		store,
		"google_oauth_state",
		false,
	)

	req := httptest.NewRequest(http.MethodGet, "/auth/google", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusFound)
	}

	if res.Header.Get("Location") != "https://provider.example/authorize" {
		t.Fatalf(
			"Location = %q, want %q",
			res.Header.Get("Location"),
			"https://provider.example/authorize",
		)
	}

	if !provider.called {
		t.Fatal("provider AuthorizationURL() was not called")
	}

	if provider.state == "" {
		t.Fatal("provider received an empty state")
	}

	if provider.challenge == "" {
		t.Fatal("provider received an empty PKCE challenge")
	}

	var stateCookie *http.Cookie

	for _, cookie := range res.Cookies() {
		if cookie.Name == "google_oauth_state" {
			stateCookie = cookie
			break
		}
	}

	if stateCookie == nil {
		t.Fatal("google oauth state cookie was not set")
	}

	if stateCookie.Value != provider.state {
		t.Fatalf(
			"cookie state = %q, provider state = %q",
			stateCookie.Value,
			provider.state,
		)
	}

	flow, err := store.Consume(provider.state, "google")
	if err != nil {
		t.Fatalf("Consume() error: %v", err)
	}

	if flow.Verifier == "" {
		t.Fatal("stored PKCE verifier is empty")
	}

	if PKCEChallenge(flow.Verifier) != provider.challenge {
		t.Fatal("stored verifier does not match provider PKCE challenge")
	}
}

func TestAuthorizationHandlerRejectsWrongMethod(t *testing.T) {
	provider := &recordingAuthorizationProvider{}

	handler := NewAuthorizationHandler(
		provider,
		"google",
		NewOAuthStateStore(),
		"google_oauth_state",
		false,
	)

	req := httptest.NewRequest(http.MethodPost, "/auth/google", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusMethodNotAllowed,
		)
	}

	if rec.Header().Get("Allow") != http.MethodGet {
		t.Fatalf(
			"Allow = %q, want %q",
			rec.Header().Get("Allow"),
			http.MethodGet,
		)
	}

	if provider.called {
		t.Fatal("provider was called for an unsupported method")
	}
}

package oauth

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestGitHubAuthorizationHandlerRedirectsToProvider(t *testing.T) {
	store := NewOAuthStateStore()

	cfg := ProviderConfig{
		ClientID:              "github-client-id",
		ClientSecret:          "github-client-secret",
		RedirectURL:           "http://localhost:8080/auth/github/callback",
		AuthorizationEndpoint: "https://github.example/authorize",
		TokenEndpoint:         "https://github.example/token",
		UserEndpoint:          "https://github.example/user",
		Client:                DefaultHTTPClient(),
	}

	handler := NewGitHubAuthorizationHandler(
		cfg,
		store,
		"github_oauth_state",
		false,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/auth/github",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusFound {
		t.Fatalf(
			"status = %d, want %d",
			res.StatusCode,
			http.StatusFound,
		)
	}

	location := res.Header.Get("Location")
	if location == "" {
		t.Fatal("Location header is empty")
	}

	redirectURL, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse redirect URL: %v", err)
	}

	if redirectURL.Scheme != "https" {
		t.Errorf(
			"redirect scheme = %q, want %q",
			redirectURL.Scheme,
			"https",
		)
	}

	if redirectURL.Host != "github.example" {
		t.Errorf(
			"redirect host = %q, want %q",
			redirectURL.Host,
			"github.example",
		)
	}

	query := redirectURL.Query()

	if query.Get("client_id") != "github-client-id" {
		t.Errorf(
			"client_id = %q, want %q",
			query.Get("client_id"),
			"github-client-id",
		)
	}

	if query.Get("redirect_uri") != cfg.RedirectURL {
		t.Errorf(
			"redirect_uri = %q, want %q",
			query.Get("redirect_uri"),
			cfg.RedirectURL,
		)
	}

	if query.Get("state") == "" {
		t.Fatal("state query parameter is empty")
	}

	if query.Get("code_challenge") == "" {
		t.Fatal("code_challenge query parameter is empty")
	}

	if query.Get("code_challenge_method") != "S256" {
		t.Errorf(
			"code_challenge_method = %q, want %q",
			query.Get("code_challenge_method"),
			"S256",
		)
	}

	if !strings.Contains(query.Get("scope"), "user:email") {
		t.Errorf(
			"scope = %q, want user:email",
			query.Get("scope"),
		)
	}

	cookies := res.Cookies()

	var stateCookie *http.Cookie

	for _, cookie := range cookies {
		if cookie.Name == "github_oauth_state" {
			stateCookie = cookie
			break
		}
	}

	if stateCookie == nil {
		t.Fatal("github oauth state cookie not set")
	}

	if stateCookie.Value == "" {
		t.Fatal("github oauth state cookie is empty")
	}

	if stateCookie.Value != query.Get("state") {
		t.Fatalf(
			"cookie state = %q, query state = %q",
			stateCookie.Value,
			query.Get("state"),
		)
	}
}
func TestGitHubAuthorizationHandlerRejectsWrongMethod(t *testing.T) {
	store := NewOAuthStateStore()

	cfg := ProviderConfig{
		ClientID:              "github-client-id",
		ClientSecret:          "github-client-secret",
		RedirectURL:           "http://localhost:8080/auth/github/callback",
		AuthorizationEndpoint: "https://github.example/authorize",
		TokenEndpoint:         "https://github.example/token",
		UserEndpoint:          "https://github.example/user",
		Client:                DefaultHTTPClient(),
	}

	handler := NewGitHubAuthorizationHandler(
		cfg,
		store,
		"github_oauth_state",
		false,
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/auth/github",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf(
			"status = %d, want %d",
			res.StatusCode,
			http.StatusMethodNotAllowed,
		)
	}

	if res.Header.Get("Allow") != http.MethodGet {
		t.Fatalf(
			"Allow = %q, want %q",
			res.Header.Get("Allow"),
			http.MethodGet,
		)
	}
}

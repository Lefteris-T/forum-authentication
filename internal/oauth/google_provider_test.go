package oauth

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
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

package oauth

import (
	"context"
	"testing"
)

type fakeProvider struct{}

func (fakeProvider) AuthorizationURL(state, challenge string) (string, error) {
	return "https://provider.example/authorize", nil
}

func (fakeProvider) ExchangeCode(
	ctx context.Context,
	code string,
	verifier string,
) (string, error) {
	return "access-token", nil
}

func (fakeProvider) FetchUser(
	ctx context.Context,
	accessToken string,
) (User, error) {
	return User{
		Provider:          "github",
		ProviderUserID:    "123456",
		VerifiedEmail:     "oauth@example.com",
		SuggestedUsername: "oauth-user",
	}, nil
}

func TestProviderContract(t *testing.T) {
	var provider Provider = fakeProvider{}

	url, err := provider.AuthorizationURL("state-value", "pkce-challenge")
	if err != nil {
		t.Fatalf("AuthorizationURL() error: %v", err)
	}

	if url == "" {
		t.Fatal("AuthorizationURL() returned empty URL")
	}

	token, err := provider.ExchangeCode(
		context.Background(),
		"authorization-code",
		"pkce-verifier",
	)
	if err != nil {
		t.Fatalf("ExchangeCode() error: %v", err)
	}

	if token != "access-token" {
		t.Fatalf("token = %q, want %q", token, "access-token")
	}

	user, err := provider.FetchUser(
		context.Background(),
		token,
	)
	if err != nil {
		t.Fatalf("FetchUser() error: %v", err)
	}

	if user.Provider != "github" {
		t.Errorf("Provider = %q, want %q", user.Provider, "github")
	}

	if user.ProviderUserID != "123456" {
		t.Errorf(
			"ProviderUserID = %q, want %q",
			user.ProviderUserID,
			"123456",
		)
	}

	if user.VerifiedEmail != "oauth@example.com" {
		t.Errorf(
			"VerifiedEmail = %q, want %q",
			user.VerifiedEmail,
			"oauth@example.com",
		)
	}

	if user.SuggestedUsername != "oauth-user" {
		t.Errorf(
			"SuggestedUsername = %q, want %q",
			user.SuggestedUsername,
			"oauth-user",
		)
	}
}

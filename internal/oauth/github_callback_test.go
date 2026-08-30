package oauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeCallbackProvider struct {
	exchangeCode     string
	exchangeVerifier string
	fetchToken       string
}

func (p *fakeCallbackProvider) AuthorizationURL(
	state string,
	challenge string,
) (string, error) {
	return "", nil
}

func (p *fakeCallbackProvider) ExchangeCode(
	ctx context.Context,
	code string,
	verifier string,
) (string, error) {
	p.exchangeCode = code
	p.exchangeVerifier = verifier

	return "access-token", nil
}

func (p *fakeCallbackProvider) FetchUser(
	ctx context.Context,
	accessToken string,
) (User, error) {
	p.fetchToken = accessToken

	return User{
		Provider:          "github",
		ProviderUserID:    "123456",
		VerifiedEmail:     "oauth@example.com",
		SuggestedUsername: "octocat",
	}, nil
}

func TestGitHubCallbackHandlerCompletesOAuthFlow(t *testing.T) {
	store := NewOAuthStateStore()

	state := "state-value"
	verifier := "pkce-verifier"

	store.Save(
		state,
		"github",
		verifier,
		time.Now().Add(10*time.Minute),
	)

	provider := &fakeCallbackProvider{}

	var authenticatedUser User
	var successCalled bool

	handler := NewGitHubCallbackHandler(
		provider,
		store,
		"github_oauth_state",
		false,
		func(
			w http.ResponseWriter,
			r *http.Request,
			user User,
		) {
			successCalled = true
			authenticatedUser = user

			w.WriteHeader(http.StatusNoContent)
		},
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/auth/github/callback?code=authorization-code&state="+state,
		nil,
	)

	req.AddCookie(&http.Cookie{
		Name:  "github_oauth_state",
		Value: state,
	})

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf(
			"status = %d, want %d",
			res.StatusCode,
			http.StatusNoContent,
		)
	}

	if !successCalled {
		t.Fatal("success handler was not called")
	}

	if provider.exchangeCode != "authorization-code" {
		t.Errorf(
			"exchange code = %q, want %q",
			provider.exchangeCode,
			"authorization-code",
		)
	}

	if provider.exchangeVerifier != verifier {
		t.Errorf(
			"exchange verifier = %q, want %q",
			provider.exchangeVerifier,
			verifier,
		)
	}

	if provider.fetchToken != "access-token" {
		t.Errorf(
			"fetch token = %q, want %q",
			provider.fetchToken,
			"access-token",
		)
	}

	if authenticatedUser.ProviderUserID != "123456" {
		t.Errorf(
			"ProviderUserID = %q, want %q",
			authenticatedUser.ProviderUserID,
			"123456",
		)
	}

	if authenticatedUser.VerifiedEmail != "oauth@example.com" {
		t.Errorf(
			"VerifiedEmail = %q, want %q",
			authenticatedUser.VerifiedEmail,
			"oauth@example.com",
		)
	}

	// The state must now be consumed.
	_, err := store.Consume(state, "github")
	if err == nil {
		t.Fatal("state can still be consumed after successful callback")
	}
}

func TestGitHubCallbackRejectsMissingState(t *testing.T) {
	store := NewOAuthStateStore()
	provider := &fakeCallbackProvider{}

	successCalled := false

	handler := NewGitHubCallbackHandler(
		provider,
		store,
		"github_oauth_state",
		false,
		func(
			w http.ResponseWriter,
			r *http.Request,
			user User,
		) {
			successCalled = true
		},
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/auth/github/callback?code=authorization-code",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusBadRequest,
		)
	}

	if successCalled {
		t.Fatal("success handler was called")
	}
}

func TestGitHubCallbackRejectsMismatchedStateCookie(t *testing.T) {
	store := NewOAuthStateStore()

	store.Save(
		"expected-state",
		"github",
		"pkce-verifier",
		time.Now().Add(10*time.Minute),
	)

	provider := &fakeCallbackProvider{}

	successCalled := false

	handler := NewGitHubCallbackHandler(
		provider,
		store,
		"github_oauth_state",
		false,
		func(
			w http.ResponseWriter,
			r *http.Request,
			user User,
		) {
			successCalled = true
		},
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/auth/github/callback?code=authorization-code&state=expected-state",
		nil,
	)

	req.AddCookie(&http.Cookie{
		Name:  "github_oauth_state",
		Value: "different-state",
	})

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusBadRequest,
		)
	}

	if successCalled {
		t.Fatal("success handler was called")
	}
}

func TestGitHubCallbackRejectsMissingCode(t *testing.T) {
	store := NewOAuthStateStore()

	state := "state-value"

	store.Save(
		state,
		"github",
		"pkce-verifier",
		time.Now().Add(10*time.Minute),
	)

	provider := &fakeCallbackProvider{}

	successCalled := false

	handler := NewGitHubCallbackHandler(
		provider,
		store,
		"github_oauth_state",
		false,
		func(
			w http.ResponseWriter,
			r *http.Request,
			user User,
		) {
			successCalled = true
		},
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/auth/github/callback?state="+state,
		nil,
	)

	req.AddCookie(&http.Cookie{
		Name:  "github_oauth_state",
		Value: state,
	})

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusBadRequest,
		)
	}

	if successCalled {
		t.Fatal("success handler was called")
	}
}

func TestGitHubCallbackRejectsProviderDeniedAuthorization(t *testing.T) {
	store := NewOAuthStateStore()

	provider := &fakeCallbackProvider{}

	successCalled := false

	handler := NewGitHubCallbackHandler(
		provider,
		store,
		"github_oauth_state",
		false,
		func(
			w http.ResponseWriter,
			r *http.Request,
			user User,
		) {
			successCalled = true
		},
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/auth/github/callback?error=access_denied",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusBadRequest,
		)
	}

	if successCalled {
		t.Fatal("success handler was called")
	}
}

type failingCallbackProvider struct {
	exchangeErr error
	fetchErr    error
}

func (p *failingCallbackProvider) AuthorizationURL(
	state string,
	challenge string,
) (string, error) {
	return "", nil
}

func (p *failingCallbackProvider) ExchangeCode(
	ctx context.Context,
	code string,
	verifier string,
) (string, error) {
	if p.exchangeErr != nil {
		return "", p.exchangeErr
	}

	return "access-token", nil
}

func (p *failingCallbackProvider) FetchUser(
	ctx context.Context,
	accessToken string,
) (User, error) {
	if p.fetchErr != nil {
		return User{}, p.fetchErr
	}

	return User{
		Provider:          "github",
		ProviderUserID:    "123456",
		VerifiedEmail:     "oauth@example.com",
		SuggestedUsername: "octocat",
	}, nil
}
func TestGitHubCallbackReturnsBadGatewayOnExchangeFailure(t *testing.T) {
	store := NewOAuthStateStore()

	state := "state-value"

	store.Save(
		state,
		"github",
		"pkce-verifier",
		time.Now().Add(10*time.Minute),
	)

	provider := &failingCallbackProvider{
		exchangeErr: errors.New("exchange failed"),
	}

	handler := NewGitHubCallbackHandler(
		provider,
		store,
		"github_oauth_state",
		false,
		func(
			w http.ResponseWriter,
			r *http.Request,
			user User,
		) {
		},
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/auth/github/callback?code=authorization-code&state="+state,
		nil,
	)

	req.AddCookie(&http.Cookie{
		Name:  "github_oauth_state",
		Value: state,
	})

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusBadGateway,
		)
	}
}

func TestGitHubCallbackReturnsBadGatewayOnFetchUserFailure(t *testing.T) {
	store := NewOAuthStateStore()

	state := "state-value"

	store.Save(
		state,
		"github",
		"pkce-verifier",
		time.Now().Add(10*time.Minute),
	)

	provider := &failingCallbackProvider{
		fetchErr: errors.New("fetch failed"),
	}

	handler := NewGitHubCallbackHandler(
		provider,
		store,
		"github_oauth_state",
		false,
		func(
			w http.ResponseWriter,
			r *http.Request,
			user User,
		) {
		},
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/auth/github/callback?code=authorization-code&state="+state,
		nil,
	)

	req.AddCookie(&http.Cookie{
		Name:  "github_oauth_state",
		Value: state,
	})

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusBadGateway,
		)
	}
}

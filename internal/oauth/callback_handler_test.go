package oauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type recordingCallbackProvider struct {
	exchangeCode     string
	exchangeVerifier string
	fetchToken       string
	exchangeErr      error
	fetchErr         error
}

func (p *recordingCallbackProvider) AuthorizationURL(
	state string,
	challenge string,
) (string, error) {
	return "", nil
}

func (p *recordingCallbackProvider) ExchangeCode(
	ctx context.Context,
	code string,
	verifier string,
) (string, error) {
	p.exchangeCode = code
	p.exchangeVerifier = verifier

	if p.exchangeErr != nil {
		return "", p.exchangeErr
	}

	return "google-access-token", nil
}

func (p *recordingCallbackProvider) FetchUser(
	ctx context.Context,
	accessToken string,
) (User, error) {
	p.fetchToken = accessToken

	if p.fetchErr != nil {
		return User{}, p.fetchErr
	}

	return User{
		Provider:          "google",
		ProviderUserID:    "google-subject",
		VerifiedEmail:     "oauth@example.com",
		SuggestedUsername: "OAuth User",
	}, nil
}

func TestCallbackHandlerCompletesProviderFlow(t *testing.T) {
	store := NewOAuthStateStore()
	provider := &recordingCallbackProvider{}
	state := "state-value"
	verifier := "pkce-verifier"

	store.Save(
		state,
		"google",
		verifier,
		time.Now().Add(10*time.Minute),
	)

	var authenticatedUser User

	handler := NewCallbackHandler(
		provider,
		"google",
		store,
		"google_oauth_state",
		false,
		func(w http.ResponseWriter, r *http.Request, user User) {
			authenticatedUser = user
			w.WriteHeader(http.StatusNoContent)
		},
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/auth/google/callback?code=authorization-code&state="+state,
		nil,
	)
	req.AddCookie(&http.Cookie{Name: "google_oauth_state", Value: state})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}

	assertStateCookieCleared(t, rec, "google_oauth_state")

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

	if provider.fetchToken != "google-access-token" {
		t.Errorf(
			"fetch token = %q, want %q",
			provider.fetchToken,
			"google-access-token",
		)
	}

	if authenticatedUser.Provider != "google" {
		t.Errorf(
			"authenticated provider = %q, want %q",
			authenticatedUser.Provider,
			"google",
		)
	}

	if _, err := store.Consume(state, "google"); err == nil {
		t.Fatal("state can still be consumed after successful callback")
	}
}

func TestCallbackHandlerRejectsInvalidProviderFlow(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		expires       time.Time
		consumeBefore bool
		query         string
	}{
		{
			name:     "provider denial",
			provider: "google",
			expires:  time.Now().Add(10 * time.Minute),
			query:    "error=access_denied",
		},
		{
			name:     "expired state",
			provider: "google",
			expires:  time.Now().Add(-time.Minute),
			query:    "code=authorization-code&state=state-value",
		},
		{
			name:          "replayed state",
			provider:      "google",
			expires:       time.Now().Add(10 * time.Minute),
			consumeBefore: true,
			query:         "code=authorization-code&state=state-value",
		},
		{
			name:     "provider mismatch",
			provider: "github",
			expires:  time.Now().Add(10 * time.Minute),
			query:    "code=authorization-code&state=state-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewOAuthStateStore()
			provider := &recordingCallbackProvider{}

			store.Save(
				"state-value",
				tt.provider,
				"pkce-verifier",
				tt.expires,
			)

			if tt.consumeBefore {
				_, _ = store.Consume("state-value", "google")
			}

			handler := NewCallbackHandler(
				provider,
				"google",
				store,
				"google_oauth_state",
				false,
				func(w http.ResponseWriter, r *http.Request, user User) {
					t.Fatal("success handler was called")
				},
			)

			req := httptest.NewRequest(
				http.MethodGet,
				"/auth/google/callback?"+tt.query,
				nil,
			)
			req.AddCookie(&http.Cookie{
				Name:  "google_oauth_state",
				Value: "state-value",
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

			assertStateCookieCleared(t, rec, "google_oauth_state")

			if provider.exchangeCode != "" {
				t.Fatal("authorization code was exchanged for an invalid flow")
			}
		})
	}
}

func TestCallbackHandlerReturnsBadGatewayForProviderFailure(t *testing.T) {
	tests := []struct {
		name     string
		provider *recordingCallbackProvider
	}{
		{
			name: "token exchange",
			provider: &recordingCallbackProvider{
				exchangeErr: errors.New("exchange failed"),
			},
		},
		{
			name: "profile retrieval",
			provider: &recordingCallbackProvider{
				fetchErr: errors.New("fetch failed"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewOAuthStateStore()
			store.Save(
				"state-value",
				"google",
				"pkce-verifier",
				time.Now().Add(10*time.Minute),
			)

			handler := NewCallbackHandler(
				tt.provider,
				"google",
				store,
				"google_oauth_state",
				false,
				func(w http.ResponseWriter, r *http.Request, user User) {
					t.Fatal("success handler was called")
				},
			)

			req := httptest.NewRequest(
				http.MethodGet,
				"/auth/google/callback?code=authorization-code&state=state-value",
				nil,
			)
			req.AddCookie(&http.Cookie{
				Name:  "google_oauth_state",
				Value: "state-value",
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

			assertStateCookieCleared(t, rec, "google_oauth_state")
		})
	}
}

func assertStateCookieCleared(
	t *testing.T,
	rec *httptest.ResponseRecorder,
	cookieName string,
) {
	t.Helper()

	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == cookieName && cookie.MaxAge < 0 && cookie.Value == "" {
			return
		}
	}

	t.Fatalf("state cookie %q was not cleared", cookieName)
}

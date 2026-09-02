package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"forum/internal/model"
	"forum/internal/oauth"
	"forum/internal/service"
)

type fakeOAuthLoginService struct {
	user   model.User
	err    error
	called bool
}

func (f *fakeOAuthLoginService) Login(
	oauthUser oauth.User,
) (model.User, error) {
	f.called = true
	return f.user, f.err
}

type fakeOAuthSessionManager struct {
	sessionID string
	createErr error
	created   bool
	cleared   bool
}

func (f *fakeOAuthSessionManager) Create(
	w http.ResponseWriter,
) (string, error) {
	f.created = true
	return f.sessionID, f.createErr
}

func (f *fakeOAuthSessionManager) Clear(
	w http.ResponseWriter,
) {
	f.cleared = true
}

type fakeOAuthSessionService struct {
	sessionID string
	userID    int64
	err       error
	called    bool
}

func (f *fakeOAuthSessionService) CreateSession(
	sessionID string,
	userID int64,
) error {
	f.called = true
	f.sessionID = sessionID
	f.userID = userID
	return f.err
}

func TestOAuthSuccessCreatesForumSession(t *testing.T) {
	oauthLogin := &fakeOAuthLoginService{
		user: model.User{
			ID:       42,
			Email:    "oauth@example.com",
			Username: "octocat",
		},
	}

	sessionManager := &fakeOAuthSessionManager{
		sessionID: "session-123",
	}

	sessionService := &fakeOAuthSessionService{}

	handler := NewOAuthSuccessHandler(
		oauthLogin,
		sessionManager,
		sessionService,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/auth/github/callback",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.Handle(
		rec,
		req,
		oauth.User{
			Provider:          "github",
			ProviderUserID:    "123456",
			VerifiedEmail:     "oauth@example.com",
			SuggestedUsername: "octocat",
		},
	)

	if !oauthLogin.called {
		t.Fatal("OAuth Login() was not called")
	}

	if !sessionManager.created {
		t.Fatal("session cookie was not created")
	}

	if !sessionService.called {
		t.Fatal("CreateSession() was not called")
	}

	if sessionService.sessionID != "session-123" {
		t.Fatalf(
			"session ID = %q, want %q",
			sessionService.sessionID,
			"session-123",
		)
	}

	if sessionService.userID != 42 {
		t.Fatalf(
			"user ID = %d, want %d",
			sessionService.userID,
			42,
		)
	}

	if rec.Code != http.StatusSeeOther {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusSeeOther,
		)
	}

	if location := rec.Header().Get("Location"); location != "/" {
		t.Fatalf(
			"Location = %q, want %q",
			location,
			"/",
		)
	}
}

func TestOAuthSuccessClearsCookieWhenDatabaseSessionFails(
	t *testing.T,
) {
	oauthLogin := &fakeOAuthLoginService{
		user: model.User{
			ID: 42,
		},
	}

	sessionManager := &fakeOAuthSessionManager{
		sessionID: "session-123",
	}

	sessionService := &fakeOAuthSessionService{
		err: errors.New("database failed"),
	}

	handler := NewOAuthSuccessHandler(
		oauthLogin,
		sessionManager,
		sessionService,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/auth/github/callback",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.Handle(
		rec,
		req,
		oauth.User{
			Provider:       "github",
			ProviderUserID: "123456",
		},
	)

	if !sessionManager.cleared {
		t.Fatal(
			"session cookie was not cleared after database failure",
		)
	}

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusInternalServerError,
		)
	}
}

func TestOAuthSuccessRejectsEmailCollisionWithoutSession(t *testing.T) {
	oauthLogin := &fakeOAuthLoginService{
		err: service.ErrOAuthEmailConflict,
	}
	sessionManager := &fakeOAuthSessionManager{
		sessionID: "session-123",
	}
	sessionService := &fakeOAuthSessionService{}
	handler := NewOAuthSuccessHandler(
		oauthLogin,
		sessionManager,
		sessionService,
	)
	rec := httptest.NewRecorder()

	handler.Handle(
		rec,
		httptest.NewRequest(http.MethodGet, "/auth/google/callback", nil),
		oauth.User{
			Provider:       "google",
			ProviderUserID: "google-subject",
			VerifiedEmail:  "existing@example.com",
		},
	)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusConflict)
	}

	if sessionManager.created || sessionService.called {
		t.Fatal("forum session was created for an email collision")
	}

	if strings.Contains(rec.Body.String(), "existing@example.com") {
		t.Fatal("colliding email was exposed in the response")
	}
}

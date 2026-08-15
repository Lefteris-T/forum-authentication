package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"forum/internal/model"
	"forum/internal/repository"
)

type fakeSessionReader struct {
	id string
	ok bool
}

func (f *fakeSessionReader) Read(r *http.Request) (string, bool) {
	return f.id, f.ok
}

type fakeSessionFinder struct {
	session model.Session
	err     error
}

func (f *fakeSessionFinder) ByID(id string) (model.Session, error) {
	return f.session, f.err
}

type fakeUserFinder struct {
	user model.User
	err  error
}

func (f *fakeUserFinder) ByID(id int64) (model.User, error) {
	return f.user, f.err
}

func TestAuthenticateGuest(t *testing.T) {
	sessions := &fakeSessionReader{
		ok: false,
	}

	middleware := NewAuthentication(
		sessions,
		nil,
		nil,
	)

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if _, ok := CurrentUser(r.Context()); ok {
			t.Fatal("guest request has current user")
		}

		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(
		http.MethodGet,
		"/",
		nil,
	)

	rec := httptest.NewRecorder()

	middleware(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusOK,
		)
	}
}

func TestAuthenticateValidSessionAddsCurrentUser(t *testing.T) {
	cookieReader := &fakeSessionReader{
		id: "session-1",
		ok: true,
	}

	sessionFinder := &fakeSessionFinder{
		session: model.Session{
			ID:        "session-1",
			UserID:    42,
			ExpiresAt: time.Now().UTC().Add(time.Hour),
		},
	}

	userFinder := &fakeUserFinder{
		user: model.User{
			ID:       42,
			Email:    "lefteris@example.com",
			Username: "lefteris",
		},
	}

	middleware := NewAuthentication(
		cookieReader,
		sessionFinder,
		userFinder,
	)

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		user, ok := CurrentUser(r.Context())
		if !ok {
			t.Fatal("current user missing from context")
		}

		if user.ID != 42 {
			t.Fatalf(
				"user.ID = %d, want 42",
				user.ID,
			)
		}

		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(
		http.MethodGet,
		"/",
		nil,
	)

	rec := httptest.NewRecorder()

	middleware(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusOK,
		)
	}
}
func TestAuthenticateExpiredOrUnknownSessionReturnsGuest(t *testing.T) {
	tests := []struct {
		name       string
		session    model.Session
		sessionErr error
	}{
		{
			name: "expired session",
			session: model.Session{
				ID:        "session-1",
				UserID:    42,
				ExpiresAt: time.Now().UTC().Add(-time.Hour),
			},
		},
		{
			name:       "unknown session",
			sessionErr: repository.ErrSessionNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cookieReader := &fakeSessionReader{
				id: "session-1",
				ok: true,
			}

			sessionFinder := &fakeSessionFinder{
				session: tt.session,
				err:     tt.sessionErr,
			}

			userFinder := &fakeUserFinder{
				user: model.User{
					ID:       42,
					Email:    "lefteris@example.com",
					Username: "lefteris",
				},
			}

			middleware := NewAuthentication(
				cookieReader,
				sessionFinder,
				userFinder,
			)

			next := http.HandlerFunc(func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				if _, ok := CurrentUser(r.Context()); ok {
					t.Fatal("guest request has current user")
				}

				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(
				http.MethodGet,
				"/",
				nil,
			)

			rec := httptest.NewRecorder()

			middleware(next).ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf(
					"status = %d, want %d",
					rec.Code,
					http.StatusOK,
				)
			}
		})
	}
}
func TestRequireAuthRejectsGuest(t *testing.T) {
	nextCalled := false

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(
		http.MethodGet,
		"/posts/new",
		nil,
	)

	rec := httptest.NewRecorder()

	RequireAuth(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusUnauthorized,
		)
	}

	if nextCalled {
		t.Fatal("protected handler was called for guest")
	}
}

func TestRequireAuthAllowsAuthenticatedUser(t *testing.T) {
	nextCalled := false

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	user := model.User{
		ID:       42,
		Email:    "lefteris@example.com",
		Username: "lefteris",
	}

	ctx := ContextWithUser(
		context.Background(),
		user,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/posts/new",
		nil,
	).WithContext(ctx)

	rec := httptest.NewRecorder()

	RequireAuth(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusOK,
		)
	}

	if !nextCalled {
		t.Fatal("protected handler was not called")
	}
}

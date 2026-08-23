// Package middleware handles cross-cutting HTTP concerns around every route.
package middleware

import (
	"context"
	"net/http"
	"time"

	"forum/internal/model"
)

type contextKey string

const currentUserKey contextKey = "current-user"

// SessionReader extracts a candidate session identifier from a request.
type SessionReader interface {
	Read(r *http.Request) (string, bool)
}

type SessionFinder interface {
	ByID(id string) (model.Session, error)
}

type UserFinder interface {
	ByID(id int64) (model.User, error)
}

// NewAuthentication resolves a valid, unexpired session into a current user.
// Invalid cookies behave as guest requests and never expose repository errors.
func NewAuthentication(
	cookies SessionReader,
	sessions SessionFinder,
	users UserFinder,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			sessionID, ok := cookies.Read(r)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			session, err := sessions.ByID(sessionID)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			if !session.ExpiresAt.After(time.Now().UTC()) {
				next.ServeHTTP(w, r)
				return
			}

			user, err := users.ByID(session.UserID)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := ContextWithUser(
				r.Context(),
				user,
			)

			next.ServeHTTP(
				w,
				r.WithContext(ctx),
			)
		})
	}
}

// ContextWithUser attaches an authenticated user to a request context.
func ContextWithUser(
	ctx context.Context,
	user model.User,
) context.Context {
	return context.WithValue(
		ctx,
		currentUserKey,
		user,
	)
}

// CurrentUser retrieves the user installed by authentication middleware.
func CurrentUser(ctx context.Context) (model.User, bool) {
	user, ok := ctx.Value(currentUserKey).(model.User)

	return user, ok
}

// RequireAuth rejects guest access before a protected handler runs.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if _, ok := CurrentUser(r.Context()); !ok {
			http.Error(
				w,
				http.StatusText(http.StatusUnauthorized),
				http.StatusUnauthorized,
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}

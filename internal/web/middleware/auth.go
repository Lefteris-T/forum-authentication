package middleware

import (
	"context"
	"net/http"
	"time"

	"forum/internal/model"
)

type contextKey string

const currentUserKey contextKey = "current-user"

type SessionReader interface {
	Read(r *http.Request) (string, bool)
}

type SessionFinder interface {
	ByID(id string) (model.Session, error)
}

type UserFinder interface {
	ByID(id int64) (model.User, error)
}

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

func CurrentUser(ctx context.Context) (model.User, bool) {
	user, ok := ctx.Value(currentUserKey).(model.User)

	return user, ok
}

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

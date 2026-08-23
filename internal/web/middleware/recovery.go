package middleware

import (
	"log"
	"net/http"
)

// Recovery converts handler panics into a generic 500 response while retaining
// diagnostic information in server logs.
func Recovery(
	logger *log.Logger,
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recover() != nil {
				logger.Printf(
					"panic recovered method=%s path=%s",
					r.Method,
					r.URL.Path,
				)

				http.Error(
					w,
					http.StatusText(http.StatusInternalServerError),
					http.StatusInternalServerError,
				)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

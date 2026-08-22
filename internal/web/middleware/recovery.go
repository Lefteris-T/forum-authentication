package middleware

import (
	"log"
	"net/http"
)

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
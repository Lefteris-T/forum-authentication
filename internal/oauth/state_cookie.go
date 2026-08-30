package oauth

import (
	"net/http"
	"time"
)

const stateCookieMaxAge = 10 * time.Minute

func WriteStateCookie(
	w http.ResponseWriter,
	name string,
	value string,
	secure bool,
) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(stateCookieMaxAge.Seconds()),
		Expires:  time.Now().Add(stateCookieMaxAge),
	})
}

func ClearStateCookie(
	w http.ResponseWriter,
	name string,
	secure bool,
) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

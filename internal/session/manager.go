package session

import (
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Manager struct {
	cookieName string
	duration   time.Duration
	secure     bool
}

func NewManager(
	cookieName string,
	duration time.Duration,
	secure bool,
) *Manager {
	return &Manager{
		cookieName: cookieName,
		duration:   duration,
		secure:     secure,
	}
}

func (m *Manager) Create(w http.ResponseWriter) (string, error) {
	id := uuid.NewString()

	http.SetCookie(w, &http.Cookie{
		Name:     m.cookieName,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		Secure:   m.secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().UTC().Add(m.duration),
	})

	return id, nil
}

func (m *Manager) Read(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(m.cookieName)
	if err != nil {
		return "", false
	}

	if _, err := uuid.Parse(cookie.Value); err != nil {
		return "", false
	}

	return cookie.Value, true
}

func (m *Manager) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     m.cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   m.secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

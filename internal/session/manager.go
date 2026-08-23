// Package session manages opaque authentication cookies.
package session

import (
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Manager creates and validates cookie values; session ownership and expiry are
// stored server-side in SQLite.
type Manager struct {
	cookieName string
	duration   time.Duration
	secure     bool
}

// NewManager configures cookie name, lifetime, and HTTPS-only behavior.
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

// Create writes a hardened cookie and returns its UUID for server-side storage.
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

// Read accepts only a present, syntactically valid UUID cookie.
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

// Clear expires the browser cookie during logout.
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

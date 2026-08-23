package model

import "time"

// Session connects an opaque UUID cookie value to an authenticated user.
type Session struct {
	ID        string
	UserID    int64
	ExpiresAt time.Time
	CreatedAt time.Time
}

package model

import "time"

// User contains stored account identity; PasswordHash is never rendered.
type User struct {
	ID           int64
	Email        string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

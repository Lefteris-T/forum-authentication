package model

import "time"

// Post is a discussion created by a registered user.
type Post struct {
	ID        int64
	AuthorID  int64
	Title     string
	Body      string
	CreatedAt time.Time
}

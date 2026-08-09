package model

import "time"

type Post struct {
	ID        int64
	AuthorID  int64
	Title     string
	Body      string
	CreatedAt time.Time
}

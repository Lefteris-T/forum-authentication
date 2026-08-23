package model

import "time"

// Comment is a user-authored reply belonging to one post.
type Comment struct {
	ID        int64
	PostID    int64
	AuthorID  int64
	Body      string
	CreatedAt time.Time
}

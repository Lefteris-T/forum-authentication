// Package model defines the forum's persistence-independent domain data.
package model

// Category is a reusable label that can be attached to many posts.
type Category struct {
	ID   int64
	Name string
}

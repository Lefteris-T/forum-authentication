package model

// Reaction is the database representation of a like or dislike.
type Reaction int

const (
	// ReactionDislike and ReactionLike intentionally match the database CHECK
	// constraint, keeping invalid reaction states unrepresentable in storage.
	ReactionDislike Reaction = -1
	ReactionLike    Reaction = 1
)

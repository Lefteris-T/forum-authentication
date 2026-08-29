package model

import "time"

// OAuthAccount links a local forum user to an external OAuth identity.
type OAuthAccount struct {
	ID             int64
	UserID         int64
	Provider       string
	ProviderUserID string
	Email          string
	CreatedAt      time.Time
}

package oauth

import (
	"net/http"
	"time"
)

type ProviderConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string

	AuthorizationEndpoint string
	TokenEndpoint         string
	UserEndpoint          string

	Client *http.Client
}

func DefaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
	}
}

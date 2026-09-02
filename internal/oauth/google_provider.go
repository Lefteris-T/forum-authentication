package oauth

import (
	"fmt"
	"net/url"
)

type GoogleProvider struct {
	cfg ProviderConfig
}

func NewGoogleProvider(cfg ProviderConfig) *GoogleProvider {
	return &GoogleProvider{cfg: cfg}
}

func (p *GoogleProvider) AuthorizationURL(
	state string,
	challenge string,
) (string, error) {
	u, err := url.Parse(p.cfg.AuthorizationEndpoint)
	if err != nil {
		return "", fmt.Errorf("parse google authorization endpoint: %w", err)
	}

	query := u.Query()

	query.Set("client_id", p.cfg.ClientID)
	query.Set("redirect_uri", p.cfg.RedirectURL)
	query.Set("response_type", "code")
	query.Set("scope", "openid email profile")
	query.Set("state", state)
	query.Set("code_challenge", challenge)
	query.Set("code_challenge_method", "S256")

	u.RawQuery = query.Encode()

	return u.String(), nil
}

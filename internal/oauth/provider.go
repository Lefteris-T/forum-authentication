package oauth

import "context"

type User struct {
	Provider          string
	ProviderUserID    string
	VerifiedEmail     string
	SuggestedUsername string
}

type Provider interface {
	AuthorizationURL(state, challenge string) (string, error)

	ExchangeCode(
		ctx context.Context,
		code string,
		verifier string,
	) (string, error)

	FetchUser(
		ctx context.Context,
		accessToken string,
	) (User, error)
}

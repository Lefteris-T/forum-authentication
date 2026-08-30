package oauth

import (
	"crypto/sha256"
	"errors"
	"sync"
	"time"
)

var (
	ErrOAuthStateNotFound         = errors.New("oauth state not found")
	ErrOAuthStateExpired          = errors.New("oauth state expired")
	ErrOAuthStateProviderMismatch = errors.New("oauth state provider mismatch")
)

type OAuthFlow struct {
	Provider string
	Verifier string
	Expires  time.Time
}

type OAuthStateStore struct {
	mu    sync.Mutex
	flows map[[32]byte]OAuthFlow
}

func NewOAuthStateStore() *OAuthStateStore {
	return &OAuthStateStore{
		flows: make(map[[32]byte]OAuthFlow),
	}
}

func (s *OAuthStateStore) Save(
	state string,
	provider string,
	verifier string,
	expires time.Time,
) {
	key := sha256.Sum256([]byte(state))

	s.mu.Lock()
	defer s.mu.Unlock()

	s.flows[key] = OAuthFlow{
		Provider: provider,
		Verifier: verifier,
		Expires:  expires,
	}
}

func (s *OAuthStateStore) Consume(
	state string,
	provider string,
) (OAuthFlow, error) {
	key := sha256.Sum256([]byte(state))

	s.mu.Lock()
	defer s.mu.Unlock()

	flow, ok := s.flows[key]
	if !ok {
		return OAuthFlow{}, ErrOAuthStateNotFound
	}

	// consume first so even failures cannot replay this state
	delete(s.flows, key)

	if time.Now().After(flow.Expires) {
		return OAuthFlow{}, ErrOAuthStateExpired
	}

	if flow.Provider != provider {
		return OAuthFlow{}, ErrOAuthStateProviderMismatch
	}

	return flow, nil
}

package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

const randomBytes = 32

func GenerateState() (string, error) {
	b := make([]byte, randomBytes)

	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate oauth state: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

func GeneratePKCE() (verifier string, challenge string, err error) {
	b := make([]byte, randomBytes)

	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate pkce verifier: %w", err)
	}

	verifier = base64.RawURLEncoding.EncodeToString(b)
	challenge = PKCEChallenge(verifier)

	return verifier, challenge, nil
}

func PKCEChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))

	return base64.RawURLEncoding.EncodeToString(sum[:])
}

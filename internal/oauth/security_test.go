package oauth

import (
	"testing"
)

func TestGenerateState(t *testing.T) {
	state1, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState() error: %v", err)
	}

	state2, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState() second call error: %v", err)
	}

	if state1 == "" {
		t.Fatal("GenerateState() returned empty state")
	}

	if state2 == "" {
		t.Fatal("GenerateState() second call returned empty state")
	}

	if state1 == state2 {
		t.Fatal("GenerateState() returned identical values")
	}
}

func TestGeneratePKCE(t *testing.T) {
	verifier, challenge, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE() error: %v", err)
	}

	if verifier == "" {
		t.Fatal("PKCE verifier is empty")
	}

	if challenge == "" {
		t.Fatal("PKCE challenge is empty")
	}

	if verifier == challenge {
		t.Fatal("PKCE challenge must not equal verifier")
	}
}

func TestPKCEChallengeMatchesVerifier(t *testing.T) {
	verifier, challenge, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE() error: %v", err)
	}

	got := PKCEChallenge(verifier)

	if got != challenge {
		t.Fatalf(
			"PKCEChallenge() = %q, want %q",
			got,
			challenge,
		)
	}
}

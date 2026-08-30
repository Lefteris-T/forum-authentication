package oauth

import (
	"testing"
	"time"
)

func TestOAuthStateStoreSaveAndConsume(t *testing.T) {
	store := NewOAuthStateStore()

	state := "state-value"
	verifier := "pkce-verifier"

	store.Save(
		state,
		"github",
		verifier,
		time.Now().Add(10*time.Minute),
	)

	flow, err := store.Consume(state, "github")
	if err != nil {
		t.Fatalf("Consume() error: %v", err)
	}

	if flow.Provider != "github" {
		t.Errorf(
			"flow.Provider = %q, want %q",
			flow.Provider,
			"github",
		)
	}

	if flow.Verifier != verifier {
		t.Errorf(
			"flow.Verifier = %q, want %q",
			flow.Verifier,
			verifier,
		)
	}
}

func TestOAuthStateStoreIsSingleUse(t *testing.T) {
	store := NewOAuthStateStore()

	state := "state-value"

	store.Save(
		state,
		"github",
		"pkce-verifier",
		time.Now().Add(10*time.Minute),
	)

	_, err := store.Consume(state, "github")
	if err != nil {
		t.Fatalf("first Consume() error: %v", err)
	}

	_, err = store.Consume(state, "github")
	if err == nil {
		t.Fatal("second Consume() error = nil, want error")
	}
}

func TestOAuthStateStoreRejectsExpiredState(t *testing.T) {
	store := NewOAuthStateStore()

	store.Save(
		"expired-state",
		"github",
		"pkce-verifier",
		time.Now().Add(-time.Minute),
	)

	_, err := store.Consume("expired-state", "github")
	if err == nil {
		t.Fatal("Consume() error = nil, want expired-state error")
	}
}

func TestOAuthStateStoreSeparatesProviders(t *testing.T) {
	store := NewOAuthStateStore()

	store.Save(
		"state-value",
		"github",
		"pkce-verifier",
		time.Now().Add(10*time.Minute),
	)

	_, err := store.Consume("state-value", "google")
	if err == nil {
		t.Fatal("Consume() error = nil, want provider mismatch error")
	}
}

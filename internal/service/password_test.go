package service

import "testing"

func TestPasswordHashDiffersFromPlaintext(t *testing.T) {
	passwords := NewPasswordManager()

	plain := "strong-password-123"

	hash, err := passwords.Hash(plain)
	if err != nil {
		t.Fatalf("Hash() error: %v", err)
	}

	if hash == plain {
		t.Fatal("hash equals plaintext")
	}
}

func TestPasswordCompareMatchesCorrectPassword(t *testing.T) {
	passwords := NewPasswordManager()

	plain := "strong-password-123"

	hash, err := passwords.Hash(plain)
	if err != nil {
		t.Fatalf("Hash() error: %v", err)
	}

	if err := passwords.Compare(hash, plain); err != nil {
		t.Fatalf("Compare() error: %v", err)
	}
}

func TestPasswordCompareRejectsWrongPassword(t *testing.T) {
	passwords := NewPasswordManager()

	hash, err := passwords.Hash("strong-password-123")
	if err != nil {
		t.Fatalf("Hash() error: %v", err)
	}

	if err := passwords.Compare(hash, "wrong-password"); err == nil {
		t.Fatal("Compare() error = nil, want error")
	}
}
func TestPasswordHashUsesSalt(t *testing.T) {
	passwords := NewPasswordManager()

	plain := "strong-password-123"

	firstHash, err := passwords.Hash(plain)
	if err != nil {
		t.Fatalf("first Hash() error: %v", err)
	}

	secondHash, err := passwords.Hash(plain)
	if err != nil {
		t.Fatalf("second Hash() error: %v", err)
	}

	if firstHash == secondHash {
		t.Fatal("two hashes for the same password are equal")
	}
}

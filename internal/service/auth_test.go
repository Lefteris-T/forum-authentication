package service

import (
	"testing"

	"forum/internal/repository"
	"forum/internal/validation"
)

type fakeUserRepository struct {
	createCalled bool

	email        string
	username     string
	passwordHash string

	id  int64
	err error
}

func (f *fakeUserRepository) Create(
	email string,
	username string,
	passwordHash string,
) (int64, error) {
	f.createCalled = true
	f.email = email
	f.username = username
	f.passwordHash = passwordHash

	return f.id, f.err
}

type fakePasswordManager struct {
	hashCalled bool

	gotPassword string

	hash string
	err  error
}

func (f *fakePasswordManager) Hash(password string) (string, error) {
	f.hashCalled = true
	f.gotPassword = password

	return f.hash, f.err
}

func TestRegisterValidUser(t *testing.T) {
	repo := &fakeUserRepository{
		id: 42,
	}

	passwords := &fakePasswordManager{
		hash: "hashed-password",
	}

	auth := NewAuthService(repo, passwords)

	id, err := auth.Register(validation.RegistrationInput{
		Email:    "  Lefteris@Example.COM  ",
		Username: "  lefteris  ",
		Password: "strong-password-123",
	})
	if err != nil {
		t.Fatalf("Register() error: %v", err)
	}

	if id != 42 {
		t.Fatalf("Register() id = %d, want 42", id)
	}

	if !passwords.hashCalled {
		t.Fatal("password Hash() was not called")
	}

	if passwords.gotPassword != "strong-password-123" {
		t.Fatalf(
			"Hash() password = %q, want original password",
			passwords.gotPassword,
		)
	}

	if !repo.createCalled {
		t.Fatal("repository Create() was not called")
	}

	if repo.email != "lefteris@example.com" {
		t.Fatalf(
			"repository email = %q, want normalized email",
			repo.email,
		)
	}

	if repo.username != "lefteris" {
		t.Fatalf(
			"repository username = %q, want normalized username",
			repo.username,
		)
	}

	if repo.passwordHash != "hashed-password" {
		t.Fatalf(
			"repository password hash = %q, want hashed-password",
			repo.passwordHash,
		)
	}
}
func TestRegisterInvalidInputStopsEarly(t *testing.T) {
	repo := &fakeUserRepository{}

	passwords := &fakePasswordManager{
		hash: "hashed-password",
	}

	auth := NewAuthService(repo, passwords)

	_, err := auth.Register(validation.RegistrationInput{
		Email:    "not-an-email",
		Username: "lefteris",
		Password: "strong-password-123",
	})

	if err == nil {
		t.Fatal("Register() error = nil, want validation error")
	}

	if passwords.hashCalled {
		t.Fatal("password Hash() was called for invalid input")
	}

	if repo.createCalled {
		t.Fatal("repository Create() was called for invalid input")
	}
}
func TestRegisterPreservesDuplicateErrors(t *testing.T) {
	tests := []struct {
		name     string
		repoErr  error
		expected error
	}{
		{
			name:     "duplicate email",
			repoErr:  repository.ErrEmailExists,
			expected: repository.ErrEmailExists,
		},
		{
			name:     "duplicate username",
			repoErr:  repository.ErrUsernameExists,
			expected: repository.ErrUsernameExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeUserRepository{
				err: tt.repoErr,
			}

			passwords := &fakePasswordManager{
				hash: "hashed-password",
			}

			auth := NewAuthService(repo, passwords)

			_, err := auth.Register(validation.RegistrationInput{
				Email:    "lefteris@example.com",
				Username: "lefteris",
				Password: "strong-password-123",
			})

			if err != tt.expected {
				t.Fatalf(
					"Register() error = %v, want %v",
					err,
					tt.expected,
				)
			}
		})
	}
}

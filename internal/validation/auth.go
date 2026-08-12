package validation

import (
	"fmt"
	"net/mail"
	"strings"
)

const (
	minUsernameLength = 3
	maxUsernameLength = 32
	minPasswordLength = 8
	maxPasswordLength = 72
)

type RegistrationInput struct {
	Email    string
	Username string
	Password string
}

func ValidateRegistration(input RegistrationInput) (RegistrationInput, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	username := strings.TrimSpace(input.Username)

	if email == "" {
		return RegistrationInput{}, fmt.Errorf("email is required")
	}

	if _, err := mail.ParseAddress(email); err != nil {
		return RegistrationInput{}, fmt.Errorf("invalid email")
	}

	if username == "" {
		return RegistrationInput{}, fmt.Errorf("username is required")
	}

	if len(username) < minUsernameLength {
		return RegistrationInput{}, fmt.Errorf(
			"username must be at least %d characters",
			minUsernameLength,
		)
	}

	if len(username) > maxUsernameLength {
		return RegistrationInput{}, fmt.Errorf(
			"username must be at most %d characters",
			maxUsernameLength,
		)
	}

	if input.Password == "" {
		return RegistrationInput{}, fmt.Errorf("password is required")
	}

	if len(input.Password) < minPasswordLength {
		return RegistrationInput{}, fmt.Errorf(
			"password must be at least %d characters",
			minPasswordLength,
		)
	}

	if len(input.Password) > maxPasswordLength {
		return RegistrationInput{}, fmt.Errorf(
			"password must be at most %d characters",
			maxPasswordLength,
		)
	}

	return RegistrationInput{
		Email:    email,
		Username: username,
		Password: input.Password,
	}, nil
}

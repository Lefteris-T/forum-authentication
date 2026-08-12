package service

import (
	"forum/internal/validation"
)

type UserCreator interface {
	Create(
		email string,
		username string,
		passwordHash string,
	) (int64, error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
}

type AuthService struct {
	users     UserCreator
	passwords PasswordHasher
}

func NewAuthService(
	users UserCreator,
	passwords PasswordHasher,
) *AuthService {
	return &AuthService{
		users:     users,
		passwords: passwords,
	}
}

func (s *AuthService) Register(
	input validation.RegistrationInput,
) (int64, error) {
	validated, err := validation.ValidateRegistration(input)
	if err != nil {
		return 0, err
	}

	hash, err := s.passwords.Hash(validated.Password)
	if err != nil {
		return 0, err
	}

	return s.users.Create(
		validated.Email,
		validated.Username,
		hash,
	)
}

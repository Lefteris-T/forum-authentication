package service

import (
	"errors"
	"forum/internal/model"
	"forum/internal/repository"
	"forum/internal/validation"
	"time"
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

var ErrInvalidCredentials = errors.New("invalid credentials")

const dummyPasswordHash = "$2a$10$7EqJtq98hPqEX7fNZaFWoO5xDBmIf6M1q3P0RHFN6GmlCaypVH7De"

type UserFinder interface {
	ByEmail(email string) (model.User, error)
}

type PasswordComparer interface {
	Compare(hash, password string) error
}

type SessionStore interface {
	Replace(
		id string,
		userID int64,
		expiresAt time.Time,
	) error

	Delete(id string) error
}

type LoginService struct {
	users           UserFinder
	passwords       PasswordComparer
	sessions        SessionStore
	sessionDuration time.Duration
}

func NewLoginService(
	users UserFinder,
	passwords PasswordComparer,
	sessions SessionStore,
	sessionDuration time.Duration,
) *LoginService {
	return &LoginService{
		users:           users,
		passwords:       passwords,
		sessions:        sessions,
		sessionDuration: sessionDuration,
	}
}

func (s *LoginService) Login(
	input validation.LoginInput,
) (model.User, error) {
	validated, err := validation.ValidateLogin(input)
	if err != nil {
		return model.User{}, err
	}

	user, err := s.users.ByEmail(validated.Email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			_ = s.passwords.Compare(
				dummyPasswordHash,
				validated.Password,
			)

			return model.User{}, ErrInvalidCredentials
		}

		return model.User{}, err
	}

	if err := s.passwords.Compare(
		user.PasswordHash,
		validated.Password,
	); err != nil {
		return model.User{}, ErrInvalidCredentials
	}

	return user, nil
}

func (s *LoginService) CreateSession(
	id string,
	userID int64,
) error {
	expiresAt := time.Now().
		UTC().
		Add(s.sessionDuration)

	return s.sessions.Replace(
		id,
		userID,
		expiresAt,
	)
}

func (s *LoginService) Logout(id string) error {
	return s.sessions.Delete(id)
}

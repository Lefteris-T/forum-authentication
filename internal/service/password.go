package service

import "golang.org/x/crypto/bcrypt"

// PasswordManager encapsulates bcrypt so callers never handle plaintext storage.
type PasswordManager struct {
	cost int
}

// NewPasswordManager uses bcrypt's maintained default work factor.
func NewPasswordManager() PasswordManager {
	return PasswordManager{
		cost: bcrypt.DefaultCost,
	}
}

// Hash creates the value persisted in users.password_hash.
func (p PasswordManager) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		p.cost,
	)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

// Compare verifies a submitted password against a stored bcrypt hash.
func (p PasswordManager) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword(
		[]byte(hash),
		[]byte(password),
	)
}

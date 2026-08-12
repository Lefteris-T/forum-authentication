package service

import "golang.org/x/crypto/bcrypt"

type PasswordManager struct {
	cost int
}

func NewPasswordManager() PasswordManager {
	return PasswordManager{
		cost: bcrypt.DefaultCost,
	}
}

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

func (p PasswordManager) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword(
		[]byte(hash),
		[]byte(password),
	)
}

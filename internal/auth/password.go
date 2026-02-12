package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("empty password")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func ComparePassword(hash, password string) error {
	if hash == "" || password == "" {
		return errors.New("missing hash or password")
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

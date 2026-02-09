package password

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrPasswordRequired = errors.New("password is required")
	ErrPasswordMaxLen64 = errors.New("password too long, max 64 characters")
	ErrPasswordGenerate = errors.New("password generate error")
)

func HashPassword(password string, passCost int) (string, error) {
	if len(password) < 1 {
		return "", ErrPasswordRequired
	}
	if len(password) > 64 {
		return "", ErrPasswordMaxLen64
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), passCost)
	if err != nil {
		return "", ErrPasswordGenerate
	}

	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

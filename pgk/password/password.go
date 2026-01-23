package password

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string, passCost int) (string, error) {
	if len(password) < 1 {
		return "", errors.New("password is required")
	}
	if len(password) > 64 {
		return "", errors.New("password too long, max 64 characters")
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), passCost)
	if err != nil {
		return "", errors.New("password generate error")
	}

	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

package password

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type Repository struct {
	passCost int
}

func New(passCost int) *Repository {
	return &Repository{
		passCost: passCost,
	}
}

func (r *Repository) HashPassword(password string) (string, error) {
	if len(password) < 1 {
		return "", errors.New("password is required")
	}
	if len(password) > 64 {
		return "", errors.New("password too long, max 64 characters")
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), r.passCost)
	if err != nil {
		return "", errors.New("password generate error")
	}

	return string(bytes), err
}

func (r *Repository) CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

package password

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword_EmptyPassword(t *testing.T) {
	hash, err := HashPassword("", 4)

	assert.ErrorIs(t, err, ErrPasswordRequired)
	assert.Empty(t, hash)
}

func TestHashPassword_TooLongPassword(t *testing.T) {
	longPassword := string(make([]byte, 65))

	hash, err := HashPassword(longPassword, 4)

	assert.ErrorIs(t, err, ErrPasswordMaxLen64)
	assert.Empty(t, hash)
}

func TestHashPassword_ValidPassword(t *testing.T) {
	password := "testpass123"

	hash, err := HashPassword(password, 4)
	assert.NoError(t, err)

	assert.Contains(t, hash, "$2a$")
	assert.Contains(t, hash, "04$")

	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	assert.NoError(t, err)
}

func TestHashPassword_BcryptError(t *testing.T) {
	hash, err := HashPassword("testpass", 32)

	assert.ErrorIs(t, err, ErrPasswordGenerate)
	assert.Empty(t, hash)
}

func TestCheckPasswordHash_Valid(t *testing.T) {
	hash, err := HashPassword("testpass", 4)
	assert.NoError(t, err)

	result := CheckPasswordHash("testpass", hash)
	assert.True(t, result)
}

func TestCheckPasswordHash_Invalid(t *testing.T) {
	hash, _ := HashPassword("testpass", 4)

	result := CheckPasswordHash("wrongpass", hash)
	assert.False(t, result)
}

func TestHashPassword_Basic(t *testing.T) {
	tests := []struct {
		password string
		cost     int
	}{
		{"pass", 4},
		{"a" + string(make([]byte, 63)), 4}, // max len
	}

	for _, tt := range tests {
		t.Run(tt.password, func(t *testing.T) {
			hash, err := HashPassword(tt.password, tt.cost)
			assert.NoError(t, err)
			assert.NotEmpty(t, hash)
		})
	}
}

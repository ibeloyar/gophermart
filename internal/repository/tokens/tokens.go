package tokens

import (
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ibeloyar/gophermart/internal/model"
)

type Repository struct {
	secretKey     []byte
	tokenExpHours int
}

type Claims struct {
	jwt.RegisteredClaims
	model.TokenInfo
}

func New(secretKey string, tokenExpHours int) *Repository {
	return &Repository{
		secretKey:     []byte(secretKey),
		tokenExpHours: tokenExpHours,
	}
}

func (r *Repository) GenerateToken(input model.TokenInfo) (token string, err error) {
	exp := time.Duration(r.tokenExpHours) * time.Hour

	tokenData := jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
		TokenInfo: model.TokenInfo{
			ID:    input.ID,
			Login: input.Login,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	})

	return tokenData.SignedString(r.secretKey)
}

func (r *Repository) VerifyJWTToken(tokenString string) (*model.TokenInfo, error) {
	claims := &Claims{}

	tokenParts := strings.Split(tokenString, " ")
	if len(tokenParts) != 2 {
		return nil, jwt.ErrSignatureInvalid
	}
	if tokenParts[0] != "Bearer" {
		return nil, jwt.ErrInvalidType
	}

	token, err := jwt.ParseWithClaims(tokenParts[1], claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrInvalidKeyType
		}
		return r.secretKey, nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return &claims.TokenInfo, nil
}

package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type tokenDataContextKeyType string

const tokenDataContextKey = tokenDataContextKeyType("token")

type Claims[T any] struct {
	jwt.RegisteredClaims
	TokenInfo T
}

func GenerateBearerToken[T any](input T, exp time.Duration, secret string) (token string, err error) {
	tokenData := jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims[T]{
		TokenInfo: input,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	})

	token, err = tokenData.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Bearer %s", token), nil
}

func VerifyJWTBearerToken[T any](tokenString, secret string) (*T, error) {
	claims := &Claims[T]{}

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
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return &claims.TokenInfo, nil
}

func AuthBearerMiddlewareInit[T any](secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenInfo, err := VerifyJWTBearerToken[T](r.Header.Get("Authorization"), secret)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), tokenDataContextKey, tokenInfo)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetTokenInfo[T any](r *http.Request) *T {
	ctx := r.Context()

	tokenInfo, ok := ctx.Value(tokenDataContextKey).(*T)
	if !ok {
		return nil
	}

	return tokenInfo
}

func NewAuthenticatedRequest[T any](method, url string, tokenInfo *T, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, url, body)

	ctx := context.WithValue(req.Context(), tokenDataContextKey, tokenInfo)

	return req.WithContext(ctx)
}

package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

type TokenInfo struct {
	ID int64 `json:"id"`
}

func TestGenerateBearerToken_Success(t *testing.T) {
	tokenInfo := TokenInfo{ID: 123}
	token, err := GenerateBearerToken(tokenInfo, time.Hour, "secret")

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.True(t, strings.HasPrefix(token, "Bearer "))
}

func TestVerifyJWTBearerToken_Valid(t *testing.T) {
	tokenInfo := TokenInfo{ID: 123}
	tokenStr, err := GenerateBearerToken(tokenInfo, time.Hour, "secret")
	if err != nil {
		t.Fatal(err)
	}

	verified, err := VerifyJWTBearerToken[TokenInfo](tokenStr, "secret")

	assert.NoError(t, err)
	assert.Equal(t, &tokenInfo, verified)
}

func TestVerifyJWTBearerToken_InvalidFormat(t *testing.T) {
	testCases := []string{
		"invalid",
		"Bearer",
		"Bearer token without space",
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			_, err := VerifyJWTBearerToken[TokenInfo](tc, "secret")
			assert.Error(t, err)
		})
	}
}

func TestVerifyJWTBearerToken_WrongSecret(t *testing.T) {
	tokenInfo := TokenInfo{ID: 123}
	tokenStr, _ := GenerateBearerToken(tokenInfo, time.Hour, "secret")

	_, err := VerifyJWTBearerToken[TokenInfo](tokenStr, "wrong-secret")

	assert.Error(t, err)
}

func TestVerifyJWTBearerToken_Expired(t *testing.T) {
	tokenInfo := TokenInfo{ID: 123}
	tokenData := jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims[TokenInfo]{
		TokenInfo: tokenInfo,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	})
	token, _ := tokenData.SignedString([]byte("secret"))
	fullToken := fmt.Sprintf("Bearer %s", token)

	_, err := VerifyJWTBearerToken[TokenInfo](fullToken, "secret")

	assert.Error(t, err)
}

func TestAuthBearerMiddleware_ValidToken(t *testing.T) {
	tokenInfo := TokenInfo{ID: 123}
	tokenStr, _ := GenerateBearerToken(tokenInfo, time.Hour, "secret")

	middleware := AuthBearerMiddlewareInit[TokenInfo]("secret")
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := GetTokenInfo[TokenInfo](r)
		assert.Equal(t, &tokenInfo, info)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", tokenStr)
	w := httptest.NewRecorder()

	middleware(nextHandler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthBearerMiddleware_InvalidToken(t *testing.T) {
	middleware := AuthBearerMiddlewareInit[TokenInfo]("secret")
	nextHandler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})

	testCases := []struct {
		name string
		auth string
	}{
		{"no header", ""},
		{"invalid format", "invalid"},
		{"wrong secret", "Bearer wrongtoken"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Authorization", tc.auth)
			w := httptest.NewRecorder()

			middleware(nextHandler).ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestGetTokenInfo_NotFound(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	info := GetTokenInfo[TokenInfo](req)

	assert.Nil(t, info)
}

func TestGetTokenInfo_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), "key", "wrong type")
	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)

	info := GetTokenInfo[TokenInfo](req)

	assert.Nil(t, info)
}

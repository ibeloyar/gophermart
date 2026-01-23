package model

import "errors"

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const (
	ErrInternalServerMessage         = "internal server error"
	ErrInvalidLoginOrPasswordMessage = "invalid login or password"
)

var (
	ErrOrderHasBeenLoadedCurrentUser = errors.New("order has been loaded current user")
	ErrOrderHasBeenLoadedSomeUser    = errors.New("order has been loaded some user")
)

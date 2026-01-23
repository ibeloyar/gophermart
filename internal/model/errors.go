package model

import "errors"

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const (
	ErrInternalServerMessage         = "internal server error"
	ErrInvalidLoginOrPasswordMessage = "invalid login or password"
	ErrUserAlreadyExistMessage       = "user already exists"
	ErrOrdersNotFoundMessage         = "no orders found"
	ErrOrderNumberRequiredMessage    = "invalid order is required"
	ErrOrderInvalidNumberMessage     = "invalid order number"
)

var (
	ErrInvalidLoginOrPassword = errors.New(ErrInvalidLoginOrPasswordMessage)

	ErrOrderHasBeenLoadedCurrentUser = errors.New("order has been loaded current user")
	ErrOrderHasBeenLoadedSomeUser    = errors.New("order has been loaded some user")
	ErrInsufficientFunds             = errors.New("insufficient funds")
)

package service

import (
	"errors"
	"net/http"

	"github.com/ibeloyar/gophermart/internal/model"
)

const (
	minPassLen  = 4
	maxPassLen  = 64
	minLoginLen = 3
	maxLoginLen = 64

	asciiZero = 48
	asciiTen  = 57
)

func validateLoginDTO(input model.LoginDTO) error {
	if err := validateLogin(input.Password); err != nil {
		return err
	}

	if err := validatePassword(input.Password); err != nil {
		return err
	}

	return nil
}

func validateRegisterDTO(input model.RegisterDTO) error {
	if err := validateLogin(input.Password); err != nil {
		return err
	}

	if err := validatePassword(input.Password); err != nil {
		return err
	}

	return nil
}

func validateLogin(login string) error {
	if len(login) < minLoginLen || len(login) > maxLoginLen {
		return errors.New(model.ErrInvalidLoginOrPasswordMessage)
	}

	return nil
}

func validatePassword(password string) error {
	if len(password) < minPassLen || len(password) > maxPassLen {
		return errors.New(model.ErrInvalidLoginOrPasswordMessage)
	}

	return nil
}

func validateOrderNumber(number string) *model.APIError {
	if number == "" {
		return &model.APIError{
			Code:    http.StatusBadRequest,
			Message: model.ErrOrderNumberRequiredMessage,
		}
	}

	p := len(number) % 2
	sum, err := calculateLuhnSum(number, p)
	if err != nil {
		return &model.APIError{
			Code:    http.StatusUnprocessableEntity,
			Message: model.ErrOrderInvalidNumberMessage,
		}
	}

	// If the total modulo 10 is not equal to 0, then the number is invalid.
	if sum%10 != 0 {
		return &model.APIError{
			Code:    http.StatusUnprocessableEntity,
			Message: model.ErrOrderInvalidNumberMessage,
		}
	}

	return nil
}

func calculateLuhnSum(number string, parity int) (int64, error) {
	var sum int64
	for i, d := range number {
		if d < asciiZero || d > asciiTen {
			return 0, errors.New("invalid digit")
		}

		d = d - asciiZero
		// Double the value of every second digit.
		if i%2 == parity {
			d *= 2
			// If the result of this doubling operation is greater than 9.
			if d > 9 {
				// The same final result can be found by subtracting 9 from that result.
				d -= 9
			}
		}

		// Take the sum of all the digits.
		sum += int64(d)
	}

	return sum, nil
}

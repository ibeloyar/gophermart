package service

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ibeloyar/gophermart/internal/model"
	"github.com/ibeloyar/gophermart/internal/repository/pg"
	"github.com/ibeloyar/gophermart/pgk/auth"
	"github.com/ibeloyar/gophermart/pgk/password"
)

type StorageRepo interface {
	CreateUser(user model.User) (int64, error)
	GetUserByLogin(login string) *model.User
	CreateOrder(userID int64, number string) error
	GetOrdersByUserID(userID int64) ([]model.Order, error)
	GetBalanceByUserID(userID int64) (*model.Balance, error)
	SetWithdraw(userID int64, input model.SetWithdrawDTO) error
	GetWithdrawsByUserID(userID int64) ([]model.Withdraw, error)
}

type Service struct {
	storage      StorageRepo
	passwordCost int
	tokenSecret  string
	tokenExp     time.Duration
}

func New(storage StorageRepo, passwordCost int, tokenExp time.Duration, tokenSecret string) *Service {
	return &Service{
		storage:      storage,
		passwordCost: passwordCost,
		tokenExp:     tokenExp,
		tokenSecret:  tokenSecret,
	}
}

func (s *Service) Register(input model.RegisterDTO) (string, *model.APIError) {
	if err := validateRegisterDTO(input); err != nil {
		return "", &model.APIError{
			Code:    http.StatusBadRequest,
			Message: model.ErrInvalidLoginOrPasswordMessage,
		}
	}

	passwordHash, err := password.HashPassword(input.Password, s.passwordCost)
	if err != nil {
		return "", &model.APIError{
			Code:    http.StatusInternalServerError,
			Message: model.ErrInternalServerMessage,
		}
	}

	userID, err := s.storage.CreateUser(model.User{
		Login:    input.Login,
		Password: passwordHash,
	})
	if err != nil {
		if strings.Contains(err.Error(), pg.ErrIsExistCode) {
			return "", &model.APIError{
				Code:    http.StatusConflict,
				Message: model.ErrUserAlreadyExistMessage,
			}
		}
		return "", &model.APIError{
			Code:    http.StatusInternalServerError,
			Message: model.ErrInternalServerMessage,
		}
	}

	token, err := auth.GenerateBearerToken(model.TokenInfo{
		ID:    userID,
		Login: input.Login,
	}, s.tokenExp, s.tokenSecret)
	if err != nil {
		return "", &model.APIError{
			Code:    http.StatusInternalServerError,
			Message: model.ErrInternalServerMessage,
		}
	}

	return token, nil
}

func (s *Service) Login(input model.LoginDTO) (string, *model.APIError) {
	if err := validateLoginDTO(input); err != nil {
		return "", &model.APIError{
			Code:    http.StatusBadRequest,
			Message: model.ErrInvalidLoginOrPasswordMessage,
		}
	}

	user := s.storage.GetUserByLogin(input.Login)
	if user == nil {
		return "", &model.APIError{
			Code:    http.StatusUnauthorized,
			Message: model.ErrInvalidLoginOrPasswordMessage,
		}
	}

	if !password.CheckPasswordHash(input.Password, user.Password) {
		return "", &model.APIError{
			Code:    http.StatusUnauthorized,
			Message: model.ErrInvalidLoginOrPasswordMessage,
		}
	}

	token, err := auth.GenerateBearerToken(model.TokenInfo{
		ID:    user.ID,
		Login: user.Login,
	}, s.tokenExp, s.tokenSecret)
	if err != nil {
		return "", &model.APIError{
			Code:    http.StatusInternalServerError,
			Message: model.ErrInternalServerMessage,
		}
	}

	return token, nil
}

func (s *Service) CreateOrder(userID int64, orderNumber string) *model.APIError {
	if err := validateOrderNumber(orderNumber); err != nil {
		return err
	}

	err := s.storage.CreateOrder(userID, orderNumber)
	if err != nil {
		// номер заказа уже был загружен этим пользователем;
		if errors.Is(err, model.ErrOrderHasBeenLoadedCurrentUser) {
			return &model.APIError{
				Code:    http.StatusOK,
				Message: model.ErrOrderHasBeenLoadedCurrentUser.Error(),
			}
		}
		// номер заказа уже был загружен другим пользователем;
		if errors.Is(err, model.ErrOrderHasBeenLoadedSomeUser) {
			return &model.APIError{
				Code:    http.StatusConflict,
				Message: model.ErrOrderHasBeenLoadedSomeUser.Error(),
			}
		}
		return &model.APIError{
			Code:    http.StatusInternalServerError,
			Message: model.ErrInternalServerMessage,
		}
	}

	return nil
}

func (s *Service) GetOrders(userID int64) ([]model.Order, *model.APIError) {
	orders, err := s.storage.GetOrdersByUserID(userID)
	if err != nil {
		return nil, &model.APIError{
			Code:    http.StatusInternalServerError,
			Message: model.ErrInternalServerMessage,
		}
	}

	if len(orders) == 0 {
		return nil, &model.APIError{
			Code:    http.StatusNoContent,
			Message: model.ErrOrdersNotFoundMessage,
		}
	}

	return orders, nil
}

func (s *Service) GetBalance(userID int64) (*model.Balance, *model.APIError) {
	balance, err := s.storage.GetBalanceByUserID(userID)
	if err != nil {
		return nil, &model.APIError{
			Code:    http.StatusInternalServerError,
			Message: model.ErrInternalServerMessage,
		}
	}

	return balance, nil
}

func (s *Service) SetWithdraw(userID int64, input model.SetWithdrawDTO) *model.APIError {
	err := s.storage.SetWithdraw(userID, input)
	if err != nil {
		if errors.Is(err, model.ErrInsufficientFunds) {
			return &model.APIError{
				Code:    http.StatusPaymentRequired,
				Message: model.ErrInsufficientFundsMessage,
			}
		}
		return &model.APIError{
			Code:    http.StatusInternalServerError,
			Message: model.ErrInternalServerMessage,
		}
	}

	return nil
}

func (s *Service) GetWithdraws(userID int64) ([]model.Withdraw, *model.APIError) {
	withdraws, err := s.storage.GetWithdrawsByUserID(userID)
	if err != nil {
		return nil, &model.APIError{
			Code:    http.StatusInternalServerError,
			Message: model.ErrInternalServerMessage,
		}
	}

	return withdraws, nil
}

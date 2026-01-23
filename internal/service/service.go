package service

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ibeloyar/gophermart/internal/model"
	"github.com/ibeloyar/gophermart/internal/repository/pg"
	"github.com/ibeloyar/gophermart/pgk/auth"
)

type StorageRepo interface {
	GetUserByLogin(login string) *model.User
	CreateUser(user model.User) error
	CreateOrder(userID int64, number string) error
	GetOrdersByUserID(userID int64) ([]model.Order, error)
	GetBalanceByUserID(userID int64) (*model.Balance, error)
	SetWithdraw(userID int64, input model.SetWithdrawDTO) error
	GetWithdrawsByUserID(userID int64) ([]model.Withdraw, error)
}

type PasswordRepo interface {
	HashPassword(password string) (string, error)
	CheckPasswordHash(password, hash string) bool
}

//type TokensRepo interface {
//	GenerateToken(input model.TokenInfo) (token string, err error)
//	VerifyJWTToken(tokenString string) (*model.TokenInfo, error)
//}

type AccrualRepo interface {
	GetAccrual(orderNumber string) (*model.Accrual, error)
}

type Service struct {
	storage  StorageRepo
	password PasswordRepo
	//tokens   TokensRepo

	tokenSecret string
	tokenExp    time.Duration
}

func New(s StorageRepo, p PasswordRepo, tokenExp time.Duration, tokenSecret string) *Service {
	return &Service{
		storage:  s,
		password: p,
		//tokens:   t,

		tokenExp:    tokenExp,
		tokenSecret: tokenSecret,
	}
}

func (s *Service) Login(input model.LoginDTO) (string, *model.APIError) {
	if err := loginDTOValidate(input); err != nil {
		return "", &model.APIError{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		}
	}

	user := s.storage.GetUserByLogin(input.Login)
	if user == nil {
		return "", &model.APIError{
			Code:    http.StatusUnauthorized,
			Message: model.ErrInvalidLoginOrPasswordMessage,
		}
	}

	if !s.password.CheckPasswordHash(input.Password, user.Password) {
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

func (s *Service) Register(input model.RegisterDTO) (string, *model.APIError) {
	passwordHash, err := s.password.HashPassword(input.Password)
	if err != nil {
		return "", &model.APIError{
			Code:    http.StatusInternalServerError,
			Message: model.ErrInternalServerMessage,
		}
	}

	if err := s.storage.CreateUser(model.User{
		Login:    input.Login,
		Password: passwordHash,
	}); err != nil {
		if strings.Contains(err.Error(), pg.ErrIsExistCode) {
			return "", &model.APIError{
				Code:    http.StatusConflict,
				Message: "user already exists",
			}
		}
		return "", &model.APIError{
			Code:    http.StatusInternalServerError,
			Message: model.ErrInternalServerMessage,
		}
	}

	user := s.storage.GetUserByLogin(input.Login)
	if user == nil {
		return "", &model.APIError{
			Code:    http.StatusInternalServerError,
			Message: model.ErrInternalServerMessage,
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
			Message: "no orders found",
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

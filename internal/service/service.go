package service

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ibeloyar/gophermart/internal/model"
	"github.com/ibeloyar/gophermart/internal/repository/pg"
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

type TokensRepo interface {
	GenerateToken(input model.TokenInfo) (token string, err error)
	VerifyJWTToken(tokenString string) (*model.TokenInfo, error)
}

type AccrualRepo interface {
	GetAccrual(orderNumber string) (*model.Accrual, error)
}

type Service struct {
	storage  StorageRepo
	password PasswordRepo
	tokens   TokensRepo
}

func New(s StorageRepo, p PasswordRepo, t TokensRepo) *Service {
	return &Service{
		storage:  s,
		password: p,
		tokens:   t,
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

	token, err := s.tokens.GenerateToken(model.TokenInfo{
		ID:    user.ID,
		Login: user.Login,
	})
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

	token, err := s.tokens.GenerateToken(model.TokenInfo{
		ID:    user.ID,
		Login: user.Login,
	})
	if err != nil {
		return "", &model.APIError{
			Code:    http.StatusInternalServerError,
			Message: model.ErrInternalServerMessage,
		}
	}

	return token, nil
}

func (s *Service) CreateOrder(token, orderNumber string) *model.APIError {
	//- `401` — пользователь не аутентифицирован;
	tokenInfo, err := s.tokens.VerifyJWTToken(token)
	if err != nil {
		return &model.APIError{
			Code:    http.StatusUnauthorized,
			Message: err.Error(),
		}
	}

	//- `400` — неверный формат запроса;
	//- `422` — неверный формат номера заказа;
	if err := validateOrderNumber(orderNumber); err != nil {
		return err
	}

	//- `500` — внутренняя ошибка сервера.
	err = s.storage.CreateOrder(tokenInfo.ID, orderNumber)
	if err != nil {
		//- `200` — номер заказа уже был загружен этим пользователем;
		if errors.Is(err, model.ErrOrderHasBeenLoadedCurrentUser) {
			return &model.APIError{
				Code:    http.StatusOK,
				Message: model.ErrOrderHasBeenLoadedCurrentUser.Error(),
			}
		}
		//- `409` — номер заказа уже был загружен другим пользователем;
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

	//- `202` — новый номер заказа принят в обработку;
	return nil
}

func (s *Service) GetOrders(token string) ([]model.Order, *model.APIError) {
	tokenInfo, err := s.tokens.VerifyJWTToken(token)
	if err != nil {
		return nil, &model.APIError{
			Code:    http.StatusUnauthorized,
			Message: err.Error(),
		}
	}

	orders, err := s.storage.GetOrdersByUserID(tokenInfo.ID)
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

func (s *Service) GetBalance(token string) (*model.Balance, *model.APIError) {
	tokenInfo, err := s.tokens.VerifyJWTToken(token)
	if err != nil {
		return nil, &model.APIError{
			Code:    http.StatusUnauthorized,
			Message: err.Error(),
		}
	}

	balance, err := s.storage.GetBalanceByUserID(tokenInfo.ID)
	if err != nil {
		return nil, &model.APIError{
			Code:    http.StatusInternalServerError,
			Message: model.ErrInternalServerMessage,
		}
	}

	return balance, nil
}

func (s *Service) SetWithdraw(token string, input model.SetWithdrawDTO) *model.APIError {
	tokenInfo, err := s.tokens.VerifyJWTToken(token)
	if err != nil {
		return &model.APIError{
			Code:    http.StatusUnauthorized,
			Message: err.Error(),
		}
	}

	err = s.storage.SetWithdraw(tokenInfo.ID, input)
	if err != nil {
		return &model.APIError{
			Code:    http.StatusInternalServerError,
			Message: model.ErrInternalServerMessage,
		}
	}

	return nil
}

func (s *Service) GetWithdraws(token string) ([]model.Withdraw, *model.APIError) {
	tokenInfo, err := s.tokens.VerifyJWTToken(token)
	if err != nil {
		return nil, &model.APIError{
			Code:    http.StatusUnauthorized,
			Message: err.Error(),
		}
	}

	withdraws, err := s.storage.GetWithdrawsByUserID(tokenInfo.ID)
	if err != nil {
		return nil, &model.APIError{
			Code:    http.StatusInternalServerError,
			Message: model.ErrInternalServerMessage,
		}
	}

	return withdraws, nil
}

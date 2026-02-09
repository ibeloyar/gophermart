package service

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/ibeloyar/gophermart/internal/model"
	"github.com/ibeloyar/gophermart/internal/repository/pg"
	"github.com/ibeloyar/gophermart/pgk/password"
	"github.com/stretchr/testify/assert"

	mockPG "github.com/ibeloyar/gophermart/internal/repository/pg/mocks"
)

const validOrderNumber = "27220117637"

func TestService_Register_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mockPG.NewMockStorageRepo(ctrl)
	svc := New(mockStorage, 3, 1*time.Hour, "secret")

	input := model.RegisterDTO{
		Login:    "testuser",
		Password: "testpass123",
	}

	mockStorage.EXPECT().
		CreateUser(gomock.Any()).
		Return(int64(123), nil).
		Times(1)

	token, apiErr := svc.Register(input)

	assert.Nil(t, apiErr)
	assert.NotEmpty(t, token)
}

func TestService_Register_CreateUserConflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mockPG.NewMockStorageRepo(ctrl)
	svc := New(mockStorage, 3, 1*time.Hour, "secret")

	input := model.RegisterDTO{
		Login:    "testuser",
		Password: "testpass123",
	}

	mockStorage.EXPECT().
		CreateUser(gomock.Any()).
		Return(int64(0), errors.New(pg.ErrIsExistCode))

	token, apiErr := svc.Register(input)

	assert.Empty(t, token)
	assert.NotNil(t, apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.Code)
	assert.Equal(t, model.ErrUserAlreadyExistMessage, apiErr.Message)
}

func TestService_Register_CreateUserError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mockPG.NewMockStorageRepo(ctrl)
	svc := New(mockStorage, 3, 1*time.Hour, "secret")

	input := model.RegisterDTO{
		Login:    "testuser",
		Password: "testpass123",
	}

	mockStorage.EXPECT().
		CreateUser(gomock.Any()).
		Return(int64(0), errors.New("database connection failed"))

	token, apiErr := svc.Register(input)

	assert.Empty(t, token)
	assert.NotNil(t, apiErr)
	assert.Equal(t, http.StatusInternalServerError, apiErr.Code)
	assert.Equal(t, model.ErrInternalServerMessage, apiErr.Message)
}

func TestService_Register_UserExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mockPG.NewMockStorageRepo(ctrl)
	svc := New(mockStorage, 3, 1*time.Hour, "secret")

	input := model.RegisterDTO{
		Login:    "testuser",
		Password: "testpass123",
	}

	mockStorage.EXPECT().
		CreateUser(gomock.Any()).
		Return(int64(0), errors.New(pg.ErrIsExistCode)).
		Times(1)

	token, apiErr := svc.Register(input)

	assert.Empty(t, token)
	assert.NotNil(t, apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.Code)
}

func TestService_Login_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mockPG.NewMockStorageRepo(ctrl)
	svc := New(mockStorage, 3, 1*time.Hour, "secret")

	input := model.LoginDTO{
		Login:    "testuser",
		Password: "testpass123",
	}

	hashedPass, err := password.HashPassword("testpass123", svc.passwordCost)
	assert.NoError(t, err)

	user := &model.User{
		ID:       123,
		Login:    "testuser",
		Password: hashedPass,
	}

	mockStorage.EXPECT().
		GetUserByLogin("testuser").
		Return(user).
		Times(1)

	token, apiErr := svc.Login(input)

	assert.Nil(t, apiErr)
	assert.NotEmpty(t, token)
}

func TestService_Login_InvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mockPG.NewMockStorageRepo(ctrl)
	svc := New(mockStorage, 3, 1*time.Hour, "secret")

	input := model.LoginDTO{
		Login:    "testuser",
		Password: "test",
	}

	mockStorage.EXPECT().
		GetUserByLogin("testuser").
		Return(&model.User{Login: "testuser", Password: "not_test"}).
		Times(1)

	token, apiErr := svc.Login(input)

	assert.Empty(t, token)
	assert.NotNil(t, apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.Code)
	mockStorage.EXPECT().GetUserByLogin(gomock.Any()).Times(0)
}

func TestService_Login_UserNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mockPG.NewMockStorageRepo(ctrl)
	svc := New(mockStorage, 3, 1*time.Hour, "secret")

	input := model.LoginDTO{
		Login:    "nonexistent",
		Password: "testpass123",
	}

	mockStorage.EXPECT().
		GetUserByLogin("nonexistent").
		Return(nil).
		Times(1)

	token, apiErr := svc.Login(input)

	assert.Empty(t, token)
	assert.NotNil(t, apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.Code)
}

func TestService_CreateOrder_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mockPG.NewMockStorageRepo(ctrl)
	svc := New(mockStorage, 3, 1*time.Hour, "secret")

	mockStorage.EXPECT().
		CreateOrder(int64(123), validOrderNumber).
		Return(nil).
		Times(1)

	apiErr := svc.CreateOrder(123, validOrderNumber)

	assert.Nil(t, apiErr)
}

func TestService_CreateOrder_InvalidOrderNumber(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mockPG.NewMockStorageRepo(ctrl)
	svc := New(mockStorage, 3, 1*time.Hour, "secret")

	invalidOrderNumber := "1"

	apiErr := svc.CreateOrder(123, invalidOrderNumber)

	assert.NotNil(t, apiErr)
	mockStorage.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).Times(0)
}

func TestService_CreateOrder_AlreadyLoadedCurrentUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mockPG.NewMockStorageRepo(ctrl)
	svc := New(mockStorage, 3, 1*time.Hour, "secret")

	mockStorage.EXPECT().
		CreateOrder(int64(123), validOrderNumber).
		Return(model.ErrOrderHasBeenLoadedCurrentUser).
		Times(1)

	apiErr := svc.CreateOrder(123, validOrderNumber)

	assert.NotNil(t, apiErr)
	assert.Equal(t, http.StatusOK, apiErr.Code)
	assert.Equal(t, model.ErrOrderHasBeenLoadedCurrentUser.Error(), apiErr.Message)
}

func TestService_CreateOrder_AlreadyLoadedOtherUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mockPG.NewMockStorageRepo(ctrl)
	svc := &Service{storage: mockStorage}

	mockStorage.EXPECT().
		CreateOrder(int64(123), validOrderNumber).
		Return(model.ErrOrderHasBeenLoadedSomeUser).
		Times(1)

	apiErr := svc.CreateOrder(123, validOrderNumber)

	assert.NotNil(t, apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.Code)
	assert.Equal(t, model.ErrOrderHasBeenLoadedSomeUser.Error(), apiErr.Message)
}

func TestService_CreateOrder_StorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mockPG.NewMockStorageRepo(ctrl)
	svc := &Service{storage: mockStorage}

	unexpectedErr := errors.New("unexpected database error")

	mockStorage.EXPECT().
		CreateOrder(int64(123), validOrderNumber).
		Return(unexpectedErr).
		Times(1)

	apiErr := svc.CreateOrder(123, validOrderNumber)

	assert.NotNil(t, apiErr)
	assert.Equal(t, http.StatusInternalServerError, apiErr.Code)
	assert.Equal(t, model.ErrInternalServerMessage, apiErr.Message)
}

func TestService_GetOrders_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mockPG.NewMockStorageRepo(ctrl)
	svc := New(mockStorage, 3, 1*time.Hour, "secret")

	orders := []model.Order{{Number: validOrderNumber}}

	mockStorage.EXPECT().
		GetOrdersByUserID(int64(123)).
		Return(orders, nil).
		Times(1)

	result, apiErr := svc.GetOrders(123)

	assert.Nil(t, apiErr)
	assert.Equal(t, orders, result)
}

func TestService_GetOrders_Empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mockPG.NewMockStorageRepo(ctrl)
	svc := New(mockStorage, 3, 1*time.Hour, "secret")

	mockStorage.EXPECT().
		GetOrdersByUserID(int64(123)).
		Return(nil, nil).
		Times(1)

	_, apiErr := svc.GetOrders(123)

	assert.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNoContent, apiErr.Code)
}

func TestService_GetOrders_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mockPG.NewMockStorageRepo(ctrl)
	svc := New(mockStorage, 3, 1*time.Hour, "secret")

	mockStorage.EXPECT().
		GetOrdersByUserID(int64(123)).
		Return(nil, errors.New("db error")).
		Times(1)

	_, apiErr := svc.GetOrders(123)

	assert.NotNil(t, apiErr)
	assert.Equal(t, http.StatusInternalServerError, apiErr.Code)
}

func TestService_GetBalance_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mockPG.NewMockStorageRepo(ctrl)
	svc := New(mockStorage, 3, 1*time.Hour, "secret")

	balance := &model.Balance{Current: 100.5, Withdrawn: 50.0}

	mockStorage.EXPECT().
		GetBalanceByUserID(int64(123)).
		Return(balance, nil).
		Times(1)

	result, apiErr := svc.GetBalance(123)

	assert.Nil(t, apiErr)
	assert.Equal(t, balance, result)
}

func TestService_GetBalance_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mockPG.NewMockStorageRepo(ctrl)
	svc := New(mockStorage, 3, 1*time.Hour, "secret")

	mockStorage.EXPECT().
		GetBalanceByUserID(int64(123)).
		Return(nil, errors.New("db error")).
		Times(1)

	_, apiErr := svc.GetBalance(123)

	assert.NotNil(t, apiErr)
	assert.Equal(t, http.StatusInternalServerError, apiErr.Code)
}

func TestService_SetWithdraw_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mockPG.NewMockStorageRepo(ctrl)
	svc := New(mockStorage, 3, 1*time.Hour, "secret")

	input := model.SetWithdrawDTO{
		Order: validOrderNumber,
		Sum:   10.5,
	}

	mockStorage.EXPECT().
		SetWithdraw(int64(123), input).
		Return(nil).
		Times(1)

	apiErr := svc.SetWithdraw(123, input)

	assert.Nil(t, apiErr)
}

func TestService_SetWithdraw_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mockPG.NewMockStorageRepo(ctrl)
	svc := New(mockStorage, 3, 1*time.Hour, "secret")

	input := model.SetWithdrawDTO{
		Order: validOrderNumber,
		Sum:   10.5,
	}

	mockStorage.EXPECT().
		SetWithdraw(int64(123), input).
		Return(errors.New("db error")).
		Times(1)

	apiErr := svc.SetWithdraw(123, input)

	assert.NotNil(t, apiErr)
	assert.Equal(t, http.StatusInternalServerError, apiErr.Code)
}

func TestService_GetWithdraws_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mockPG.NewMockStorageRepo(ctrl)
	svc := New(mockStorage, 3, 1*time.Hour, "secret")

	withdraws := []model.Withdraw{{OrderNumber: validOrderNumber}}

	mockStorage.EXPECT().
		GetWithdrawsByUserID(int64(123)).
		Return(withdraws, nil).
		Times(1)

	result, apiErr := svc.GetWithdraws(123)

	assert.Nil(t, apiErr)
	assert.Equal(t, withdraws, result)
}

func TestService_GetWithdraws_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mockPG.NewMockStorageRepo(ctrl)
	svc := New(mockStorage, 3, 1*time.Hour, "secret")

	mockStorage.EXPECT().
		GetWithdrawsByUserID(int64(123)).
		Return(nil, errors.New("db error")).
		Times(1)

	_, apiErr := svc.GetWithdraws(123)

	assert.NotNil(t, apiErr)
	assert.Equal(t, http.StatusInternalServerError, apiErr.Code)
}

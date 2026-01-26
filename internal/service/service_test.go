package service

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/ibeloyar/gophermart/internal/model"
	"github.com/ibeloyar/gophermart/internal/repository/pg"
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

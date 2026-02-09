package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/ibeloyar/gophermart/internal/model"
	"github.com/ibeloyar/gophermart/pgk/auth"
	"github.com/stretchr/testify/assert"

	service "github.com/ibeloyar/gophermart/internal/service/mocks"
)

func TestController_Register_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := service.NewMockService(ctrl)
	controller := New(mockSvc, nil)

	input := model.RegisterDTO{
		Login:    "testuser",
		Password: "testpass123",
	}

	mockSvc.EXPECT().
		Register(input).
		Return("Bearer token123", nil).
		Times(1)

	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	controller.Register(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Bearer token123", w.Header().Get("Authorization"))
}

func TestController_Login_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := service.NewMockService(ctrl)
	controller := New(mockSvc, nil)

	input := model.LoginDTO{
		Login:    "testuser",
		Password: "testpass123",
	}

	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	mockSvc.EXPECT().
		Login(input).
		Return("Bearer token123", nil).
		Times(1)

	controller.Login(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Bearer token123", w.Header().Get("Authorization"))
}

func TestController_CreateOrder_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := service.NewMockService(ctrl)
	controller := New(mockSvc, nil)

	orderNumber := "order-123"
	userID := int64(123)

	mockSvc.EXPECT().
		CreateOrder(userID, orderNumber).
		Return(nil).
		Times(1)

	body, _ := json.Marshal(orderNumber)
	req := auth.NewAuthenticatedRequest(http.MethodPost, "/orders", &model.TokenInfo{ID: userID}, bytes.NewReader(body))
	//req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	//ctx := context.WithValue(req.Context(), auth.TokenDataContextKey, &model.TokenInfo{ID: userID})
	//req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	controller.CreateOrder(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
}

func TestController_CreateOrder_AlreadyLoadedCurrentUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := service.NewMockService(ctrl)
	controller := New(mockSvc, nil)

	orderNumber := "order-456"
	userID := int64(123)
	apiErr := &model.APIError{
		Code:    http.StatusOK,
		Message: "order already loaded by current user",
	}

	mockSvc.EXPECT().
		CreateOrder(userID, orderNumber).
		Return(apiErr).
		Times(1)

	body, _ := json.Marshal(orderNumber)
	req := auth.NewAuthenticatedRequest(http.MethodPost, "/orders", &model.TokenInfo{ID: userID}, bytes.NewReader(body))
	//req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	//ctx := context.WithValue(req.Context(), auth.TokenDataContextKey, &model.TokenInfo{ID: userID})
	//req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	controller.CreateOrder(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestController_CreateOrder_ConflictOtherUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := service.NewMockService(ctrl)
	controller := New(mockSvc, nil)

	orderNumber := "order-789"
	userID := int64(123)
	apiErr := &model.APIError{
		Code:    http.StatusConflict,
		Message: "order already loaded by other user",
	}

	mockSvc.EXPECT().
		CreateOrder(userID, orderNumber).
		Return(apiErr).
		Times(1)

	body, _ := json.Marshal(orderNumber)
	req := auth.NewAuthenticatedRequest(http.MethodPost, "/orders", &model.TokenInfo{ID: userID}, bytes.NewReader(body))
	//req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	//ctx := context.WithValue(req.Context(), auth.TokenDataContextKey, &model.TokenInfo{ID: userID})
	//req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	controller.CreateOrder(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestController_GetOrders_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := service.NewMockService(ctrl)
	controller := New(mockSvc, nil)

	userID := int64(123)
	orders := []model.Order{{Number: "order-123"}}

	mockSvc.EXPECT().
		GetOrders(userID).
		Return(orders, nil).
		Times(1)

	req := auth.NewAuthenticatedRequest(http.MethodGet, "/orders", &model.TokenInfo{ID: userID}, nil)
	//req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	//ctx := context.WithValue(req.Context(), auth.TokenDataContextKey, &model.TokenInfo{ID: userID})
	//req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	controller.GetOrders(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestController_GetOrders_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := service.NewMockService(ctrl)
	controller := New(mockSvc, nil)

	userID := int64(123)
	apiErr := &model.APIError{
		Code:    http.StatusInternalServerError,
		Message: "database error",
	}

	mockSvc.EXPECT().
		GetOrders(userID).
		Return(nil, apiErr).
		Times(1)

	req := auth.NewAuthenticatedRequest(http.MethodGet, "/orders", &model.TokenInfo{ID: userID}, nil)
	//req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	//ctx := context.WithValue(req.Context(), auth.TokenDataContextKey, &model.TokenInfo{ID: userID})
	//req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	controller.GetOrders(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestController_GetBalance_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := service.NewMockService(ctrl)
	controller := New(mockSvc, nil)

	userID := int64(123)
	balance := &model.Balance{Current: 100.5, Withdrawn: 50.0}

	mockSvc.EXPECT().
		GetBalance(userID).
		Return(balance, nil).
		Times(1)

	req := auth.NewAuthenticatedRequest(http.MethodGet, "/balance", &model.TokenInfo{ID: userID}, nil)
	//req := httptest.NewRequest(http.MethodGet, "/balance", nil)
	//ctx := context.WithValue(req.Context(), auth.TokenDataContextKey, &model.TokenInfo{ID: userID})
	//req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	controller.GetBalance(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestController_SetWithdrawal_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := service.NewMockService(ctrl)
	controller := New(mockSvc, nil)

	userID := int64(123)
	withdraw := model.SetWithdrawDTO{Order: "order-123", Sum: 10.5}

	mockSvc.EXPECT().
		SetWithdraw(userID, withdraw).
		Return(nil).
		Times(1)

	body, _ := json.Marshal(withdraw)

	//req := httptest.NewRequest(http.MethodPost, "/withdraw", bytes.NewReader(body))
	//ctx := context.WithValue(req.Context(), auth.TokenDataContextKey, &model.TokenInfo{ID: userID})
	//req = req.WithContext(ctx)
	req := auth.NewAuthenticatedRequest(http.MethodPost, "/withdraw", &model.TokenInfo{ID: userID}, bytes.NewReader(body))
	w := httptest.NewRecorder()

	controller.SetWithdrawal(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestController_GetWithdrawals_Empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := service.NewMockService(ctrl)
	controller := New(mockSvc, nil)

	userID := int64(123)

	mockSvc.EXPECT().
		GetWithdraws(userID).
		Return([]model.Withdraw{}, nil).
		Times(1)

	req := auth.NewAuthenticatedRequest(http.MethodGet, "/withdrawals", &model.TokenInfo{ID: userID}, nil)
	//req := httptest.NewRequest(http.MethodGet, "/withdrawals", nil)
	//ctx := context.WithValue(req.Context(), auth.TokenDataContextKey, &model.TokenInfo{ID: userID})
	//req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	controller.GetWithdrawals(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestController_GetWithdrawals_WithData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := service.NewMockService(ctrl)
	controller := New(mockSvc, nil)

	userID := int64(123)
	withdrawals := []model.Withdraw{{OrderNumber: "order-123"}}

	mockSvc.EXPECT().
		GetWithdraws(userID).
		Return(withdrawals, nil).
		Times(1)

	req := auth.NewAuthenticatedRequest(http.MethodGet, "/withdrawals", &model.TokenInfo{ID: userID}, nil)
	//req := httptest.NewRequest(http.MethodGet, "/withdrawals", nil)
	//ctx := context.WithValue(req.Context(), auth.TokenDataContextKey, &model.TokenInfo{ID: userID})
	//req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	controller.GetWithdrawals(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

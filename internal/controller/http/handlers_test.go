package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/ibeloyar/gophermart/internal/model"
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

//func TestController_Register_InvalidBody(t *testing.T) {
//	ctrl := gomock.NewController(t)
//	defer ctrl.Finish()
//
//	mockSvc := service.NewMockService(ctrl)
//	controller := New(mockSvc, nil)
//
//	invalidBody := bytes.NewReader([]byte("invalid json"))
//	req := httptest.NewRequest(http.MethodPost, "/register", invalidBody)
//	w := httptest.NewRecorder()
//
//	controller.Register(w, req)
//
//	assert.Equal(t, http.StatusInternalServerError, w.Code)
//}

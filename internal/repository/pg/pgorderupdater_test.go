package pg

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ibeloyar/gophermart/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

type MockRepository struct {
	mock.Mock
	*Repository
}

// Тест для getAccrual
func TestRepository_getAccrual_Success(t *testing.T) {
	repo := &Repository{accrualAddress: "http://localhost:8080"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(model.Accrual{Status: "PROCESSED", Accrual: 100.5})
	}))
	defer server.Close()

	repo.accrualAddress = server.URL

	ctx := context.Background()
	accrual, err := repo.getAccrual(ctx, "order123")

	assert.NoError(t, err)
	assert.NotNil(t, accrual)
	assert.Equal(t, model.OrderStatus("PROCESSED"), accrual.Status)
}

// Тест для getAccrual с ошибкой HTTP
func TestRepository_getAccrual_HTTPError(t *testing.T) {
	repo := &Repository{accrualAddress: "http://localhost:8080"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	repo.accrualAddress = server.URL

	ctx := context.Background()
	_, err := repo.getAccrual(ctx, "order123")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Bad Gateway")
}

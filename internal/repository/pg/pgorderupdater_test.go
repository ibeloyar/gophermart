package pg

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ibeloyar/gophermart/internal/model"
	"github.com/ibeloyar/gophermart/pgk/retryablehttp"
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

func TestRepository_getAccrual(t *testing.T) {
	tests := []struct {
		name           string
		mockStatus     int
		mockBody       string
		mockRetryCount int
		setupServer    func(*httptest.Server)
		wantStatus     model.OrderStatus
		wantAccrual    float32
		wantErr        bool
	}{
		{
			name:        "успешный ответ",
			mockStatus:  http.StatusOK,
			mockBody:    `{"status": "PROCESSED", "accrual": 100.5}`,
			wantStatus:  "PROCESSED",
			wantAccrual: 100.5,
		},
		{
			name:        "INVALID статус",
			mockStatus:  http.StatusOK,
			mockBody:    `{"status": "INVALID", "accrual": 0}`,
			wantStatus:  "INVALID",
			wantAccrual: 0,
		},
		{
			name:       "невалидный JSON",
			mockStatus: http.StatusOK,
			mockBody:   `{invalid json}`,
			wantErr:    true,
		},
		{
			name:       "4xx ошибка (не retry)",
			mockStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:           "5xx retry -> успех",
			mockStatus:     http.StatusServiceUnavailable,
			mockRetryCount: 1,
			setupServer: func(server *httptest.Server) {
				// Имитируем retry: первый 503, второй 200
			},
			wantStatus:  "PROCESSED",
			wantAccrual: 50.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &Repository{
				accrualAddress: "http://localhost:8080",
				retryClient:    retryablehttp.NewRetryableClient(retryablehttp.RetryConfig{}),
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatus)
				w.Write([]byte(tt.mockBody))
			}))
			defer server.Close()

			if tt.setupServer != nil {
				tt.setupServer(server)
				return
			}

			repo.accrualAddress = server.URL

			accrual, err := repo.getAccrual(context.Background(), "order123")

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, accrual)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, accrual)
				if tt.wantStatus != "" {
					assert.Equal(t, tt.wantStatus, accrual.Status)
				}
				if tt.wantAccrual != 0 {
					assert.Equal(t, tt.wantAccrual, accrual.Accrual)
				}
			}
		})
	}
}

//
//func TestRepository_getAccrual_Success(t *testing.T) {
//	repo := &Repository{accrualAddress: "http://localhost:8080"}
//
//	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		w.WriteHeader(http.StatusOK)
//		json.NewEncoder(w).Encode(model.Accrual{Status: "PROCESSED", Accrual: 100.5})
//	}))
//	defer server.Close()
//
//	repo.accrualAddress = server.URL
//
//	ctx := context.Background()
//	accrual, err := repo.getAccrual(ctx, "order123")
//
//	assert.NoError(t, err)
//	assert.NotNil(t, accrual)
//	assert.Equal(t, model.OrderStatus("PROCESSED"), accrual.Status)
//}
//
//func TestRepository_getAccrual_HTTPError(t *testing.T) {
//	repo := &Repository{accrualAddress: "http://localhost:8080"}
//
//	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		w.WriteHeader(http.StatusBadGateway)
//	}))
//	defer server.Close()
//
//	repo.accrualAddress = server.URL
//
//	ctx := context.Background()
//	_, err := repo.getAccrual(ctx, "order123")
//
//	assert.Error(t, err)
//	assert.Contains(t, err.Error(), "Bad Gateway")
//}

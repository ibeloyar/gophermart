package retryablehttp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRetryableClient_Defaults(t *testing.T) {
	config := RetryConfig{}
	client := NewRetryableClient(config)

	assert.Equal(t, 3, client.retryConfig.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, client.retryConfig.BaseDelay)
	assert.Equal(t, 5*time.Second, client.retryConfig.MaxDelay)
	assert.Equal(t, 100*time.Millisecond, client.retryConfig.MaxJitter)
}

func TestIsRetryable_NetworkError(t *testing.T) {
	client := NewRetryableClient(RetryConfig{})
	result := client.isRetryable(nil, fmt.Errorf("network error"))
	assert.True(t, result)
}

func TestIsRetryable_ServerErrors(t *testing.T) {
	client := NewRetryableClient(RetryConfig{})

	tests := []int{500, 502, 503, 504, 599, 429, 408}
	for _, code := range tests {
		t.Run(fmt.Sprintf("Status_%d", code), func(t *testing.T) {
			resp := httptest.NewRecorder()
			resp.WriteHeader(code)
			result := client.isRetryable(resp.Result(), nil)
			assert.True(t, result)
		})
	}
}

func TestIsRetryable_SuccessNoRetry(t *testing.T) {
	client := NewRetryableClient(RetryConfig{})

	tests := []int{200, 201, 400, 401, 403, 404}
	for _, code := range tests {
		t.Run(fmt.Sprintf("Status_%d", code), func(t *testing.T) {
			resp := httptest.NewRecorder()
			resp.WriteHeader(code)
			result := client.isRetryable(resp.Result(), nil)
			assert.False(t, result)
		})
	}
}

func TestBackoffDelay_Calculation(t *testing.T) {
	config := RetryConfig{
		BaseDelay: 100 * time.Millisecond,
		MaxDelay:  2 * time.Second,
		MaxJitter: 50 * time.Millisecond,
	}
	client := &RetryableClient{retryConfig: config}

	delay0 := client.backoffDelay(0)
	assert.GreaterOrEqual(t, delay0, 100*time.Millisecond)
	assert.Less(t, delay0, 150*time.Millisecond)

	delay3 := client.backoffDelay(3)
	assert.GreaterOrEqual(t, delay3, 800*time.Millisecond)
	assert.LessOrEqual(t, delay3, 2*time.Second+50*time.Millisecond)
}

func TestDo_SuccessFirstTry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewRetryableClient(RetryConfig{})
	req, _ := http.NewRequest("GET", server.URL, nil)
	ctx := context.Background()

	result, err := client.Do(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 200, result.StatusCode)
}

func TestDo_RetryServerError(t *testing.T) {
	var attempts int32 // Используем atomic counter если нужно
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 1 {
			w.WriteHeader(503)
			return
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewRetryableClient(RetryConfig{MaxRetries: 1})
	req, _ := http.NewRequest("GET", server.URL, nil)
	ctx := context.Background()

	result, err := client.Do(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, int32(2), attempts)
}

func TestDo_RetryRateLimit(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(429)
			return
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewRetryableClient(RetryConfig{MaxRetries: 1})
	req, _ := http.NewRequest("GET", server.URL, nil)
	ctx := context.Background()

	result, err := client.Do(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, int32(2), attempts)
}

func TestDo_MaxRetriesExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server.Close()

	client := NewRetryableClient(RetryConfig{MaxRetries: 1})
	req, _ := http.NewRequest("GET", server.URL, nil)
	ctx := context.Background()

	result, err := client.Do(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "последняя попытка failed")
	assert.NotNil(t, result)
}

func TestDo_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := NewRetryableClient(RetryConfig{})
	req, _ := http.NewRequest("GET", server.URL, nil)

	result, err := client.Do(ctx, req)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, result)
}

func TestDo_ContextTimeout(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(200)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	client := NewRetryableClient(RetryConfig{})
	req, _ := http.NewRequest("GET", server.URL, nil)

	result, err := client.Do(ctx, req)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Nil(t, result)
}

func TestRetryConfig_CustomValues(t *testing.T) {
	config := RetryConfig{
		MaxRetries: 5,
		BaseDelay:  500 * time.Millisecond,
		MaxDelay:   30 * time.Second,
		MaxJitter:  200 * time.Millisecond,
	}

	client := NewRetryableClient(config)
	assert.Equal(t, config, client.retryConfig)
}

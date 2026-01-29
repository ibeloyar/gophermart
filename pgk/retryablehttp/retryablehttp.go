package retryablehttp

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

type RetryConfig struct {
	MaxRetries int           // Максимум попыток (по умолчанию 3)
	BaseDelay  time.Duration // Базовая задержка (по умолчанию 100ms)
	MaxDelay   time.Duration // Максимальная задержка (по умолчанию 5s)
	MaxJitter  time.Duration // Максимальный jitter (по умолчанию 100ms)
}

type RetryableClient struct {
	client      *http.Client
	retryConfig RetryConfig
}

func NewRetryableClient(config RetryConfig) *RetryableClient {
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.BaseDelay == 0 {
		config.BaseDelay = 100 * time.Millisecond
	}
	if config.MaxDelay == 0 {
		config.MaxDelay = 5 * time.Second
	}
	if config.MaxJitter == 0 {
		config.MaxJitter = 100 * time.Millisecond
	}

	return &RetryableClient{
		client:      &http.Client{},
		retryConfig: config,
	}
}

// isRetryable определяет, нужно ли делать retry
func (c *RetryableClient) isRetryable(resp *http.Response, err error) bool {
	if err != nil {
		// Сетевые ошибки всегда retry
		return true
	}

	if resp == nil {
		return false
	}

	// Retry для серверных ошибок и rate limiting
	statusCode := resp.StatusCode
	return statusCode == 0 || // Неизвестная ошибка
		(statusCode >= 500 && statusCode <= 599) || // 5xx, 502 Bad Gateway, 503 Service Unavailable, 504 Gateway Timeout etc
		statusCode == 429 || // Too Many Requests
		statusCode == 408 // Request Timeout
}

func (c *RetryableClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		// Проверка отмены контекста
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		resp, err = c.client.Do(req)

		// Успех
		if err == nil && !c.isRetryable(resp, nil) {
			return resp, nil
		}

		// Закрываем тело ответа при retry
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}

		// Последняя попытка - возвращаем ошибку
		if attempt == c.retryConfig.MaxRetries {
			if resp != nil {
				return resp, fmt.Errorf("последняя попытка failed: %s", resp.Status)
			}
			return nil, fmt.Errorf("последняя попытка failed: %v", err)
		}

		// Exponential backoff + jitter
		delay := c.backoffDelay(attempt)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}

	return nil, fmt.Errorf("unexpected error")
}

// backoffDelay вычисляет задержку с экспоненциальным ростом и jitter
func (c *RetryableClient) backoffDelay(attempt int) time.Duration {
	backoff := time.Duration(1<<uint(attempt)) * c.retryConfig.BaseDelay
	if backoff > c.retryConfig.MaxDelay {
		backoff = c.retryConfig.MaxDelay
	}

	jitter := time.Duration(rand.Int63n(int64(c.retryConfig.MaxJitter)))
	return backoff + jitter
}

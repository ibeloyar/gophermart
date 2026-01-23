package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// readBody читает и парсит JSON и Text/Plain тело запроса в структуру T
func readBody[T any](r *http.Request) (T, error) {
	var body T

	contentType := r.Header.Get("Content-Type")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return body, fmt.Errorf("failed to read request body: %w", err)
	}
	defer r.Body.Close()

	if strings.HasPrefix(contentType, "text/plain") {
		switch any(body).(type) {
		case string:
			if len(bodyBytes) == 0 {
				return body, nil
			}

			return any(string(bodyBytes)).(T), nil
		default:
			return body, fmt.Errorf("failed to read request body: %s", contentType)
		}
	}

	if strings.HasPrefix(contentType, "application/json") {
		if err := json.Unmarshal(bodyBytes, &body); err != nil {
			return body, fmt.Errorf("failed to read request body %s: %w", contentType, err)
		}
	}

	return body, nil
}

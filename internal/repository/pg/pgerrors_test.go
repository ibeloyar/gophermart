package pg

import (
	"errors"
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestPostgresErrorClassifier_Classify_Nil(t *testing.T) {
	classifier := NewPostgresErrorClassifier()

	result := classifier.Classify(nil)

	assert.Equal(t, NonRetriable, result)
}

func TestPostgresErrorClassifier_Classify_NonPQError(t *testing.T) {
	classifier := NewPostgresErrorClassifier()
	customErr := errors.New("custom error")

	result := classifier.Classify(customErr)

	assert.Equal(t, NonRetriable, result)
}

func TestPostgresErrorClassifier_Classify_ConnectionErrors_Retriable(t *testing.T) {
	classifier := NewPostgresErrorClassifier()
	testCases := []string{
		"08000", "08001", "08003", "08004", "08006", "08007", // Класс 08
	}

	for _, code := range testCases {
		t.Run(code, func(t *testing.T) {
			pqErr := &pq.Error{Code: pq.ErrorCode(code)}
			result := classifier.Classify(pqErr)
			assert.Equal(t, Retriable, result)
		})
	}
}

func TestPostgresErrorClassifier_Classify_TransactionErrors_Retriable(t *testing.T) {
	classifier := NewPostgresErrorClassifier()
	testCases := []string{"40000", "40001", "40P01"} // Класс 40

	for _, code := range testCases {
		t.Run(code, func(t *testing.T) {
			pqErr := &pq.Error{Code: pq.ErrorCode(code)}
			result := classifier.Classify(pqErr)
			assert.Equal(t, Retriable, result)
		})
	}
}

func TestPostgresErrorClassifier_Classify_OperatorErrors_Retriable(t *testing.T) {
	classifier := NewPostgresErrorClassifier()
	testCases := []string{"57P03"} // Класс 57

	for _, code := range testCases {
		t.Run(code, func(t *testing.T) {
			pqErr := &pq.Error{Code: pq.ErrorCode(code)}
			result := classifier.Classify(pqErr)
			assert.Equal(t, Retriable, result)
		})
	}
}

func TestPostgresErrorClassifier_Classify_DataErrors_NonRetriable(t *testing.T) {
	classifier := NewPostgresErrorClassifier()
	testCases := []string{"22000", "22004"} // Класс 22

	for _, code := range testCases {
		t.Run(code, func(t *testing.T) {
			pqErr := &pq.Error{Code: pq.ErrorCode(code)}
			result := classifier.Classify(pqErr)
			assert.Equal(t, NonRetriable, result)
		})
	}
}

func TestPostgresErrorClassifier_Classify_IntegrityErrors_NonRetriable(t *testing.T) {
	classifier := NewPostgresErrorClassifier()
	testCases := []string{
		"23000", "23001", "23502", "23503", ErrIsExistCode, "23514", // Класс 23
	}

	for _, code := range testCases {
		t.Run(code, func(t *testing.T) {
			pqErr := &pq.Error{Code: pq.ErrorCode(code)}
			result := classifier.Classify(pqErr)
			assert.Equal(t, NonRetriable, result)
		})
	}
}

func TestPostgresErrorClassifier_Classify_SyntaxErrors_NonRetriable(t *testing.T) {
	classifier := NewPostgresErrorClassifier()
	testCases := []string{"42601", "42P01", "42703", "42P02", "42P03"} // Класс 42

	for _, code := range testCases {
		t.Run(code, func(t *testing.T) {
			pqErr := &pq.Error{Code: pq.ErrorCode(code)}
			result := classifier.Classify(pqErr)
			assert.Equal(t, NonRetriable, result)
		})
	}
}

func TestPostgresErrorClassifier_Classify_UnknownError_NonRetriable(t *testing.T) {
	classifier := NewPostgresErrorClassifier()

	testCases := []string{
		"00000", "12345", "99999", "ABCDE", // несуществующие коды
	}

	for _, code := range testCases {
		t.Run(code, func(t *testing.T) {
			pqErr := &pq.Error{Code: pq.ErrorCode(code)}
			result := classifier.Classify(pqErr)
			assert.Equal(t, NonRetriable, result)
		})
	}
}

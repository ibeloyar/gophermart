package pg

import (
	"errors"

	"github.com/lib/pq"
)

type ErrorClassification int

const (
	NonRetriable ErrorClassification = iota
	Retriable

	ErrIsExistCode = "23505"
)

type PostgresErrorClassifier struct{}

func NewPostgresErrorClassifier() *PostgresErrorClassifier {
	return &PostgresErrorClassifier{}
}

func (c *PostgresErrorClassifier) Classify(err error) ErrorClassification {
	if err == nil {
		return NonRetriable
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return classifyPgError(pqErr)
	}

	// По умолчанию считаем ошибку неповторяемой
	return NonRetriable
}

func classifyPgError(pqErr *pq.Error) ErrorClassification {
	// Коды ошибок PostgreSQL: https://www.postgresql.org/docs/current/errcodes-appendix.html

	switch pqErr.Code {
	// Класс 08 - Ошибки соединения
	case "08000", "08001", "08003", "08004", "08006", "08007":
		return Retriable

	// Класс 40 - Откат транзакции
	case "40000", "40001", "40P01":
		return Retriable

	// Класс 57 - Ошибка оператора
	case "57P03":
		return Retriable
	}

	// Класс 22 - Ошибки данных
	switch pqErr.Code {
	case "22000", "22004":
		return NonRetriable
	}

	// Класс 23 - Нарушение ограничений целостности
	switch pqErr.Code {
	case "23000", "23001", "23502", "23503", ErrIsExistCode, "23514":
		return NonRetriable
	}

	// Класс 42 - Синтаксические ошибки
	switch pqErr.Code {
	case "42601", "42P01", "42703", "42P02", "42P03":
		return NonRetriable
	}

	// По умолчанию считаем ошибку неповторяемой
	return NonRetriable
}

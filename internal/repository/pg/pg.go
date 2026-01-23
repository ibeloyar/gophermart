package pg

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/ibeloyar/gophermart/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

const (
	migrationsTable = "schema_migrations"
	schemaName      = "public"
	migrationsPath  = "./migrations"

	maxAttempts = 3
)

type Repository struct {
	db             *sql.DB
	accrualAddress string
	classifier     *PostgresErrorClassifier

	stopAccrualChan chan struct{}
}

func New(databaseURI, accrualAddress string) (*Repository, error) {
	pool, err := pgxpool.New(context.Background(), databaseURI)
	if err != nil {
		return nil, err
	}

	db := stdlib.OpenDBFromPool(pool)

	driver, err := postgres.WithInstance(db, &postgres.Config{
		MigrationsTable: migrationsTable,
		SchemaName:      schemaName,
	})
	if err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		return nil, err
	}

	m, err := migrate.NewWithDatabaseInstance("file://"+absPath, "postgres", driver)
	if err != nil {
		return nil, err
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, err
	}

	return &Repository{
		db:              db,
		accrualAddress:  accrualAddress,
		stopAccrualChan: make(chan struct{}),
		classifier:      NewPostgresErrorClassifier(),
	}, nil
}

func (r *Repository) GetUserByLogin(login string) *model.User {
	var user model.User
	query := `SELECT * FROM users WHERE login = $1`

	err := r.executeWithRetryConnection(func(db *sql.DB) error {
		row := db.QueryRow(query, login)
		return row.Scan(&user.ID, &user.Login, &user.Password, &user.CreatedAt)
	})

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return nil
	}

	return &user
}

func (r *Repository) CreateUser(user model.User) (int64, error) {
	var userID int64

	err := r.executeWithRetryConnection(func(db *sql.DB) error {
		query := `INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id`

		row := db.QueryRow(query, user.Login, user.Password)
		
		return row.Scan(&userID)
	})

	return userID, err
}

func (r *Repository) CreateOrder(userID int64, number string) error {
	return r.executeWithRetryConnection(func(db *sql.DB) error {
		queryOrder := `SELECT user_id, number FROM orders WHERE number = $1`

		var order model.Order
		row := db.QueryRow(queryOrder, number)
		err := row.Scan(&order.UserID, &order.Number)

		if order.UserID != 0 && order.Number != "" {
			if order.UserID == userID {
				return model.ErrOrderHasBeenLoadedCurrentUser
			}
			if order.UserID != userID {
				return model.ErrOrderHasBeenLoadedSomeUser
			}
		}

		query := `INSERT INTO orders (user_id, number) VALUES ($1, $2)`

		_, err = db.Exec(query, userID, number)

		return err
	})
}

func (r *Repository) GetOrdersByUserID(userID int64) ([]model.Order, error) {
	result := make([]model.Order, 0)

	err := r.executeWithRetryConnection(func(db *sql.DB) error {
		query := `SELECT number, status, accrual, uploaded_at 
		FROM orders WHERE user_id = $1 
        ORDER BY uploaded_at DESC`

		rows, err := db.Query(query, userID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var order model.Order
			if err := rows.Scan(&order.Number, &order.Status, &order.Accrual, &order.UploadedAt); err != nil {
				return err
			}

			result = append(result, order)
		}

		return rows.Err()
	})

	if err != nil {
		return result, err
	}

	return result, nil
}

func (r *Repository) GetBalanceByUserID(userID int64) (*model.Balance, error) {
	var balance model.Balance
	query := `SELECT COALESCE(SUM(amount), 0) AS current, 
		COALESCE(SUM(CASE WHEN amount < 0 THEN ABS(amount) ELSE 0 END), 0) AS withdrawn
		FROM balance WHERE user_id = $1`

	if err := r.executeWithRetryConnection(func(db *sql.DB) error {
		row := db.QueryRow(query, userID)
		return row.Scan(&balance.Current, &balance.Withdrawn)
	}); err != nil {
		return nil, err
	}

	return &balance, nil
}

func (r *Repository) SetWithdraw(userID int64, input model.SetWithdrawDTO) error {
	return r.executeWithRetryConnection(func(db *sql.DB) error {
		ctx := context.Background()

		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}

		// 1. Сначала блокируем строки баланса
		_, err = tx.ExecContext(ctx, `
		SELECT id FROM balance
		WHERE user_id = $1
		FOR UPDATE
	`, userID)
		if err != nil {
			_ = tx.Rollback()
			return err
		}

		// 2. Проверяем текущий баланс
		var current float64
		err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0) AS current
		FROM balance
		WHERE user_id = $1
	`, userID).Scan(&current)
		if err != nil {
			_ = tx.Rollback()
			return err
		}

		// 3. Проверка остатка
		absAmount := math.Abs(input.Sum)
		if current < absAmount {
			_ = tx.Rollback()
			return errors.New("insufficient funds")
		}

		// 4. Вставляем новую запись и возвращаем данные
		_, err = tx.ExecContext(ctx, `
		INSERT INTO balance (user_id, order_number, amount)
		VALUES ($1, $2, $3)
	`, userID, input.Order, -absAmount)

		if err != nil {
			_ = tx.Rollback()
			return err
		}

		if err = tx.Commit(); err != nil {
			return err
		}

		return nil
	})
}

func (r *Repository) GetWithdrawsByUserID(userID int64) ([]model.Withdraw, error) {
	result := make([]model.Withdraw, 0)

	err := r.executeWithRetryConnection(func(db *sql.DB) error {
		query := `SELECT id, user_id, order_number, ABS(amount), uploaded_at
			FROM balance WHERE user_id = $1 AND amount < 0 ORDER BY uploaded_at DESC`

		rows, err := db.Query(query, userID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var withdraw model.Withdraw
			if err := rows.Scan(&withdraw.ID, &withdraw.UserID, &withdraw.OrderNumber, &withdraw.Amount, &withdraw.UploadedAt); err != nil {
				return err
			}

			result = append(result, withdraw)
		}

		return rows.Err()
	})

	if err != nil {
		return result, err
	}

	return result, nil
}

func (r *Repository) Shutdown() error {
	r.StopOrdersAccrualUpdater()

	return r.db.Close()
}

func (r *Repository) executeWithRetryConnection(operation func(*sql.DB) error) error {
	err := operation(r.db)
	if err == nil {
		return nil
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if r.classifier.Classify(err) != Retriable {
			return err
		}

		delay := getAttemptDelay(attempt)
		time.Sleep(delay)

		err = operation(r.db)
		if err == nil {
			return nil
		}

		lastErr = err
	}

	return lastErr // Возвращаем последнюю ошибку после 3 попыток
}

func getAttemptDelay(attempt int) time.Duration {
	switch attempt {
	case 0:
		return 1 * time.Second
	case 1:
		return 3 * time.Second
	default: // attempt >= 2
		return 5 * time.Second
	}
}

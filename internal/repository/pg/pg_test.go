package pg

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ibeloyar/gophermart/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestRepository_GetUserByLogin_Found(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Repository{db: db, classifier: NewPostgresErrorClassifier()}

	// ✅ Правильные колонки из запроса SELECT * FROM users
	createdAt := time.Now()
	rows := sqlmock.NewRows([]string{"id", "login", "password", "created_at"}).
		AddRow(123, "testuser", "hashed", createdAt)

	mock.ExpectQuery("SELECT \\* FROM users WHERE login = \\$1").
		WithArgs("testuser").
		WillReturnRows(rows)

	result := repo.GetUserByLogin("testuser")

	assert.NotNil(t, result)
	assert.Equal(t, int64(123), result.ID)
	assert.Equal(t, "testuser", result.Login)
	assert.Equal(t, "hashed", result.Password)
	assert.WithinDuration(t, createdAt, result.CreatedAt, time.Second)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetUserByLogin_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Repository{db: db, classifier: NewPostgresErrorClassifier()}

	mock.ExpectQuery("SELECT \\* FROM users WHERE login = \\$1").
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	result := repo.GetUserByLogin("nonexistent")

	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_CreateUser_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Repository{db: db, classifier: NewPostgresErrorClassifier()}

	mock.ExpectQuery("INSERT INTO users \\(login, password\\) VALUES \\(\\$1, \\$2\\) RETURNING id").
		WithArgs("testuser", "hashed").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(123)))

	userID, err := repo.CreateUser(model.User{Login: "testuser", Password: "hashed"})

	assert.NoError(t, err)
	assert.Equal(t, int64(123), userID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_CreateOrder_CurrentUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Repository{db: db, classifier: NewPostgresErrorClassifier()}

	mock.ExpectQuery("SELECT user_id, number FROM orders WHERE number = \\$1").
		WithArgs("order123").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "number"}).
			AddRow(int64(123), "order123"))

	err = repo.CreateOrder(123, "order123")

	assert.ErrorIs(t, err, model.ErrOrderHasBeenLoadedCurrentUser)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetOrdersByUserID_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Repository{db: db, classifier: NewPostgresErrorClassifier()}

	mock.ExpectQuery("SELECT number, status, accrual, uploaded_at FROM orders WHERE user_id = \\$1 ORDER BY uploaded_at DESC").
		WithArgs(int64(123)).
		WillReturnRows(sqlmock.NewRows([]string{"number", "status", "accrual", "uploaded_at"}))

	orders, err := repo.GetOrdersByUserID(123)

	assert.NoError(t, err)
	assert.Len(t, orders, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_CreateOrder_OtherUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Repository{db: db, classifier: NewPostgresErrorClassifier()}

	mock.ExpectQuery("SELECT user_id, number FROM orders WHERE number = \\$1").
		WithArgs("order123").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "number"}).
			AddRow(int64(456), "order123"))

	err = repo.CreateOrder(123, "order123")

	assert.ErrorIs(t, err, model.ErrOrderHasBeenLoadedSomeUser)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_CreateOrder_NewOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Repository{db: db, classifier: NewPostgresErrorClassifier()}

	mock.ExpectQuery("SELECT user_id, number FROM orders WHERE number = \\$1").
		WithArgs("neworder").
		WillReturnError(sql.ErrNoRows)

	mock.ExpectExec("INSERT INTO orders \\(user_id, number\\) VALUES \\(\\$1, \\$2\\)").
		WithArgs(int64(123), "neworder").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.CreateOrder(123, "neworder")

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetBalanceByUserID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Repository{db: db, classifier: NewPostgresErrorClassifier()}

	mock.ExpectQuery(`SELECT COALESCE\(SUM\(amount\), 0\) AS current, COALESCE\(SUM\(CASE WHEN amount < 0 THEN ABS\(amount\) ELSE 0 END\), 0\) AS withdrawn FROM balance WHERE user_id = \$1`).
		WithArgs(int64(123)).
		WillReturnRows(sqlmock.NewRows([]string{"current", "withdrawn"}).
			AddRow(float32(100.5), float32(50.0))) // ✅ float32 для model.Balance

	balance, err := repo.GetBalanceByUserID(123)

	assert.NoError(t, err)
	assert.Equal(t, float32(100.5), balance.Current)
	assert.Equal(t, float32(50.0), balance.Withdrawn)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetWithdrawsByUserID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Repository{db: db, classifier: NewPostgresErrorClassifier()}

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "user_id", "order_number", "amount", "uploaded_at"}).
		AddRow(int64(1), int64(123), "order123", 10.5, now)

	mock.ExpectQuery(`SELECT id, user_id, order_number, ABS\(amount\), uploaded_at FROM balance WHERE user_id = \$1 AND amount < 0 ORDER BY uploaded_at DESC`).
		WithArgs(int64(123)).
		WillReturnRows(rows)

	withdraws, err := repo.GetWithdrawsByUserID(123)

	assert.NoError(t, err)
	assert.Len(t, withdraws, 1)
	assert.Equal(t, int64(1), withdraws[0].ID)
	assert.Equal(t, "order123", withdraws[0].OrderNumber)
	assert.Equal(t, 10.5, withdraws[0].Amount)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_getAttemptDelay(t *testing.T) {
	tests := []struct {
		attempt int
		delay   time.Duration
	}{
		{0, 1 * time.Second},
		{1, 3 * time.Second},
		{2, 5 * time.Second},
		{3, 5 * time.Second},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			delay := getAttemptDelay(tt.attempt)
			assert.Equal(t, tt.delay, delay)
		})
	}
}

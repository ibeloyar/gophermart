package pg

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/ibeloyar/gophermart/internal/model"
)

var numWorkers = runtime.NumCPU()

func (r *Repository) processOrderWorker(ctx context.Context, order model.Order) {
	accrual, err := r.getAccrual(ctx, order.Number)
	if err != nil {
		r.lg.Errorf("getting accruals error: %v", err)
		return
	}

	if accrual != nil {
		if err := r.updateOrderStatusAndAccrual(ctx,
			order.UserID,
			order.Number,
			accrual.Status,
			accrual.Accrual,
		); err != nil {
			r.lg.Errorf("updating order status error: %v", err)
		}
	}
}

// RunOrdersAccrualUpdater - запускает обновление
func (r *Repository) RunOrdersAccrualUpdater() {
	ticker := time.NewTicker(5 * time.Second)

	go func() {
		for {
			select {
			case <-ticker.C:
				orders, err := r.getOrdersWithNewOrProcessingStatus()
				if err != nil {
					// "ошибка получения заказов NEW/PROCESSING"
					continue
				}

				if len(orders) > 0 {
					tasks := make(chan model.Order, len(orders))
					var wg sync.WaitGroup

					wg.Add(numWorkers)
					for i := 0; i < numWorkers; i++ {
						go func() {
							defer wg.Done()
							for order := range tasks {
								r.processOrderWorker(context.Background(), order)
							}
						}()
					}

					for _, order := range orders {
						tasks <- order
					}
					close(tasks)
					wg.Wait()
				}

			case <-r.stopAccrualChan:
				ticker.Stop()
				return
			}
		}
	}()
}

func (r *Repository) StopOrdersAccrualUpdater() {
	if r.stopAccrualChan != nil {
		close(r.stopAccrualChan)
		r.stopAccrualChan = nil
	}
}

// getAccrual - получить данные по начислению баллов для указанного заказа
func (r *Repository) getAccrual(ctx context.Context, orderNumber string) (*model.Accrual, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", r.accrualAddress+"/api/orders/"+orderNumber, nil)

	if err != nil {
		return nil, err
	}

	response, err := r.retryClient.Do(ctx, req)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("accrual update request failed: %s", http.StatusText(response.StatusCode))
	}

	var accrual model.Accrual
	err = json.NewDecoder(response.Body).Decode(&accrual)

	if err != nil {
		return nil, err
	}

	return &accrual, nil
}

func (r *Repository) getOrdersWithNewOrProcessingStatus() ([]model.Order, error) {
	result := make([]model.Order, 0)

	err := r.executeWithRetryConnection(func(db *sql.DB) error {
		query := `SELECT user_id, number, status, accrual, uploaded_at 
		FROM orders WHERE status = 'NEW' OR status = 'PROCESSING'`

		rows, err := db.Query(query)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var order model.Order
			if err := rows.Scan(&order.UserID, &order.Number, &order.Status, &order.Accrual, &order.UploadedAt); err != nil {
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

// updateOrderStatusAndAccrual - обновление статуса заказа и суммы начислений
func (r *Repository) updateOrderStatusAndAccrual(ctx context.Context, userID int64, orderNumber string, status model.OrderStatus, accrual float32) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `UPDATE orders SET status = $1, accrual = $2 WHERE number = $3`,
		status,
		accrual,
		orderNumber,
	)

	if status == "PROCESSED" && accrual > 0 {
		_, err = tx.ExecContext(ctx, `INSERT INTO balance (user_id, order_number, amount) VALUES ($1, $2, $3)`,
			userID,
			orderNumber,
			accrual,
		)

		if err != nil {
			return err
		}
	}

	if err != nil {
		return err
	}

	return tx.Commit()
}

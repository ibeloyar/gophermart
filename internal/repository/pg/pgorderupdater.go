package pg

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ibeloyar/gophermart/internal/model"
)

// RunOrdersAccrualUpdater - запускает обновление
func (r *Repository) RunOrdersAccrualUpdater() {
	ticker := time.NewTicker(5 * time.Second)

	go func() {
		for {
			r.workerPool.pauseMu.Lock()
			if r.workerPool.paused {
				r.workerPool.pauseCond.Wait() // БЛОКИРУЕМОСЬ до resume
				r.workerPool.pauseMu.Unlock()
				continue
			}
			r.workerPool.pauseMu.Unlock()

			select {
			case <-ticker.C:
				orders, err := r.getOrdersWithNewOrProcessingStatus()
				if err != nil {
					r.lg.Errorf("getOrdersWithNewOrProcessingStatus error: %v", err)
					continue
				}

				if len(orders) > 0 {
					r.workerPool.wg.Add(r.workerPool.numWorkers)
					for i := 0; i < r.workerPool.numWorkers; i++ {
						go func() {
							defer r.workerPool.wg.Done()
							for order := range r.workerPool.jobsQueue {
								r.worker(r.workerPool.ctx, order)
							}
						}()
					}

					for _, order := range orders {
						r.workerPool.jobsQueue <- order
					}

					close(r.workerPool.jobsQueue)

					r.workerPool.wg.Wait()
				}
			case <-r.stopAccrualChan:
				ticker.Stop()
				return
			}
		}
	}()
}

func (r *Repository) StopOrdersAccrualUpdater() {
	timeout := 4 * time.Second

	if r.stopAccrualChan != nil {
		close(r.stopAccrualChan)
		r.stopAccrualChan = nil
	}

	ctx, cancel := context.WithTimeout(r.shutdownCtx, timeout)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		r.workerPool.shutdown()
	}()

	select {
	case <-done:
		r.lg.Info("Graceful shutdown completed")
	case <-ctx.Done():
		r.lg.Warn("Force shutdown after timeout")
		r.shutdownCancel()
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
		if response != nil && response.StatusCode == http.StatusTooManyRequests {
			retryAfter := getRetryAfter(response) // из заголовка
			r.workerPool.pausePoolWithTimer(retryAfter)
			response.Body.Close()
			return nil, fmt.Errorf("rate limited: %v", retryAfter)
		}
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

func getRetryAfter(resp *http.Response) time.Duration {
	if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
		if seconds, err := strconv.ParseInt(retryAfter, 10, 64); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return 60 * time.Second // дефолт
}

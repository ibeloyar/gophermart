package pg

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ibeloyar/gophermart/internal/model"
)

func (r *Repository) RunOrdersAccrualUpdater() {
	//ctx, cancel := context.WithTimeout(context.Background(), 500*time.Second)

	ticker := time.NewTicker(5 * time.Second)

	go func() {
		for {
			select {
			case <-ticker.C:
				orders, err := r.getOrdersWithNewOrProcessingStatus()
				if err != nil {
					log.Printf("failed to get orders with NEW/PROCESSING status: %v", err)
					continue
				}

				for _, order := range orders {
					accrual, err := r.getAccrual(context.Background(), order.Number)
					if err != nil {
						log.Printf("failed to get accrual for order %s: %v", order.Number, err)
					}

					if accrual != nil {
						if err := r.updateOrderStatusAndAccrual(context.Background(), order.UserID, order.Number, accrual.Status, accrual.Accrual); err != nil {
							log.Printf("failed to update order status for %s: %v", order.Number, err)
						}
					}
				}

			case <-r.stopAccrualChan:
				//cancel()
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

	client := http.Client{}

	response, err := client.Do(req)
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
			_ = tx.Rollback()
			return err
		}
	}

	if err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

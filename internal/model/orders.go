package model

type OrderStatus string

const (
	OrderStatusNew        OrderStatus = "NEW"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusInvalid    OrderStatus = "INVALID"
	OrderStatusProcessed  OrderStatus = "PROCESSED"
)

type Order struct {
	UserID     int64       `json:"-"`
	Number     string      `json:"number"`
	Status     OrderStatus `json:"status"`
	Accrual    float64     `json:"accrual"`
	UploadedAt string      `json:"uploaded_at"`
}

type GetOrdersResponse = []Order

type Accrual struct {
	Order   string      `json:"order"`
	Status  OrderStatus `json:"status"`
	Accrual float32     `json:"accrual,omitempty"`
}

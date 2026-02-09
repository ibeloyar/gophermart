package model

type Withdraw struct {
	ID          int64   `json:"-"`
	UserID      int64   `json:"-"`
	OrderNumber string  `json:"order"`
	Amount      float64 `json:"sum"`
	UploadedAt  string  `json:"processed_at"`
}

type SetWithdrawDTO struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

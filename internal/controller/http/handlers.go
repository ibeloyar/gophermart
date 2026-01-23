package http

import (
	"net/http"

	"github.com/ibeloyar/gophermart/internal/model"
	"github.com/ibeloyar/gophermart/pgk/auth"
	"go.uber.org/zap"
)

type Service interface {
	Register(input model.RegisterDTO) (string, *model.APIError)
	Login(input model.LoginDTO) (string, *model.APIError)

	CreateOrder(userID int64, orderNumber string) *model.APIError
	GetOrders(userID int64) ([]model.Order, *model.APIError)
	GetBalance(userID int64) (*model.Balance, *model.APIError)
	SetWithdraw(userID int64, input model.SetWithdrawDTO) *model.APIError
	GetWithdraws(userID int64) ([]model.Withdraw, *model.APIError)
}

type Controller struct {
	service Service
	lg      *zap.SugaredLogger
}

func New(s Service, lg *zap.SugaredLogger) *Controller {
	return &Controller{
		lg:      lg,
		service: s,
	}
}

func (c *Controller) Register(w http.ResponseWriter, r *http.Request) {
	body, err := readBody[model.RegisterDTO](r)
	if err != nil {
		c.lg.Errorf("failed to parse request body: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	bearerToken, apiErr := c.service.Register(body)
	if apiErr != nil {
		http.Error(w, apiErr.Message, apiErr.Code)
		return
	}

	w.Header().Set("Authorization", bearerToken)
	w.WriteHeader(http.StatusOK)
}

func (c *Controller) Login(w http.ResponseWriter, r *http.Request) {
	body, err := readBody[model.LoginDTO](r)
	if err != nil {
		c.lg.Errorf("failed to parse request body: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	bearerToken, apiErr := c.service.Login(body)
	if apiErr != nil {
		http.Error(w, apiErr.Message, apiErr.Code)
		return
	}

	w.Header().Set("Authorization", bearerToken)
	w.WriteHeader(http.StatusOK)
}

func (c *Controller) CreateOrder(w http.ResponseWriter, r *http.Request) {
	orderNumber, err := readBody[string](r)
	if err != nil {
		c.lg.Errorf("failed to parse request body: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	apiErr := c.service.CreateOrder(auth.GetTokenInfo[model.TokenInfo](r).ID, orderNumber)
	if apiErr != nil {
		// Если order уже был добавлен текущим пользователем
		if apiErr.Code == http.StatusOK {
			w.WriteHeader(http.StatusOK)
			return
		}

		http.Error(w, apiErr.Message, apiErr.Code)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (c *Controller) GetOrders(w http.ResponseWriter, r *http.Request) {
	orders, apiErr := c.service.GetOrders(auth.GetTokenInfo[model.TokenInfo](r).ID)
	if apiErr != nil {
		http.Error(w, apiErr.Message, apiErr.Code)
		return
	}

	writeJSON(w, orders, http.StatusOK)
}

func (c *Controller) GetBalance(w http.ResponseWriter, r *http.Request) {
	balance, apiErr := c.service.GetBalance(auth.GetTokenInfo[model.TokenInfo](r).ID)
	if apiErr != nil {
		http.Error(w, apiErr.Message, apiErr.Code)
		return
	}

	writeJSON(w, balance, http.StatusOK)
}

func (c *Controller) SetWithdrawal(w http.ResponseWriter, r *http.Request) {
	body, err := readBody[model.SetWithdrawDTO](r)
	if err != nil {
		c.lg.Errorf("failed to parse request body: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	apiErr := c.service.SetWithdraw(auth.GetTokenInfo[model.TokenInfo](r).ID, body)
	if apiErr != nil {
		http.Error(w, apiErr.Message, apiErr.Code)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (c *Controller) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	withdrawals, apiErr := c.service.GetWithdraws(auth.GetTokenInfo[model.TokenInfo](r).ID)
	if apiErr != nil {
		http.Error(w, apiErr.Message, apiErr.Code)
		return
	}

	if len(withdrawals) == 0 {
		writeJSON(w, withdrawals, http.StatusNoContent)
		return
	}

	writeJSON(w, withdrawals, http.StatusOK)
}

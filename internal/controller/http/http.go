package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ibeloyar/gophermart/internal/model"
	"go.uber.org/zap"
)

type Service interface {
	Login(input model.LoginDTO) (string, *model.APIError)
	Register(input model.RegisterDTO) (string, *model.APIError)
	CreateOrder(token, orderNumber string) *model.APIError
	GetOrders(token string) ([]model.Order, *model.APIError)
	GetBalance(token string) (*model.Balance, *model.APIError)
	SetWithdraw(token string, input model.SetWithdrawDTO) *model.APIError
	GetWithdraws(token string) ([]model.Withdraw, *model.APIError)
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

	token, apiErr := c.service.Register(body)
	if apiErr != nil {
		http.Error(w, apiErr.Message, apiErr.Code)
		return
	}

	w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
	w.WriteHeader(http.StatusOK)
}

func (c *Controller) Login(w http.ResponseWriter, r *http.Request) {
	body, err := readBody[model.LoginDTO](r)
	if err != nil {
		c.lg.Errorf("failed to parse request body: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	token, apiErr := c.service.Login(body)
	if apiErr != nil {
		http.Error(w, apiErr.Message, apiErr.Code)
		return
	}

	w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
	w.WriteHeader(http.StatusOK)
}

func (c *Controller) CreateOrder(w http.ResponseWriter, r *http.Request) {
	orderNumber, err := readBody[string](r)
	if err != nil {
		c.lg.Errorf("failed to parse request body: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	apiErr := c.service.CreateOrder(r.Header.Get("Authorization"), orderNumber)
	if apiErr != nil {
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
	orders, apiErr := c.service.GetOrders(r.Header.Get("Authorization"))
	if apiErr != nil {
		http.Error(w, apiErr.Message, apiErr.Code)
		return
	}

	response, err := json.Marshal(orders)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func (c *Controller) GetBalance(w http.ResponseWriter, r *http.Request) {
	balance, apiErr := c.service.GetBalance(r.Header.Get("Authorization"))
	if apiErr != nil {
		http.Error(w, apiErr.Message, apiErr.Code)
		return
	}

	response, err := json.Marshal(balance)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func (c *Controller) SetWithdrawal(w http.ResponseWriter, r *http.Request) {
	body, err := readBody[model.SetWithdrawDTO](r)
	if err != nil {
		c.lg.Errorf("failed to parse request body: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	apiErr := c.service.SetWithdraw(r.Header.Get("Authorization"), body)
	if apiErr != nil {
		http.Error(w, apiErr.Message, apiErr.Code)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (c *Controller) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	withdrawals, apiErr := c.service.GetWithdraws(r.Header.Get("Authorization"))
	if apiErr != nil {
		http.Error(w, apiErr.Message, apiErr.Code)
		return
	}

	response, err := json.Marshal(withdrawals)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

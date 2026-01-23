package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handlers interface {
	Register(w http.ResponseWriter, r *http.Request)
	Login(w http.ResponseWriter, r *http.Request)
	CreateOrder(w http.ResponseWriter, r *http.Request)
	GetOrders(w http.ResponseWriter, r *http.Request)
	GetBalance(w http.ResponseWriter, r *http.Request)
	SetWithdrawal(w http.ResponseWriter, r *http.Request)
	GetWithdrawals(w http.ResponseWriter, r *http.Request)
}

func InitRoutes(r *chi.Mux, handlers Handlers) *chi.Mux {
	r.Post("/api/user/register", handlers.Register)
	r.Post("/api/user/login", handlers.Login)
	r.Post("/api/user/orders", handlers.CreateOrder)
	r.Get("/api/user/orders", handlers.GetOrders)
	r.Get("/api/user/balance", handlers.GetBalance)
	r.Post("/api/user/balance/withdraw", handlers.SetWithdrawal)
	r.Get("/api/user/withdrawals", handlers.GetWithdrawals)

	return r
}

package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ibeloyar/gophermart/internal/model"
	"github.com/ibeloyar/gophermart/pgk/auth"
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

func InitRoutes(r *chi.Mux, handlers Handlers, secret string) *chi.Mux {
	r.Post("/api/user/register", handlers.Register)
	r.Post("/api/user/login", handlers.Login)

	r.Group(func(r chi.Router) {
		authMiddleware := auth.AuthBearerMiddlewareInit[model.TokenInfo](secret)

		r.Use(authMiddleware)

		r.Post("/api/user/orders", handlers.CreateOrder)
		r.Get("/api/user/orders", handlers.GetOrders)
		r.Get("/api/user/balance", handlers.GetBalance)
		r.Post("/api/user/balance/withdraw", handlers.SetWithdrawal)
		r.Get("/api/user/withdrawals", handlers.GetWithdrawals)
	})

	return r
}

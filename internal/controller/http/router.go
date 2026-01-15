package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type MetricHandlers interface {
	Ping(w http.ResponseWriter, r *http.Request)
}

func InitRoutes(r *chi.Mux, metricHandlers MetricHandlers) *chi.Mux {

	r.Get("/ping", metricHandlers.Ping)

	return r
}

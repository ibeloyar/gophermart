package http

import (
	"net/http"

	"go.uber.org/zap"
)

type Service interface {
	Ping() error
}

type Controller struct {
	service Service
	lg      *zap.SugaredLogger
}

func New(s Service, lg *zap.SugaredLogger) *Controller {
	return &Controller{
		service: s,
		lg:      lg,
	}
}

func (c *Controller) Ping(w http.ResponseWriter, r *http.Request) {
	err := c.service.Ping()
	if err != nil {
		c.lg.Errorf("ping error: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}

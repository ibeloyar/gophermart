package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/ibeloyar/gophermart/internal/config"
	"github.com/ibeloyar/gophermart/internal/logger"
	"github.com/ibeloyar/gophermart/internal/repository/pg"
	"github.com/ibeloyar/gophermart/internal/service"

	httpController "github.com/ibeloyar/gophermart/internal/controller/http"
)

func Run(cfg config.Config) error {
	lg, err := logger.New()
	if err != nil {
		return err
	}
	defer lg.Sync()

	storage, err := pg.New(cfg.DatabaseURI)
	if err != nil {
		return err
	}

	router := chi.NewRouter()

	//router.Use(gzip.Middleware)
	router.Use(logger.LoggingMiddleware(lg))
	router.Use(middleware.Recoverer)

	s := service.New(storage)

	handlers := httpController.New(s, lg)
	router = httpController.InitRoutes(router, handlers)

	srv := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: router,
	}

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	lg.Infof("starting server on %s", cfg.RunAddress)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			lg.Fatalf("server ListenAndServe error: %v", err)
		}
	}()

	<-signalCtx.Done()
	lg.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown (server) error: %v", err)
	}

	if err := storage.Shutdown(); err != nil {
		return fmt.Errorf("shutdown (repo) error: %v", err)
	}

	lg.Info("server shutdown success")
	return nil
}

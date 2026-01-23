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
	"github.com/ibeloyar/gophermart/internal/repository/password"
	"github.com/ibeloyar/gophermart/internal/repository/pg"
	"github.com/ibeloyar/gophermart/internal/service"
	"github.com/ibeloyar/gophermart/pgk/logger"

	httpController "github.com/ibeloyar/gophermart/internal/controller/http"
)

func Run(cfg config.Config) error {
	lg, err := logger.New()
	if err != nil {
		return err
	}
	defer lg.Sync()

	storageRepo, err := pg.New(cfg.DatabaseURI, cfg.AccrualSystemAddress)
	if err != nil {
		return err
	}
	passwordRepo := password.New(cfg.PassCost)
	//tokenRepo := tokens.New(cfg.SecretKey, cfg.TokenLifetimeHours)

	mainService := service.New(storageRepo, passwordRepo, time.Duration(cfg.TokenLifetimeHours)*time.Hour, cfg.SecretKey)

	router := chi.NewRouter()
	//router.Use(gzip.Middleware)
	router.Use(logger.LoggingMiddleware(lg))
	router.Use(middleware.Recoverer)
	handlers := httpController.New(mainService, lg)
	router = httpController.InitRoutes(router, handlers, cfg.SecretKey)

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

	storageRepo.RunOrdersAccrualUpdater()

	<-signalCtx.Done()
	lg.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown (server) error: %v", err)
	}

	if err := storageRepo.Shutdown(); err != nil {
		return fmt.Errorf("shutdown (repo) error: %v", err)
	}

	lg.Info("server shutdown success")
	return nil
}

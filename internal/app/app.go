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
	"github.com/ibeloyar/gophermart/internal/repository/pg"
	"github.com/ibeloyar/gophermart/internal/service"
	"github.com/ibeloyar/gophermart/pgk/logger"
	"go.uber.org/zap"

	httpController "github.com/ibeloyar/gophermart/internal/controller/http"
)

func Run(cfg config.Config, zapLogger *zap.SugaredLogger) error {
	storageRepo, err := pg.New(cfg.DatabaseURI, cfg.AccrualSystemAddress, zapLogger)
	if err != nil {
		return fmt.Errorf("failed to create a DB connection: %w", err)
	}

	mainService := service.New(storageRepo, cfg.PassCost, cfg.TokenLifetime, cfg.SecretKey)

	router := chi.NewRouter()
	router.Use(logger.LoggingMiddleware(zapLogger))
	router.Use(middleware.Recoverer)
	handlers := httpController.New(mainService, zapLogger)

	srv := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: httpController.InitRoutes(router, handlers, cfg.SecretKey),
	}

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	zapLogger.Infof("starting server on %s", cfg.RunAddress)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zapLogger.Fatalf("server ListenAndServe error: %v", err)
		}
	}()

	<-signalCtx.Done()
	zapLogger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown (server) error: %v", err)
	}

	if err := storageRepo.Shutdown(); err != nil {
		return fmt.Errorf("shutdown (repo) error: %v", err)
	}

	zapLogger.Info("server shutdown success")
	return nil
}

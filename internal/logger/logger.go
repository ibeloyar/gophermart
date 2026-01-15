package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New() (*zap.SugaredLogger, error) {
	config := zap.NewDevelopmentConfig()

	config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return logger.Sugar(), nil
}

func LoggingMiddleware(logger *zap.SugaredLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &loggingResponseWriter{
				ResponseWriter: w,
				responseData: &responseData{
					status: http.StatusOK,
					size:   0,
				}}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)
			logger.Info("request->",
				" uri: ", r.RequestURI,
				" method: ", r.Method,
				" duration: ", duration,
				" status: ", rw.responseData.status,
				" size: ", rw.responseData.size,
			)
		})
	}
}

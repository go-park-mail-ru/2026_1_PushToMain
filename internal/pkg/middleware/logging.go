package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type contextKey string

const (
	RequestIDKey contextKey = "request_id"
	LoggerKey    contextKey = "logger"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func Logging(logger *zap.SugaredLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-Id")
			if requestID == "" {
				requestID = uuid.NewString()
			}

			w.Header().Set("X-Request-Id", requestID)

			requestLogger := logger.With(
				"request_id", requestID,
			)

			ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
			ctx = context.WithValue(ctx, LoggerKey, requestLogger)

			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			requestLogger.Infof("Request started: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

			start := time.Now()
			//next.ServeHTTP(rw, r)
			next.ServeHTTP(rw, r.WithContext(ctx))
			duration := time.Since(start)

			requestLogger.Infof("Request completed with status %d in %fms ",
				rw.status, duration.Seconds()*1000)
		})
	}
}

func GetLogger(ctx context.Context) *zap.SugaredLogger {
	if logger, ok := ctx.Value(LoggerKey).(*zap.SugaredLogger); ok {
		return logger
	}
	return zap.NewNop().Sugar()
}

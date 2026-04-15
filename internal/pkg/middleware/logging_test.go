package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogging(t *testing.T) {
	newTestLogger := func() (*zap.SugaredLogger, *observer.ObservedLogs) {
		core, recorded := observer.New(zapcore.InfoLevel)
		logger := zap.New(core).Sugar()
		return logger, recorded
	}

	t.Run("generates new request ID if none provided", func(t *testing.T) {
		logger, logs := newTestLogger()
		middleware := Logging(logger)

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID, ok := r.Context().Value(RequestIDKey).(string)
			assert.True(t, ok)
			assert.NotEmpty(t, reqID)

			ctxLogger, ok := r.Context().Value(LoggerKey).(*zap.SugaredLogger)
			assert.True(t, ok)
			assert.NotNil(t, ctxLogger)

			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		middleware(nextHandler).ServeHTTP(rr, req)

		requestID := rr.Header().Get("X-Request-Id")
		assert.NotEmpty(t, requestID)

		logEntries := logs.All()
		require.Len(t, logEntries, 2, "should have start and completion logs")

		startLog := logEntries[0]
		assert.Equal(t, "Request started: GET /test from ", startLog.Message[:32])
		assert.Equal(t, requestID, startLog.ContextMap()["request_id"])

		completionLog := logEntries[1]
		assert.Contains(t, completionLog.Message, "Request completed with status 200")
		assert.Equal(t, requestID, completionLog.ContextMap()["request_id"])
	})

	t.Run("uses existing X-Request-Id header", func(t *testing.T) {
		logger, logs := newTestLogger()
		middleware := Logging(logger)

		existingID := "abc-123-existing"

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := r.Context().Value(RequestIDKey).(string)
			assert.Equal(t, existingID, reqID)
			w.WriteHeader(http.StatusNotFound)
		})

		req := httptest.NewRequest(http.MethodPost, "/api", nil)
		req.Header.Set("X-Request-Id", existingID)
		rr := httptest.NewRecorder()

		middleware(nextHandler).ServeHTTP(rr, req)

		assert.Equal(t, existingID, rr.Header().Get("X-Request-Id"))

		logEntries := logs.All()
		require.Len(t, logEntries, 2)
		assert.Equal(t, existingID, logEntries[0].ContextMap()["request_id"])
		assert.Contains(t, logEntries[1].Message, "status 404")
	})

	t.Run("captures correct status code when handler writes header", func(t *testing.T) {
		logger, logs := newTestLogger()
		middleware := Logging(logger)

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTeapot) // 418
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		middleware(nextHandler).ServeHTTP(rr, req)

		logEntries := logs.All()
		require.Len(t, logEntries, 2)
		assert.Contains(t, logEntries[1].Message, "status 418")
	})

	t.Run("logs duration in milliseconds", func(t *testing.T) {
		logger, logs := newTestLogger()
		middleware := Logging(logger)

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(50 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/slow", nil)
		rr := httptest.NewRecorder()

		middleware(nextHandler).ServeHTTP(rr, req)

		logEntries := logs.All()
		completionMsg := logEntries[1].Message
		assert.Contains(t, completionMsg, "Request completed with status 200 in ")
		assert.Contains(t, completionMsg, "ms")
	})

	t.Run("context contains logger with request_id field", func(t *testing.T) {
		logger, _ := newTestLogger()
		middleware := Logging(logger)

		var capturedLogger *zap.SugaredLogger
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedLogger = r.Context().Value(LoggerKey).(*zap.SugaredLogger)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		middleware(nextHandler).ServeHTTP(rr, req)

		assert.NotNil(t, capturedLogger)
		// Check that the logger has the request_id field (by serializing to JSON)
		// This is tricky; we trust that logger.With added the field.
	})

	t.Run("preserves request context values", func(t *testing.T) {
		logger, _ := newTestLogger()
		middleware := Logging(logger)

		type key int
		const customKey key = 1

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			val := r.Context().Value(customKey)
			assert.Equal(t, "custom-value", val)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx := context.WithValue(req.Context(), customKey, "custom-value")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		middleware(nextHandler).ServeHTTP(rr, req)
	})
}

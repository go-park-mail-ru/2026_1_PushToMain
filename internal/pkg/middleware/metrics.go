package middleware

import (
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/pkg/metrics"
	"github.com/gorilla/mux"
)

type metricsResponseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *metricsResponseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func Metrics(m *metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &metricsResponseWriter{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()

			next.ServeHTTP(rw, r)

			path := "unknown"
			if route := mux.CurrentRoute(r); route != nil {
				if tmpl, err := route.GetPathTemplate(); err == nil {
					path = tmpl
				}
			}

			m.IncRequest(rw.status, r.Method, path)
			m.ObserveDuration(r.Method, path, time.Since(start))
		})
	}
}

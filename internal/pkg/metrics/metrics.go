package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	reg            *prometheus.Registry
	requestsTotal  *prometheus.CounterVec
	requeustTiming *prometheus.HistogramVec
	// CPU, RAM and Go stats are not there
}

func New(serviceName string, namespace string, serviceExtraCollectors ...prometheus.Collector) *Metrics {
	reg := prometheus.NewRegistry()
	constLabels := prometheus.Labels{"service": serviceName}

	requestTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Name:        "http_requests_total",
			Help:        "Total number of HTTP requests",
			ConstLabels: constLabels,
		},
		[]string{"code", "method", "path"},
	)

	requestTiming := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   namespace,
			Name:        "http_requests_timing_seconds",
			Help:        "Timing of HTTP requests in seconds",
			ConstLabels: constLabels,
		},
		[]string{"method", "path"},
	)

	toRegister := []prometheus.Collector{
		collectors.NewGoCollector(),                                       // Go stuff
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}), // CPU
		requestTiming,
		requestTotal,
	}
	toRegister = append(toRegister, serviceExtraCollectors...)
	reg.MustRegister(toRegister...)

	return &Metrics{
		reg:            reg,
		requestsTotal:  requestTotal,
		requeustTiming: requestTiming,
	}
}

func (m *Metrics) IncRequest(statusCode int, method string, path string) {
	m.requestsTotal.With(prometheus.Labels{
		"code":   strconv.Itoa(statusCode),
		"method": method,
		"path":   path,
	}).Inc()
}

func (m *Metrics) ObserveDuration(method string, path string, duration time.Duration) {
	m.requeustTiming.With(prometheus.Labels{
		"method": method,
		"path":   path,
	}).Observe(duration.Seconds())
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(
		m.reg,
		promhttp.HandlerOpts{Registry: m.reg},
	)
}

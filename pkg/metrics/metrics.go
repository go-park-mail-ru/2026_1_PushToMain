package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

type Metrics struct {
	reg    *prometheus.Registry
	Errors *prometheus.CounterVec
	Hits   *prometheus.CounterVec
	Timing *prometheus.HistogramVec
	// CPU, RAM and Go stats are not there
}

func New(serviceName string, serviceExtraCollectors ...prometheus.Collector) *Metrics {
	reg := prometheus.NewRegistry()

	errors := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:        "errors_total",
		ConstLabels: prometheus.Labels{"service": serviceName},
		Help:        "Total number of errors",
	}, []string{"path", "status"})

	hits := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:        "hits_total",
		ConstLabels: prometheus.Labels{"service": serviceName},
		Help:        "Total number of hits",
	}, []string{"path", "status"})

	timing := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        "timing_seconds",
		ConstLabels: prometheus.Labels{"service": serviceName},
		Buckets:     prometheus.ExponentialBucketsRange(0.005, 10, 10),
		Help:        "Total timings",
	}, []string{"path", "status"})

	toRegister := []prometheus.Collector{
		collectors.NewGoCollector(),                                       // Go stuff
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}), // CPU
		errors,
		hits,
		timing,
	}
	toRegister = append(toRegister, serviceExtraCollectors...)

	reg.MustRegister(toRegister...)

	return &Metrics{
		reg:    reg,
		Errors: errors,
		Hits:   hits,
		Timing: timing,
	}
}

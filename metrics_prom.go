package pipeline

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type PromMetrics struct {
	in      prometheus.Counter
	out     prometheus.Counter
	errors  prometheus.Counter
	latency prometheus.Histogram
}

func NewPromMetrics(namespace string) *PromMetrics {
	return &PromMetrics{
		in: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "in_total",
			Help:      "Total input",
		}),
		out: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "out_total",
			Help:      "Total output",
		}),
		errors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "error_total",
			Help:      "Total errors",
		}),
		latency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "latency_seconds",
			Help:      "Handler latency",
		}),
	}
}

func (m *PromMetrics) Register() {
	prometheus.MustRegister(m.in, m.out, m.errors, m.latency)
}

func (m *PromMetrics) IncIn() { m.in.Inc() }

func (m *PromMetrics) IncOut() { m.out.Inc() }

func (m *PromMetrics) IncError() { m.errors.Inc() }

func (m *PromMetrics) ObserveLatency(d time.Duration) {
	m.latency.Observe(d.Seconds())
}

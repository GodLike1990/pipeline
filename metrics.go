package pipeline

import "time"

type Metrics interface {
	IncIn()
	IncOut()
	IncError()
	ObserveLatency(time.Duration)
}

type noopMetrics struct{}

func (noopMetrics) IncIn() {}

func (noopMetrics) IncOut() {}

func (noopMetrics) IncError() {}

func (noopMetrics) ObserveLatency(time.Duration) {}

func defaultMetrics() Metrics { return noopMetrics{} }

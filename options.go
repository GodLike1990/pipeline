package pipeline

import (
	"context"
	"time"
)

type Handler[I any, O any] func(context.Context, I) (O, error)

type Option[I any, O any] func(*stage[I, O])

func WithHandler[I any, O any](h Handler[I, O]) Option[I, O] {
	return func(s *stage[I, O]) { s.handler = h }
}

func WithConcurrency[I any, O any](n int) Option[I, O] {
	return func(s *stage[I, O]) { s.concurrency = n }
}

func WithRateLimit[I any, O any](rps int) Option[I, O] {
	return func(s *stage[I, O]) { s.limiter = NewRateLimiter(rps) }
}

func WithRetry[I any, O any](times int) Option[I, O] {
	return func(s *stage[I, O]) { s.retry = NewFixedRetry(times) }
}

func WithCircuitBreaker[I any, O any](threshold int, cooldown time.Duration) Option[I, O] {
	return func(s *stage[I, O]) { s.breaker = NewSimpleBreaker(threshold, cooldown) }
}

func WithMetrics[I any, O any](m Metrics) Option[I, O] {
	return func(s *stage[I, O]) { s.metrics = m }
}

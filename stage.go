package pipeline

import (
	"context"
	"sync"
	"time"
)

type stage[I any, O any] struct {
	concurrency int
	limiter     RateLimiter
	retry       RetryPolicy
	breaker     CircuitBreaker
	handler     Handler[I, O]
	metrics     Metrics
}

func newStage[I any, O any](opts ...Option[I, O]) *stage[I, O] {
	s := &stage[I, O]{
		concurrency: 1,
		retry:       NoRetry{},
		breaker:     NoBreaker{},
		metrics:     defaultMetrics(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *stage[I, O]) run(ctx context.Context, in <-chan I) <-chan Result[O] {
	out := make(chan Result[O])
	var wg sync.WaitGroup
	wg.Add(s.concurrency)

	for i := 0; i < s.concurrency; i++ {
		go func() {
			defer wg.Done()
			for v := range in {
				s.metrics.IncIn()

				if err := s.breaker.Allow(); err != nil {
					s.metrics.IncError()
					out <- Result[O]{Err: err}
					continue
				}

				if s.limiter != nil {
					if err := s.limiter.Wait(ctx); err != nil {
						return
					}
				}

				start := time.Now()
				valAny, err := s.retry.Do(ctx, func() (any, error) {
					return s.handler(ctx, v)
				})
				s.metrics.ObserveLatency(time.Since(start))

				if err != nil {
					s.metrics.IncError()
				} else {
					s.metrics.IncOut()
				}

				s.breaker.Mark(err)
				if valAny != nil {
					out <- Result[O]{Val: valAny.(O), Err: err}
				} else {
					out <- Result[O]{Err: err}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

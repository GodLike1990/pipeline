package pipeline

import "context"

type RetryPolicy interface {
	Do(context.Context, func() (any, error)) (any, error)
}

type NoRetry struct{}

func (NoRetry) Do(ctx context.Context, fn func() (any, error)) (any, error) {
	return fn()
}

type FixedRetry struct {
	times int
}

func NewFixedRetry(times int) FixedRetry {
	return FixedRetry{times: times}
}

func (r FixedRetry) Do(ctx context.Context, fn func() (any, error)) (any, error) {
	var last error
	for i := 0; i <= r.times; i++ {
		v, err := fn()
		if err == nil {
			return v, nil
		}
		last = err
	}
	return nil, last
}

package pipeline

import (
	"context"
	"time"
)

type RateLimiter interface {
	Wait(context.Context) error
}

type TokenBucket struct {
	ch <-chan time.Time
}

func NewRateLimiter(rps int) *TokenBucket {
	return &TokenBucket{
		ch: time.Tick(time.Second / time.Duration(rps)),
	}
}

func (t *TokenBucket) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.ch:
		return nil
	}
}

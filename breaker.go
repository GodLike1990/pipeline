package pipeline

import (
	"sync"
	"time"
)

type CircuitBreaker interface {
	Allow() error
	Mark(error)
}

type NoBreaker struct{}

func (NoBreaker) Allow() error { return nil }

func (NoBreaker) Mark(error) {}

type SimpleBreaker struct {
	failures  int
	threshold int
	openUntil time.Time
	cooldown  time.Duration
	mu        sync.Mutex
}

func NewSimpleBreaker(threshold int, cooldown time.Duration) *SimpleBreaker {
	return &SimpleBreaker{threshold: threshold, cooldown: cooldown}
}

func (b *SimpleBreaker) Allow() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if time.Now().Before(b.openUntil) {
		return ErrCircuitOpen
	}
	return nil
}

func (b *SimpleBreaker) Mark(err error) {
	if err == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures++
	if b.failures >= b.threshold {
		b.openUntil = time.Now().Add(b.cooldown)
		b.failures = 0
	}
}

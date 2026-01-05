package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type RateLimiter interface {
	Wait(context.Context) error
	SetRate(rps float64)
}

type TokenBucket struct {
	mu    sync.RWMutex
	rate  float64       // 当前速率 (每秒请求数)
	tick  time.Duration // 间隔时间
	ch    chan time.Time
	close chan struct{}
	once  sync.Once
}

func NewRateLimiter(rps float64) *TokenBucket {
	if rps <= 0 {
		panic(fmt.Sprintf("rps is not valid: %f", rps))
	}
	tb := &TokenBucket{
		rate:  rps,
		tick:  time.Second / time.Duration(rps*1000) * 1000, // 转换为duration
		ch:    make(chan time.Time, 1),
		close: make(chan struct{}),
	}
	tb.start()
	return tb
}

// start 启动后台ticker，动态调整速率
func (t *TokenBucket) start() {
	go func() {
		ticker := time.NewTicker(t.tick)
		defer ticker.Stop()

		for {
			select {
			case <-t.close:
				return
			case ts := <-ticker.C:
				select {
				case t.ch <- ts:
				default: // channel已满，丢弃旧token
					select {
					case <-t.ch:
						t.ch <- ts
					default:
					}
				}
			}
		}
	}()
}

// SetRate 动态设置限速
func (t *TokenBucket) SetRate(rps float64) {
	if rps <= 0 {
		rps = 1 // 最小1 req/s
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.rate = rps
	t.tick = time.Second / time.Duration(rps*1000) * 1000

	// 重启ticker以应用新速率
	close(t.close)
	t.close = make(chan struct{})
	t.start()
}

func (t *TokenBucket) Wait(ctx context.Context) error {
	t.mu.RLock()
	ch := t.ch
	t.mu.RUnlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ch:
		return nil
	}
}

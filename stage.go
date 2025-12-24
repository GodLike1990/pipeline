package pipeline

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"
)

// stage 表示数据流水线中的一个处理阶段
// 它负责将输入类型 I 的数据转换为输出类型 O 的数据
// 每个阶段可以独立配置并发控制、限流、重试、熔断等策略
type stage[I any, O any] struct {
	concurrency int            // 并发worker数量，控制同时处理的数据量
	limiter     RateLimiter    // 速率限制器，防止处理过快导致系统过载
	retry       RetryPolicy    // 重试策略，处理临时性失败
	breaker     CircuitBreaker // 熔断器，防止级联故障
	handler     Handler[I, O]  // 业务处理函数，执行实际的数据转换逻辑
	metrics     Metrics        // 监控指标收集器，记录处理性能和状态
}

// newStage 创建一个新的数据处理阶段
// 使用Option模式进行配置，提供灵活的定制化能力
// 默认配置：单线程、无重试、无熔断器、空监控
func newStage[I any, O any](opts ...Option[I, O]) *stage[I, O] {
	// 创建stage实例并设置默认值
	s := &stage[I, O]{
		concurrency: 1,                // 默认并发度为1，顺序处理
		retry:       NoRetry{},        // 默认不重试，快速失败
		breaker:     NoBreaker{},      // 默认无熔断器，始终允许请求
		metrics:     defaultMetrics(), // 默认空监控，不收集指标
	}

	// 应用用户提供的配置选项
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// run 执行数据处理阶段的核心逻辑
// 启动指定数量的并发worker来处理输入数据
// 返回一个输出channel，消费者可以从中获取处理结果
func (s *stage[I, O]) run(ctx context.Context, in <-chan I) <-chan Result[O] {
	// 创建输出channel，用于发送处理结果
	out := make(chan Result[O])
	var wg sync.WaitGroup
	wg.Add(s.concurrency)

	// 启动并发worker池
	for i := 0; i < s.concurrency; i++ {
		go func() {
			defer wg.Done()

			for v := range in {
				// 1. 记录输入指标
				s.metrics.IncIn()

				// 2. 熔断器检查 - 如果熔断器打开，直接返回错误
				if err := s.breaker.Allow(); err != nil {
					s.metrics.IncError()
					out <- Result[O]{Err: err}
					continue
				}

				// 3. 速率限制
				if s.limiter != nil {
					if err := s.limiter.Wait(ctx); err != nil {
						if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
							// 上下文错误，记录日志并静默退出
							log.Printf("Worker exiting due to context cancellation: %v", err)
							return
						}
						// 其他类型的错误，记录并发送到下游
						log.Printf("Rate limiter unexpected error: %v", err)
						s.metrics.IncError()
						out <- Result[O]{Err: err}
						return
					}
				}

				// 4. 执行业务处理逻辑（带重试机制）
				start := time.Now()
				valAny, err := s.retry.Do(ctx, func() (any, error) {
					return s.handler(ctx, v) // 调用用户定义的处理函数
				})
				s.metrics.ObserveLatency(time.Since(start)) // 记录处理延迟

				// 5. 更新指标
				if err != nil {
					s.metrics.IncError()
				} else {
					s.metrics.IncOut()
				}

				// 6. 更新熔断器状态
				s.breaker.Mark(err)

				// 7. 发送结果到输出channel
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

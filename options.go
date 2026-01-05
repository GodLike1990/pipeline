package pipeline

import (
	"context"
	"time"
)

// Handler 定义业务处理函数的类型
// 接收上下文和输入数据，返回处理结果和可能的错误
// 这是用户需要实现的核心业务逻辑
type Handler[I any, O any] func(context.Context, I) (O, error)

// Option 定义配置选项的类型
// 使用函数式选项模式，提供灵活的配置能力
// 每个Option函数都负责配置stage的某个特定方面
type Option[I any, O any] func(*stage[I, O])

// WithHandler 设置业务处理函数
// 这是唯一必需的配置选项，定义了数据转换的核心逻辑
func WithHandler[I any, O any](h Handler[I, O]) Option[I, O] {
	return func(s *stage[I, O]) { s.handler = h }
}

// WithConcurrency 设置并发worker数量
// n: 并发数，控制同时处理的数据量
// 更高的并发数可以提高吞吐量，但也会增加资源消耗
func WithConcurrency[I any, O any](n int) Option[I, O] {
	return func(s *stage[I, O]) { s.concurrency = n }
}

// WithRateLimit 设置速率限制
// rps: 每秒请求数(Requests Per Second)，支持小数
// 用于防止系统过载，保护下游服务
// 返回的限速器可通过返回的 RateLimiter 接口动态调整速率
func WithRateLimit[I any, O any](rps float64) Option[I, O] {
	return func(s *stage[I, O]) { s.limiter = NewRateLimiter(rps) }
}

// WithRateLimiter 设置自定义速率限制器
// 支持动态调整速率和更复杂的限流策略
func WithRateLimiter[I any, O any](limiter RateLimiter) Option[I, O] {
	return func(s *stage[I, O]) { s.limiter = limiter }
}

// WithRetry 设置重试策略
// times: 重试次数，失败时的重试次数
// 用于处理临时性故障，提高系统的容错能力
func WithRetry[I any, O any](times int) Option[I, O] {
	return func(s *stage[I, O]) { s.retry = NewFixedRetry(times) }
}

// WithCircuitBreaker 设置熔断器
// threshold: 失败阈值，连续失败多少次后打开熔断器
// cooldown: 冷却时间，熔断器打开后的等待时间
// 用于防止级联故障，保护系统稳定性
func WithCircuitBreaker[I any, O any](threshold int, cooldown time.Duration) Option[I, O] {
	return func(s *stage[I, O]) { s.breaker = NewSimpleBreaker(threshold, cooldown) }
}

// WithMetrics 设置监控指标收集器
// m: Metrics实现，用于收集处理性能和状态指标
// 用于系统监控和性能分析
func WithMetrics[I any, O any](m Metrics) Option[I, O] {
	return func(s *stage[I, O]) { s.metrics = m }
}

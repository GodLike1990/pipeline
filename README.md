
# Pipeline

Pipeline is a production-ready concurrent pipeline library for Go.
一个专为 Go 语言设计的高性能并发数据处理流水线库。

## Architecture Overview

```
Pipeline (用户接口层)
    ↓
Stage (核心处理层) ← Options (配置层)
    ↓
Components (组件层)
├── RateLimiter (速率限制)
├── RetryPolicy (重试策略)  
├── CircuitBreaker (熔断器)
└── Metrics (监控指标)
```

### Key Components

- **`pipeline.go`**: 主要接口和Result类型定义
- **`stage.go`**: 核心处理逻辑，worker池管理
- **`options.go`**: 配置选项和函数式选项模式
- **`builder.go`**: 多阶段流水线构建器
- **`batch.go`**: 批量数据转换工具
- **`retry.go`**: 重试策略实现
- **`breaker.go`**: 熔断器实现
- **`limiter.go`**: 速率限制器实现
- **`metrics.go`**: 监控指标接口
- **`metrics_prom.go`**: Prometheus指标实现

## Features | 特性

- Controlled concurrency (worker pool) - 受控并发（工作池模式）
- Rate limiting - 速率限制
- Retry & circuit breaker - 重试与熔断器
- Batch input - 批量输入
- Multi-stage pipeline builder - 多阶段流水线构建器
- Optional Prometheus metrics - 可选的Prometheus监控指标
- Stable API (Option pattern) - 稳定的API（选项模式）
- Built-in benchmarks - 内置基准测试

## Install | 安装

```bash
go get github.com/GodLike1990/pipeline
```

Go 1.20+

## Quick Start | 快速开始

```go
// 创建一个将int转换为string的流水线
p := pipeline.New(
    pipeline.WithConcurrency(4),
    pipeline.WithHandler(func(ctx context.Context, v int) (string, error) {
        return strconv.Itoa(v), nil
    }),
)

// 运行流水线，批量处理数据
out := p.Run(context.Background(), pipeline.Batch([]int{1,2,3}))
for res := range out {
    if res.Err != nil {
        log.Println(res.Err)
        continue
    }
    fmt.Println(res.Val)
}
```

## Builder (Multi-stage) | 构建器（多阶段）

```go
// 构建多阶段流水线：int -> int -> string
pipe := pipeline.
    NewBuilder[int]().
    Then(
        pipeline.StageFunc[int,int](func(ctx context.Context, v int) (int,error) {
            return v*2, nil
        }),
        pipeline.WithConcurrency(8),
    ).
    Then(
        pipeline.StageFunc[int,string](func(ctx context.Context, v int) (string,error) {
            return strconv.Itoa(v), nil
        }),
    ).
    Build()

out := pipe.Run(ctx, pipeline.Batch([]int{1,2,3}))
```

## Metrics | 监控指标

Prometheus metrics are optional. - Prometheus监控指标是可选的

```go
// 创建Prometheus监控指标
m := pipeline.NewPromMetrics("demo_pipeline")
m.Register()


pipeline.New(
    pipeline.WithMetrics(m),
    pipeline.WithHandler(fn),
)
```

## Benchmark | 基准测试

```bash
go test ./benchmark -bench=. -benchmem
```

## License | 许可证

MIT

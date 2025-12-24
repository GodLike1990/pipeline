
# Pipeline

Pipeline is a production-ready concurrent pipeline library for Go.

## Features

- Controlled concurrency (worker pool)
- Rate limiting
- Retry & circuit breaker
- Batch input
- Multi-stage pipeline builder
- Optional Prometheus metrics
- Stable API (Option pattern)
- Built-in benchmarks

## Install

```bash
go get github.com/GodLike1990/pipeline
```

Go 1.20+

## Quick Start

```go
p := pipeline.New[int, string](
    pipeline.WithConcurrency(4),
    pipeline.WithHandler(func(ctx context.Context, v int) (string, error) {
        return strconv.Itoa(v), nil
    }),
)

out := p.Run(context.Background(), pipeline.Batch([]int{1,2,3}))
for res := range out {
    if res.Err != nil {
        log.Println(res.Err)
        continue
    }
    fmt.Println(res.Val)
}
```

## Builder (Multi-stage)

```go
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

## Metrics

Prometheus metrics are optional.

```go
m := pipeline.NewPromMetrics("demo_pipeline")
m.Register()

pipeline.New[int,int](
    pipeline.WithMetrics(m),
    pipeline.WithHandler(fn),
)
```

## Benchmark

```bash
go test ./benchmark -bench=. -benchmem
```

## License

MIT

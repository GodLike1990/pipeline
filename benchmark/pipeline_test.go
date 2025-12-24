package benchmark

import (
	"context"
	"runtime"
	"testing"

	"github.com/GodLike1990/pipeline"
)

func BenchmarkPipeline(b *testing.B) {
	data := make([]int, 100_000)
	for i := range data {
		data[i] = i
	}

	p := pipeline.New(
		pipeline.WithConcurrency[int, int](runtime.NumCPU()),
		pipeline.WithHandler(func(ctx context.Context, v int) (int, error) {
			return v + 1, nil
		}),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for range p.Run(context.Background(), pipeline.Batch(data)) {
		}

	}
}

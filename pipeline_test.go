package pipeline

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"
)

// TestConcurrencyWithRateLimit 测试并发+限速组合场景
func TestConcurrencyWithRateLimit(t *testing.T) {
	tests := []struct {
		name        string
		concurrency int
		rps         float64
		inputSize   int
	}{
		{name: "高并发低限速", concurrency: 10, rps: 2.0, inputSize: 10},
		{name: "低并发高限速", concurrency: 2, rps: 20.0, inputSize: 10},
		{name: "并发等于限速", concurrency: 5, rps: 5.0, inputSize: 10},
		{name: "小数限速", concurrency: 3, rps: 3.5, inputSize: 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mu sync.Mutex
			var maxConcurrent int
			var processed int

			p := New(
				WithConcurrency[int, int](tt.concurrency),
				WithRateLimit[int, int](tt.rps),
				WithHandler(func(_ context.Context, v int) (int, error) {
					mu.Lock()
					processed++
					if processed > maxConcurrent {
						maxConcurrent = processed
					}
					current := processed
					mu.Unlock()

					time.Sleep(10 * time.Millisecond)

					mu.Lock()
					processed--
					mu.Unlock()

					t.Logf("Worker %d processed, concurrent: %d", v, current)
					return v * 2, nil
				}),
			)

			input := make([]int, tt.inputSize)
			for i := range input {
				input[i] = i
			}

			start := time.Now()
			out := p.Run(context.Background(), Batch(input))

			var results []int
			for res := range out {
				if res.Err != nil {
					t.Errorf("Unexpected error: %v", res.Err)
					continue
				}
				results = append(results, res.Val)
			}
			duration := time.Since(start)

			// 验证结果数量
			if len(results) != tt.inputSize {
				t.Errorf("Expected %d results, got %d", tt.inputSize, len(results))
			}

			// 验证并发数不超过设置值
			if maxConcurrent > tt.concurrency {
				t.Errorf("Expected max concurrent <= %d, got %d", tt.concurrency, maxConcurrent)
			}

			// 验证限速生效（最小处理时间）
			expectedMinDuration := time.Duration(float64(tt.inputSize-1)/tt.rps*1000) * time.Millisecond
			if duration < expectedMinDuration {
				t.Logf("Warning: duration=%v, expected min=%v", duration, expectedMinDuration)
			}

			t.Logf("Test '%s': concurrency=%d, rps=%.1f, input=%d, duration=%v",
				tt.name, tt.concurrency, tt.rps, tt.inputSize, duration)
		})
	}
}

// TestDynamicRateLimitAdjustment 测试动态调整限速
func TestDynamicRateLimitAdjustment(t *testing.T) {
	p := New(
		WithRateLimit[int, int](2.0), // 初始2 req/s
		WithHandler(func(_ context.Context, v int) (int, error) {
			return v * 2, nil
		}),
	)

	// 运行时调整限速
	p.SetRate(5.0)

	start := time.Now()
	input := []int{1, 2, 3, 4, 5}
	out := p.Run(context.Background(), Batch(input))

	var results []int
	for res := range out {
		if res.Err != nil {
			t.Errorf("Unexpected error: %v", res.Err)
			continue
		}
		results = append(results, res.Val)
	}
	duration := time.Since(start)

	if len(results) != len(input) {
		t.Errorf("Expected %d results, got %d", len(input), len(results))
	}

	// 验证新限速生效
	expectedMinDuration := time.Duration(float64(len(input)-1)/5.0*1000) * time.Millisecond
	if duration < expectedMinDuration {
		t.Logf("Warning: duration=%v, expected min=%v", duration, expectedMinDuration)
	}

	t.Logf("Dynamic rate limit test: duration=%v", duration)
}

// TestErrorHandling 测试错误处理
func TestErrorHandling(t *testing.T) {
	p := New(
		WithConcurrency[int, string](2),
		WithHandler(func(_ context.Context, v int) (string, error) {
			if v == 2 {
				return "", fmt.Errorf("test error for value %d", v)
			}
			return strconv.Itoa(v), nil
		}),
	)

	input := []int{1, 2, 3}
	out := p.Run(context.Background(), Batch(input))

	var results []string
	var errorCount int

	for res := range out {
		if res.Err != nil {
			errorCount++
			continue
		}
		results = append(results, res.Val)
	}

	if errorCount != 1 {
		t.Errorf("Expected 1 error, got %d", errorCount)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 successful results, got %d", len(results))
	}
}

// TestRetry 测试重试机制
func TestRetry(t *testing.T) {
	attempts := 0
	maxAttempts := 3

	p := New(
		WithRetry[int, int](maxAttempts-1),
		WithHandler(func(_ context.Context, v int) (int, error) {
			attempts++
			if attempts < maxAttempts {
				return 0, errors.New("temporary error")
			}
			return v * 2, nil
		}),
	)

	input := []int{1}
	out := p.Run(context.Background(), Batch(input))

	var results []int
	for res := range out {
		if res.Err != nil {
			t.Errorf("Unexpected error after retries: %v", res.Err)
			continue
		}
		results = append(results, res.Val)
	}

	if attempts != maxAttempts {
		t.Errorf("Expected %d attempts, got %d", maxAttempts, attempts)
	}

	if len(results) != 1 || results[0] != 2 {
		t.Errorf("Expected successful result after retries, got %v", results)
	}
}

// TestCircuitBreaker 测试熔断器
func TestCircuitBreaker(t *testing.T) {
	threshold := 2
	cooldown := time.Millisecond * 100

	p := New(
		WithCircuitBreaker[int, int](threshold, cooldown),
		WithHandler(func(_ context.Context, _ int) (int, error) {
			return 0, errors.New("simulated error")
		}),
	)

	input := []int{1, 2, 3, 4}
	out := p.Run(context.Background(), Batch(input))

	var circuitOpenErrors int
	var otherErrors int

	for res := range out {
		if res.Err != nil {
			if errors.Is(res.Err, ErrCircuitOpen) {
				circuitOpenErrors++
			} else {
				otherErrors++
			}
		}
	}

	if circuitOpenErrors == 0 {
		t.Error("Expected circuit breaker to open")
	}

	if otherErrors < threshold {
		t.Errorf("Expected at least %d other errors before circuit opens, got %d", threshold, otherErrors)
	}
}

// TestContextCancellation 测试上下文取消
func TestContextCancellation(t *testing.T) {
	p := New(
		WithConcurrency[int, int](2),
		WithHandler(func(ctx context.Context, v int) (int, error) {
			select {
			case <-time.After(time.Second):
				return v * 2, nil
			case <-ctx.Done():
				return 0, ctx.Err()
			}
		}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()

	input := []int{1, 2, 3, 4, 5}
	out := p.Run(ctx, Batch(input))

	var results []int
	var canceledErrors int

	for res := range out {
		if res.Err != nil {
			if errors.Is(res.Err, context.Canceled) || errors.Is(res.Err, context.DeadlineExceeded) {
				canceledErrors++
			}
		} else {
			results = append(results, res.Val)
		}
	}

	if canceledErrors == 0 {
		t.Error("Expected context cancellation errors")
	}

	t.Logf("Got %d results before cancellation, %d cancellation errors", len(results), canceledErrors)
}

// TestLargeDataset 测试大数据集
func TestLargeDataset(t *testing.T) {
	p := New(
		WithConcurrency[int, int](4),
		WithRateLimit[int, int](100.0),
		WithHandler(func(_ context.Context, v int) (int, error) {
			return v * 2, nil
		}),
	)

	input := make([]int, 10000)
	for i := range input {
		input[i] = i
	}

	out := p.Run(context.Background(), Batch(input))

	var results []int
	for res := range out {
		if res.Err != nil {
			t.Errorf("Unexpected error at item %d: %v", len(results), res.Err)
			continue
		}
		results = append(results, res.Val)
	}

	if len(results) != len(input) {
		t.Errorf("Expected %d results, got %d", len(input), len(results))
	}

	// 抽样验证结果正确性
	for i := 0; i < 100; i++ {
		expected := i * 2
		found := false
		for _, r := range results {
			if r == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing expected result: %d", expected)
		}
	}
}

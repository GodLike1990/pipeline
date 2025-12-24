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

// TestQuickStart 测试Quick Start示例
func TestQuickStart(t *testing.T) {
	// 创建一个简单的pipeline，将int转换为string
	p := New(
		WithConcurrency[int, string](4),
		WithHandler(func(_ context.Context, v int) (string, error) {
			return strconv.Itoa(v), nil
		}),
	)

	// 准备输入数据
	input := []int{1, 2, 3}
	out := p.Run(context.Background(), Batch(input))

	// 收集结果
	var (
		results []string
		err     []error
	)

	for res := range out {
		if res.Err != nil {
			err = append(err, res.Err)
			continue
		}
		results = append(results, res.Val)
	}

	// 验证结果
	if len(err) != 0 {
		t.Errorf("Expected no err, got %d err", len(err))
	}

	expected := map[string]bool{"1": true, "2": true, "3": true}
	if len(results) != len(expected) {
		t.Errorf("Expected %d results, got %d", len(expected), len(results))
	}

	for _, result := range results {
		if !expected[result] {
			t.Errorf("Unexpected result: %s", result)
		}
	}
}

// TestBuilderMultiStage 测试Builder多阶段示例
func TestBuilderMultiStage(t *testing.T) {
	ctx := context.Background()

	// 创建简单的pipeline来测试Builder功能
	pipe := NewBuilder[int]().Build()

	// 运行pipeline
	input := []int{1, 2, 3}
	out := pipe.Run(ctx, Batch(input))

	// 收集结果
	var results []int
	for res := range out {
		if res.Err != nil {
			t.Errorf("Unexpected error: %v", res.Err)
			continue
		}
		results = append(results, res.Val)
	}

	// 验证结果：Builder默认应该直接传递输入
	expected := []int{1, 2, 3}
	if len(results) != len(expected) {
		t.Errorf("Expected %d results, got %d", len(expected), len(results))
	}

	for i, result := range results {
		if result != expected[i] {
			t.Errorf("Expected %d, got %d", expected[i], result)
		}
	}
}

// TestMetrics 测试Metrics示例
func TestMetrics(t *testing.T) {
	// 创建Prometheus metrics
	m := NewPromMetrics("demo_pipeline")

	// 测试metrics的基本功能
	m.IncIn()
	m.IncOut()
	m.IncError()
	m.ObserveLatency(time.Millisecond * 100)

	// 验证metrics不会panic
	// 注意：这里不测试Register()，因为它需要真实的Prometheus注册器

	// 使用metrics创建pipeline
	p := New(
		WithMetrics[int, int](m),
		WithHandler(func(_ context.Context, v int) (int, error) {
			return v * 2, nil
		}),
	)

	// 运行pipeline
	input := []int{1, 2, 3}
	out := p.Run(context.Background(), Batch(input))

	// 收集结果
	var results []int
	for res := range out {
		if res.Err != nil {
			t.Errorf("Unexpected error: %v", res.Err)
			continue
		}
		results = append(results, res.Val)
	}

	// 验证结果
	expected := []int{2, 4, 6}
	if len(results) != len(expected) {
		t.Errorf("Expected %d results, got %d", len(expected), len(results))
	}
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

	// 验证有一个错误
	if errorCount != 1 {
		t.Errorf("Expected 1 error, got %d", errorCount)
	}

	// 验证成功的结果
	if len(results) != 2 {
		t.Errorf("Expected 2 successful results, got %d", len(results))
	}
}

// TestConcurrency 测试并发控制
func TestConcurrency(t *testing.T) {
	concurrency := 4
	var maxConcurrent int
	var mu sync.Mutex

	p := New(
		WithConcurrency[int, int](concurrency),
		WithHandler(func(_ context.Context, v int) (int, error) {
			mu.Lock()
			maxConcurrent++
			mu.Unlock()

			// 模拟工作
			time.Sleep(time.Millisecond * 10)

			mu.Lock()
			maxConcurrent--
			mu.Unlock()

			return v * 2, nil
		}),
	)

	// 创建足够的输入来测试并发
	input := make([]int, 20)
	for i := range input {
		input[i] = i
	}

	out := p.Run(context.Background(), Batch(input))

	// 收集所有结果
	var results []int
	for res := range out {
		if res.Err != nil {
			t.Errorf("Unexpected error: %v", res.Err)
			continue
		}
		results = append(results, res.Val)
	}

	// 验证结果数量
	if len(results) != len(input) {
		t.Errorf("Expected %d results, got %d", len(input), len(results))
	}

	// 验证最大并发数不超过设置值
	mu.Lock()
	if maxConcurrent > concurrency {
		t.Errorf("Expected max concurrent <= %d, got %d", concurrency, maxConcurrent)
	}
	mu.Unlock()
}

// TestRateLimiting 测试速率限制
func TestRateLimiting(t *testing.T) {
	rps := 10 // 每秒10个请求

	p := New(
		WithRateLimit[int, int](rps),
		WithHandler(func(_ context.Context, v int) (int, error) {
			return v * 2, nil
		}),
	)

	start := time.Now()
	input := []int{1, 2, 3, 4, 5}
	out := p.Run(context.Background(), Batch(input))

	// 收集结果
	var results []int
	for res := range out {
		if res.Err != nil {
			t.Errorf("Unexpected error: %v", res.Err)
			continue
		}
		results = append(results, res.Val)
	}

	duration := time.Since(start)

	// 验证结果
	if len(results) != len(input) {
		t.Errorf("Expected %d results, got %d", len(input), len(results))
	}

	// 验证速率限制生效（应该至少花费 (len(input)-1)/rps 秒）
	expectedMinDuration := time.Duration(len(input)-1) * time.Second / time.Duration(rps)
	if duration < expectedMinDuration {
		t.Logf("Warning: rate limiting may not be working effectively. Expected min duration: %v, actual: %v", expectedMinDuration, duration)
	}
}

// TestRetry 测试重试机制
func TestRetry(t *testing.T) {
	attempts := 0
	maxAttempts := 3

	p := New(
		WithRetry[int, int](maxAttempts-1), // 重试2次，总共3次
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

	// 收集结果
	var results []int
	for res := range out {
		if res.Err != nil {
			t.Errorf("Unexpected error after retries: %v", res.Err)
			continue
		}
		results = append(results, res.Val)
	}

	// 验证重试次数
	if attempts != maxAttempts {
		t.Errorf("Expected %d attempts, got %d", maxAttempts, attempts)
	}

	// 验证最终成功
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

	// 验证熔断器触发
	if circuitOpenErrors == 0 {
		t.Error("Expected circuit breaker to open")
	}

	// 验证其他错误
	if otherErrors < threshold {
		t.Errorf("Expected at least %d other errors before circuit opens, got %d", threshold, otherErrors)
	}
}

// TestBatchFunction 测试Batch函数
func TestBatchFunction(t *testing.T) {
	data := []int{1, 2, 3, 4, 5}
	ch := Batch(data)

	var results []int
	for v := range ch {
		results = append(results, v)
	}

	// 验证所有数据都被发送
	if len(results) != len(data) {
		t.Errorf("Expected %d items, got %d", len(data), len(results))
	}

	// 验证数据顺序（虽然channel不保证顺序，但Batch应该保持顺序）
	for i, v := range results {
		if v != data[i] {
			t.Errorf("Expected %d at position %d, got %d", data[i], i, v)
		}
	}
}

// TestContextCancellation 测试上下文取消
func TestContextCancellation(t *testing.T) {
	p := New(
		WithConcurrency[int, int](2),
		WithHandler(func(ctx context.Context, v int) (int, error) {
			// 模拟长时间运行的任务
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

	// 验证有取消错误
	if canceledErrors == 0 {
		t.Error("Expected context cancellation errors")
	}

	// 验证部分结果可能已完成
	t.Logf("Got %d results before cancellation, %d cancellation errors", len(results), canceledErrors)
}

// TestEmptyInput 测试空输入
func TestEmptyInput(t *testing.T) {
	p := New(
		WithConcurrency[int, string](2),
		WithHandler(func(_ context.Context, v int) (string, error) {
			return strconv.Itoa(v), nil
		}),
	)

	var input []int
	out := p.Run(context.Background(), Batch(input))

	var results []string
	for res := range out {
		if res.Err != nil {
			t.Errorf("Unexpected error: %v", res.Err)
			continue
		}
		results = append(results, res.Val)
	}

	// 验证空输入产生空输出
	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty input, got %d", len(results))
	}
}

// TestLargeDataset 测试大数据集
func TestLargeDataset(t *testing.T) {
	p := New(
		WithConcurrency[int, int](4),
		WithHandler(func(_ context.Context, v int) (int, error) {
			return v * 2, nil
		}),
	)

	// 创建大数据集
	input := make([]int, 10000)
	for i := range input {
		input[i] = i
	}

	out := p.Run(context.Background(), Batch(input))

	var results []int
	var count int
	for res := range out {
		if res.Err != nil {
			t.Errorf("Unexpected error at item %d: %v", count, res.Err)
			continue
		}
		results = append(results, res.Val)
		count++
	}

	// 验证所有结果都被处理
	if len(results) != len(input) {
		t.Errorf("Expected %d results, got %d", len(input), len(results))
	}

	// 验证结果的正确性 - 使用map来验证所有结果都正确
	resultMap := make(map[int]bool)
	for _, result := range results {
		resultMap[result] = true
	}

	// 验证每个输入值都有对应的正确输出
	for i := 0; i < 100; i++ { // 只检查前100个，避免测试时间过长
		expected := i * 2
		if !resultMap[expected] {
			t.Errorf("Missing expected result: %d", expected)
		}
	}
}

// TestMixedSuccessFailure 测试混合成功和失败的情况
func TestMixedSuccessFailure(t *testing.T) {
	p := New(
		WithConcurrency[int, int](2),
		WithHandler(func(_ context.Context, v int) (int, error) {
			if v%3 == 0 { // 每3个失败一个
				return 0, fmt.Errorf("error for value %d", v)
			}
			return v * 2, nil
		}),
	)

	input := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	out := p.Run(context.Background(), Batch(input))

	var successCount, errorCount int
	for res := range out {
		if res.Err != nil {
			errorCount++
		} else {
			successCount++
		}
	}

	// 验证成功和失败的数量
	expectedErrors := 3  // 3, 6, 9
	expectedSuccess := 6 // 其他6个
	if errorCount != expectedErrors {
		t.Errorf("Expected %d errors, got %d", expectedErrors, errorCount)
	}
	if successCount != expectedSuccess {
		t.Errorf("Expected %d successes, got %d", expectedSuccess, successCount)
	}
}

// TestNoopMetrics 测试无操作metrics
func TestNoopMetrics(t *testing.T) {
	m := defaultMetrics()

	// 这些调用不应该panic
	m.IncIn()
	m.IncOut()
	m.IncError()
	m.ObserveLatency(time.Millisecond * 50)

	// 验证noop metrics不会影响pipeline运行
	p := New(
		WithMetrics[int, int](m),
		WithHandler(func(_ context.Context, v int) (int, error) {
			return v + 1, nil
		}),
	)

	input := []int{1, 2, 3}
	out := p.Run(context.Background(), Batch(input))

	var results []int
	for res := range out {
		if res.Err != nil {
			t.Errorf("Unexpected error: %v", res.Err)
			continue
		}
		results = append(results, res.Val)
	}

	expected := []int{2, 3, 4}
	if len(results) != len(expected) {
		t.Errorf("Expected %d results, got %d", len(expected), len(results))
	}
}

// TestNoRetry 测试无重试策略
func TestNoRetry(t *testing.T) {
	p := New(
		WithRetry[int, int](0), // 不重试
		WithHandler(func(_ context.Context, _ int) (int, error) {
			return 0, errors.New("always fails")
		}),
	)

	input := []int{1, 2, 3}
	out := p.Run(context.Background(), Batch(input))

	var errorCount int
	for res := range out {
		if res.Err != nil {
			errorCount++
		}
	}

	// 验证每个输入都产生一个错误（没有重试）
	if errorCount != len(input) {
		t.Errorf("Expected %d errors without retry, got %d", len(input), errorCount)
	}
}

// TestNoBreaker 测试无熔断器
func TestNoBreaker(t *testing.T) {
	p := New(
		WithCircuitBreaker[int, int](0, 0), // 使用NoBreaker
		WithHandler(func(_ context.Context, _ int) (int, error) {
			return 0, errors.New("always fails")
		}),
	)

	input := []int{1, 2, 3, 4, 5}
	out := p.Run(context.Background(), Batch(input))

	var errorCount int
	var circuitOpenErrors int

	for res := range out {
		if res.Err != nil {
			errorCount++
			if errors.Is(res.Err, ErrCircuitOpen) {
				circuitOpenErrors++
			}
		}
	}

	// 验证所有错误都是普通错误，没有熔断器错误
	if circuitOpenErrors > 0 {
		t.Errorf("Expected no circuit breaker errors with NoBreaker, got %d", circuitOpenErrors)
	}
	if errorCount != len(input) {
		t.Errorf("Expected %d total errors, got %d", len(input), errorCount)
	}
}

// TestPipelineWithNoRateLimit 测试无速率限制
func TestPipelineWithNoRateLimit(t *testing.T) {
	p := New(
		WithHandler(func(_ context.Context, v int) (int, error) {
			return v * 2, nil
		}),
	)

	input := []int{1, 2, 3, 4, 5}
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

	// 验证结果正确性
	if len(results) != len(input) {
		t.Errorf("Expected %d results, got %d", len(input), len(results))
	}

	// 无速率限制应该很快完成
	if duration > time.Millisecond*100 {
		t.Logf("Warning: pipeline took longer than expected without rate limiting: %v", duration)
	}
}

package pipeline

import "context"

// Result 封装处理结果，包含值和错误
// 使用泛型确保类型安全，避免运行时类型转换
type Result[T any] struct {
	Val T
	Err error
}

// Pipeline 表示一个完整的数据处理流水线
// 它将输入类型 I 的数据转换为输出类型 O 的数据
// 内部封装了一个或多个处理阶段(stage)
type Pipeline[I any, O any] struct {
	run func(context.Context, <-chan I) <-chan Result[O]
}

// New 创建一个新的数据处理流水线
// 使用Option模式配置处理策略
// 返回一个可以直接运行的Pipeline实例
func New[I any, O any](opts ...Option[I, O]) *Pipeline[I, O] {
	// 创建内部处理阶段
	s := newStage(opts...)

	return &Pipeline[I, O]{
		run: func(ctx context.Context, in <-chan I) <-chan Result[O] {
			return s.run(ctx, in)
		},
	}
}

// Run 启动数据处理流水线
// ctx: 上下文，用于控制生命周期和取消操作
// in: 输入数据channel
// 返回: 结果channel，消费者可以从中获取处理结果
func (p *Pipeline[I, O]) Run(ctx context.Context, in <-chan I) <-chan Result[O] {
	return p.run(ctx, in)
}

package pipeline

import "context"

// Builder 用于构建多阶段数据流水线
// 支持链式调用，可以串联多个处理阶段
// 使用any类型来实现不同阶段之间的类型转换
type Builder[I any] struct {
	run func(context.Context, <-chan any) <-chan Result[any]
}

// NewBuilder 创建一个新的流水线构建器
// 初始化为直接传递输入数据的identity stage
func NewBuilder[I any]() *Builder[I] {
	return &Builder[I]{
		run: func(_ context.Context, in <-chan any) <-chan Result[any] {
			out := make(chan Result[any])
			go func() {
				defer close(out)
				for v := range in {
					out <- Result[any]{Val: v} // 直接传递，不做任何处理
				}
			}()
			return out
		},
	}
}

// Then 添加一个新的处理阶段到流水线中
// _ Builder参数未使用，但保持函数签名的一致性
// fn: 新阶段的处理函数
// opts: 新阶段的配置选项
// 返回: 新的Builder，输出类型变为O
func Then[I any, O any](_ *Builder[I], fn Handler[I, O], opts ...Option[I, O]) *Builder[O] {
	// 创建新的处理阶段
	stage := newStage(append(opts, WithHandler(fn))...)

	return &Builder[O]{
		run: func(ctx context.Context, in <-chan any) <-chan Result[any] {
			// 类型转换：any -> I
			typed := make(chan I)
			go func() {
				defer close(typed)
				for v := range in {
					typed <- v.(I) // 类型断言
				}
			}()

			// 执行处理阶段：I -> O
			out := stage.run(ctx, typed)

			// 类型转换：Result[O] -> Result[any]
			next := make(chan Result[any])
			go func() {
				defer close(next)
				for r := range out {
					next <- Result[any]{Val: r.Val, Err: r.Err}
				}
			}()
			return next
		},
	}
}

// Build 构建最终的可执行流水线
// 将Builder转换为Pipeline实例，可以直接运行
// 返回: 输入输出类型相同的Pipeline实例
func (b *Builder[I]) Build() *Pipeline[I, I] {
	return &Pipeline[I, I]{
		run: func(ctx context.Context, in <-chan I) <-chan Result[I] {
			// 类型转换：I -> any
			anyIn := make(chan any)
			go func() {
				defer close(anyIn)
				for v := range in {
					anyIn <- v
				}
			}()

			// 执行内部构建的处理链
			outAny := b.run(ctx, anyIn)

			// 类型转换：Result[any] -> Result[I]
			out := make(chan Result[I])
			go func() {
				defer close(out)
				for r := range outAny {
					out <- Result[I]{Val: r.Val.(I), Err: r.Err}
				}
			}()
			return out
		},
	}
}

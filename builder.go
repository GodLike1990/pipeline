package pipeline

import "context"

type StageFunc[I any, O any] func(context.Context, I) (O, error)

type Builder[I any] struct {
	run func(context.Context, <-chan any) <-chan Result[any]
}

func NewBuilder[I any]() *Builder[I] {
	return &Builder[I]{
		run: func(ctx context.Context, in <-chan any) <-chan Result[any] {
			out := make(chan Result[any])
			go func() {
				defer close(out)
				for v := range in {
					out <- Result[any]{Val: v}
				}
			}()
			return out
		},
	}
}

func Then[I any, O any](b *Builder[I], fn Handler[I, O], opts ...Option[I, O]) *Builder[O] {
	stage := newStage(append(opts, WithHandler(fn))...)

	return &Builder[O]{
		run: func(ctx context.Context, in <-chan any) <-chan Result[any] {
			typed := make(chan I)
			go func() {
				defer close(typed)
				for v := range in {
					typed <- v.(I)
				}
			}()

			out := stage.run(ctx, typed)
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

func (b *Builder[I]) Build() *Pipeline[I, I] {
	return &Pipeline[I, I]{
		run: func(ctx context.Context, in <-chan I) <-chan Result[I] {
			anyIn := make(chan any)
			go func() {
				defer close(anyIn)
				for v := range in {
					anyIn <- v
				}
			}()

			outAny := b.run(ctx, anyIn)
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

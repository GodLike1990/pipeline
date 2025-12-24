package pipeline

import "context"

type Result[T any] struct {
	Val T
	Err error
}

type Pipeline[I any, O any] struct {
	run func(context.Context, <-chan I) <-chan Result[O]
}

func New[I any, O any](opts ...Option[I, O]) *Pipeline[I, O] {
	s := newStage(opts...)
	return &Pipeline[I, O]{
		run: func(ctx context.Context, in <-chan I) <-chan Result[O] {
			return s.run(ctx, in)
		},
	}
}

func (p *Pipeline[I, O]) Run(ctx context.Context, in <-chan I) <-chan Result[O] {
	return p.run(ctx, in)
}

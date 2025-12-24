package pipeline

func Batch[T any](data []T) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		for _, v := range data {
			out <- v
		}
	}()
	return out
}

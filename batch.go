package pipeline

// Batch 将切片数据转换为channel，用于批量输入处理
// 这是将静态数据转换为流式数据的便捷方法
// data: 要转换的切片数据
// 返回: 只读channel，按顺序发送切片中的每个元素
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

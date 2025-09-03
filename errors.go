package workerpool

import "errors"

var (
	// ErrTaskQueueFull 队列已满,当 Enqueue 时队列已满应返回此错误
	ErrTaskQueueFull = errors.New("task queue full")
	// ErrTaskQueueEmpty 队列为空,当 Dequeue 没数据时应返回此错误
	ErrTaskQueueEmpty      = errors.New("task queue empty")
	ErrTaskQueueNotFound   = errors.New("task queue not found")
	ErrTaskHandlerNotFound = errors.New("task handler not found")
)

package wp

import "context"

type WorkerPool interface {
	Submit(ctx context.Context, task *Task, opts ...TaskOption) error
	Run() error
	Stop() error
	Resize(workerSize uint32) error
}

type Worker interface {
	Run() error
	Stop() error
}

type TaskQueue interface {
	// Enqueue 入队
	Enqueue(ctx context.Context, task *Task) error
	// Dequeue 出队,队列已满时返回 ErrTaskQueueFull 错误
	Dequeue(ctx context.Context) (task *Task, err error)
}

type Logger interface {
	Info(ctx context.Context, format string, args ...any)
	Warn(ctx context.Context, format string, args ...any)
}

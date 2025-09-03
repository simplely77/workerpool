package wp

import (
	"context"
	"time"

	"github.com/google/uuid"
)

var taskHandlers = make(map[string]TaskHandler)

type TaskHandler func(ctx context.Context, task *Task) error

type Task struct {
	TaskOptions
	ID  string `json:"id"`
	Key string `json:"key"`
}

type TaskOptions struct {
	Payload []byte        `json:"payload"`
	Timeout time.Duration `json:"timeout"`
	Queue   string        `json:"queue"`
}

type TaskOption func(*TaskOptions)

func (o *TaskOptions) Apply(opts ...TaskOption) {
	for _, opt := range opts {
		opt(o)
	}
}

func WithTaskPayload(payload []byte) TaskOption {
	return func(opts *TaskOptions) {
		opts.Payload = payload
	}
}

func WithTaskQueue(queue string) TaskOption {
	return func(opts *TaskOptions) {
		opts.Queue = queue
	}
}

func NewTask(key string, opts ...TaskOption) *Task {
	o := TaskOptions{
		Queue:   "default",
		Timeout: time.Hour,
	}
	o.Apply(opts...)

	return &Task{
		TaskOptions: o,
		ID:          uuid.New().String(),
		Key:         key,
	}
}

func RegisterTaskHandler(key string, handler TaskHandler) {
	taskHandlers[key] = handler
}

func taskHandler(key string) TaskHandler {
	return taskHandlers[key]
}

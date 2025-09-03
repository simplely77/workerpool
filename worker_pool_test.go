package workerpool

import (
	"context"
	"sync"
	"testing"
)

// 模拟的任务处理器
func init() {
	RegisterTaskHandler("task", func(ctx context.Context, task *Task) error {
		// 快速任务
		return nil
	})
}

// BenchmarkWorkerPool_Submit 测试任务提交性能
func BenchmarkWorkerPool_Submit(b *testing.B) {
	queue := NewMemoryTaskQueue(1000)
	pool := NewWorkerPool(&WorkerPoolConfig{
		TaskQueues: map[string]TaskQueue{
			"default": queue,
		},
		Logger: NewNopLogger(),
	})
	err := pool.Run()
	if err != nil {
		b.Fatal(err)
	}
	defer pool.Stop()

	task := NewTask("task")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// 不停丢弃避免OOM
			_, _ = queue.Dequeue(context.Background())
			err := pool.Submit(context.Background(), task)
			if err != nil {
				b.Error(err)
			}
		}
	})
}

type fifoTaskQueue struct {
	data  []*Task
	mutex sync.Mutex
}

func (q *fifoTaskQueue) Enqueue(_ context.Context, task *Task) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.data = append(q.data, task)
	return nil
}

func (q *fifoTaskQueue) Dequeue(_ context.Context) (*Task, error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if len(q.data) == 0 {
		return nil, ErrTaskQueueEmpty
	}
	task := q.data[0]
	q.data = q.data[1:]
	return task, nil
}

// BenchmarkWorkerPool_Submit_FIFOQueue 测试任务提交性能（FIFO有锁队列）
func BenchmarkWorkerPool_Submit_FIFOQueue(b *testing.B) {
	queue := &fifoTaskQueue{}
	pool := NewWorkerPool(&WorkerPoolConfig{
		TaskQueues: map[string]TaskQueue{
			"default": queue,
		},
		Logger: NewNopLogger(),
	})
	err := pool.Run()
	if err != nil {
		b.Fatal(err)
	}
	defer pool.Stop()

	task := NewTask("task")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// 不停丢弃避免OOM
			_, _ = queue.Dequeue(context.Background())
			err := pool.Submit(context.Background(), task)
			if err != nil {
				b.Error(err)
			}
		}
	})
}

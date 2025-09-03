package workerpool

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	_ "go.uber.org/automaxprocs"
)

type workerPool struct {
	conf    *WorkerPoolConfig
	running atomic.Bool
	workers []Worker
	mutex   sync.Mutex
}

type WorkerPoolConfig struct {
	// WorkerSize 工作协程数量,默认值为 CPU 数量
	WorkerSize uint32
	// TaskQueue 任务队列实现,如果没传则默认使用 MemoryTaskQueue 内存任务队列,容量1000
	TaskQueues map[string]TaskQueue
	// 队列优先级
	TaskQueuePriority map[string]int
	// Logger 日志记录器,如果没传则默认使用 StdLogger 标准日志记录器
	Logger Logger
	// NewContext 创建新的上下文,如果没传则默认使用 context.Background()
	//  可用于创建带有 request_id 或其他元数据的上下文
	NewContext func() context.Context
	// GracefulShutdownTimeout 优雅关闭超时时间,默认值为 5 秒
	GracefulShutdownTimeout time.Duration

	priorityTaskQueue *priorityTaskQueue
}

var _ WorkerPool = &workerPool{}

func NewWorkerPool(conf *WorkerPoolConfig) WorkerPool {
	if conf.WorkerSize == 0 {
		conf.WorkerSize = uint32(runtime.GOMAXPROCS(0))
	}

	if conf.TaskQueues == nil {
		conf.TaskQueues = map[string]TaskQueue{
			"default": NewMemoryTaskQueue(1000),
		}
	}
	if conf.TaskQueuePriority == nil {
		conf.TaskQueuePriority = map[string]int{
			"default": 1,
		}
	}
	if conf.Logger == nil {
		conf.Logger = NewStdLogger()
	}
	if conf.NewContext == nil {
		conf.NewContext = func() context.Context {
			return context.Background()
		}
	}
	if conf.GracefulShutdownTimeout == 0 {
		conf.GracefulShutdownTimeout = 5 * time.Second
	}

	// 根据优先级对任务队列进行排序
	priorityQueues := make([]queueWithPriority, 0, len(conf.TaskQueues))
	for name, queue := range conf.TaskQueues {
		priorityQueues = append(priorityQueues, queueWithPriority{
			Name:     name,
			Queue:    queue,
			Priority: conf.TaskQueuePriority[name],
		})
	}
	conf.priorityTaskQueue = newPriorityTaskQueue(priorityQueues...)

	return &workerPool{
		conf: conf,
	}
}

func (p *workerPool) Submit(ctx context.Context, task *Task, opts ...TaskOption) error {
	if !p.running.Load() {
		return errors.New("worker pool is not running")
	}

	// 创建 task 副本避免修改原始 task
	newTask := *task
	// 应用所有选项
	for _, opt := range opts {
		opt(&newTask.TaskOptions)
	}

	queue, ok := p.conf.TaskQueues[task.Queue]
	if !ok {
		return errors.WithStack(ErrTaskQueueNotFound)
	}

	return queue.Enqueue(ctx, task)
}

func (p *workerPool) Run() error {
	if !p.running.CompareAndSwap(false, true) {
		return errors.New("worker pool is running")
	}

	for i, size := 0, p.conf.WorkerSize; i < int(size); i++ {
		worker := newWorker(p.conf)
		p.workers = append(p.workers, worker)
		err := worker.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *workerPool) Stop() error {
	if !p.running.CompareAndSwap(true, false) {
		return errors.New("worker pool is not running")
	}

	ctx := p.conf.NewContext()
	done := make(chan struct{})
	var wg sync.WaitGroup

	// 启动 goroutine 等待所有 worker 停止
	go func() {
		defer close(done)
		for _, worker := range p.workers {
			worker := worker
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := worker.Stop()
				if err != nil {
					p.conf.Logger.Warn(ctx, "worker pool stop error: %v", err)
				}
			}()
		}
		wg.Wait()
	}()

	// 等待优雅关闭或超时
	select {
	case <-done:
		p.conf.Logger.Info(ctx, "worker pool stopped gracefully")
		return nil
	case <-time.After(p.conf.GracefulShutdownTimeout):
		p.conf.Logger.Warn(ctx, "worker pool graceful shutdown timeout after %v", p.conf.GracefulShutdownTimeout)
		return errors.New("graceful shutdown timeout")
	}
}

func (p *workerPool) Resize(workerSize uint32) error {
	if workerSize == 0 || workerSize == p.conf.WorkerSize {
		return nil
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 缩容
	if workerSize < p.conf.WorkerSize {
		for i := workerSize; i < p.conf.WorkerSize; i++ {
			if i < uint32(len(p.workers)) {
				err := p.workers[i].Stop()
				if err != nil {
					p.conf.Logger.Warn(context.Background(), "worker pool failed to stop worker: %v, idx: %d", err.Error(), i)
				}
			}
		}
		p.workers = p.workers[:workerSize]
		return nil
	}

	// 扩容
	for i := p.conf.WorkerSize; i < workerSize; i++ {
		worker := newWorker(p.conf)
		p.workers = append(p.workers, worker)
		err := worker.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

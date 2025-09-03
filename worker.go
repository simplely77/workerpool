package wp

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
)

type worker struct {
	conf    *WorkerPoolConfig
	running atomic.Bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

var _ Worker = &worker{}

func newWorker(conf *WorkerPoolConfig) *worker {
	return &worker{
		conf:   conf,
		stopCh: make(chan struct{}),
	}
}

func (w *worker) Run() error {
	if !w.running.CompareAndSwap(false, true) {
		return errors.New("already running")
	}

	w.wg.Add(1) // 在启动 goroutine 之前调用 Add(1)
	go func() {
		defer w.wg.Done()
		for {
			select {
			case <-w.stopCh:
				return

			default:
				ctx := w.conf.NewContext()
				err := w.consume(ctx)
				if err != nil && !errors.Is(err, ErrTaskQueueEmpty) {
					// 只记录真正的错误，队列空是正常状态
					w.conf.Logger.Warn(ctx, "worker consume task error: %v", err)
				}
				// 无论成功、失败还是队列空，都继续下一轮循环
			}
		}
	}()
	return nil
}

func (w *worker) Stop() error {
	if !w.running.CompareAndSwap(true, false) {
		return errors.New("not running")
	}

	close(w.stopCh)
	w.wg.Wait()
	return nil
}

func (w *worker) consume(ctx context.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = errors.WithStack(e)
			} else {
				err = errors.Errorf("panic recovered: %v", r)
			}
		}
	}()

	dequeueCtx, dequeueCancel := context.WithTimeout(ctx, time.Second)
	defer dequeueCancel()
	task, err := w.conf.priorityTaskQueue.Dequeue(dequeueCtx)
	if err != nil {
		return err
	}

	handler := taskHandler(task.Key)
	if handler == nil {
		return errors.WithStack(ErrTaskHandlerNotFound)
	}

	taskCtx, taskCancel := context.WithTimeout(ctx, task.Timeout)
	defer taskCancel()
	return handler(taskCtx, task)
}

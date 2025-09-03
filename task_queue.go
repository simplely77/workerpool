package wp

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/pkg/errors"

	"github.com/hedzr/go-ringbuf/v2"
	"github.com/hedzr/go-ringbuf/v2/mpmc"
	"github.com/redis/go-redis/v9"
)

type (
	// MemoryTaskQueue 本地任务队列
	MemoryTaskQueue struct {
		data mpmc.RingBuffer[*Task]
	}

	// RedisTaskQueue Redis任务队列
	RedisTaskQueue struct {
		rdb redis.UniversalClient
	}

	queueWithPriority struct {
		Name     string
		Queue    TaskQueue
		Priority int
	}

	//  优先级任务队列,会优先出队高优任务
	priorityTaskQueue struct {
		queues []TaskQueue
	}
)

var (
	_ TaskQueue = &MemoryTaskQueue{}
	_ TaskQueue = &RedisTaskQueue{}
)

func NewMemoryTaskQueue(capacity uint32) *MemoryTaskQueue {
	return &MemoryTaskQueue{
		data: ringbuf.New[*Task](capacity),
	}
}

func NewRedisTaskQueue(rdb redis.UniversalClient) *RedisTaskQueue {
	return &RedisTaskQueue{
		rdb: rdb,
	}
}

func newPriorityTaskQueue(queues ...queueWithPriority) *priorityTaskQueue {
	// 按优先级降序排序（优先级高的在前）
	sort.Slice(queues, func(i, j int) bool {
		return queues[i].Priority > queues[j].Priority
	})

	taskQueues := make([]TaskQueue, len(queues))
	for i, queue := range queues {
		taskQueues[i] = queue.Queue
	}

	return &priorityTaskQueue{
		queues: taskQueues,
	}
}

func (q *MemoryTaskQueue) Enqueue(_ context.Context, task *Task) error {
	err := q.data.Enqueue(task)
	if errors.Is(err, mpmc.ErrQueueFull) {
		return errors.WithStack(ErrTaskQueueFull)
	}
	return err
}

func (q *MemoryTaskQueue) Dequeue(_ context.Context) (*Task, error) {
	task, err := q.data.Dequeue()
	if errors.Is(err, mpmc.ErrQueueEmpty) {
		return nil, errors.WithStack(ErrTaskQueueEmpty)
	}
	return task, nil
}

func (q *RedisTaskQueue) Enqueue(ctx context.Context, task *Task) error {
	data, err := json.Marshal(task)
	if err != nil {
		return errors.WithStack(err)
	}

	return q.rdb.LPush(ctx, "task_queue", data).Err()
}

func (q *RedisTaskQueue) Dequeue(ctx context.Context) (*Task, error) {
	result, err := q.rdb.RPop(ctx, "task_queue").Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, errors.WithStack(ErrTaskQueueEmpty)
		}
		return nil, errors.WithStack(err)
	}

	task := &Task{}
	err = json.Unmarshal(result, task)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return task, nil
}

func (q *priorityTaskQueue) Dequeue(ctx context.Context) (*Task, error) {
	for _, queue := range q.queues {
		task, err := queue.Dequeue(ctx)
		if err != nil {
			if errors.Is(err, ErrTaskQueueEmpty) {
				continue // Try next queue
			}
			return nil, errors.WithStack(err) // Return error if not empty
		}
		return task, nil // Return the first available task
	}
	return nil, errors.WithStack(ErrTaskQueueEmpty) // All queues are empty
}

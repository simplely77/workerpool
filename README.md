# WorkerPool

一个高性能、功能丰富的 Go 语言协程池/任务队列框架，支持多队列、优先级调度、动态扩缩容等企业级特性。

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.24-blue.svg)](https://golang.org/)
[![GitHub](https://img.shields.io/badge/GitHub-simplely77%2Fworkerpool-blue)](https://github.com/simplely77/workerpool)

## ✨ 核心特性

- 🚀 **高性能**: 基于无锁环形缓冲区，支持百万级任务吞吐
- 🎯 **多队列支持**: 内存队列、Redis 队列，支持优先级调度
- 📈 **动态扩缩容**: 运行时动态调整 Worker 数量
- 🛡️ **健壮性**: 完善的错误处理、Panic 恢复、优雅关闭
- 🔌 **可插拔**: 可自定义任务处理器、日志器、队列实现
- ⚡ **轻量级**: 零外部依赖的核心实现
- 🔧 **易扩展**: 清晰的接口设计，便于自定义扩展

## 📦 安装

```bash
go get github.com/simplely77/workerpool@latest
```

## 🚀 快速开始

### 基础使用

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/simplely77/workerpool"
)

func main() {
    // 1. 注册任务处理器
    workerpool.RegisterTaskHandler("email", func(ctx context.Context, task *workerpool.Task) error {
        fmt.Printf("发送邮件: %s\n", string(task.Payload))
        return nil
    })
    
    workerpool.RegisterTaskHandler("sms", func(ctx context.Context, task *workerpool.Task) error {
        fmt.Printf("发送短信: %s\n", string(task.Payload))
        return nil
    })
    
    // 2. 创建 WorkerPool
    pool := workerpool.NewWorkerPool(&workerpool.WorkerPoolConfig{
        WorkerSize: 10, // 10 个 Worker
        Logger:     workerpool.NewStdLogger(),
    })
    
    // 3. 启动 WorkerPool
    if err := pool.Run(); err != nil {
        log.Fatal(err)
    }
    defer pool.Stop()
    
    // 4. 提交任务
    emailTask := workerpool.NewTask("email", workerpool.WithTaskPayload([]byte("欢迎注册！")))
    smsTask := workerpool.NewTask("sms", workerpool.WithTaskPayload([]byte("验证码: 123456")))
    
    pool.Submit(context.Background(), emailTask)
    pool.Submit(context.Background(), smsTask)
    
    // 等待任务完成
    time.Sleep(2 * time.Second)
}
```

### 多队列和优先级

```go
// 创建多个队列，不同优先级
pool := workerpool.NewWorkerPool(&workerpool.WorkerPoolConfig{
    WorkerSize: 5,
    TaskQueues: map[string]workerpool.TaskQueue{
        "high":   workerpool.NewMemoryTaskQueue(1000),    // 高优先级队列
        "normal": workerpool.NewMemoryTaskQueue(2000),    // 普通队列
        "low":    workerpool.NewMemoryTaskQueue(500),     // 低优先级队列
    },
    TaskQueuePriority: map[string]int{
        "high":   100,  // 高优先级
        "normal": 50,   // 中优先级
        "low":    10,   // 低优先级
    },
})

// 提交到不同队列
urgentTask := workerpool.NewTask("process", 
    workerpool.WithTaskQueue("high"),
    workerpool.WithTaskPayload([]byte("紧急任务")))

normalTask := workerpool.NewTask("process", 
    workerpool.WithTaskQueue("normal"),
    workerpool.WithTaskPayload([]byte("普通任务")))
```

### Redis 队列

```go
import "github.com/redis/go-redis/v9"

// 使用 Redis 作为队列存储
rdb := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

pool := workerpool.NewWorkerPool(&workerpool.WorkerPoolConfig{
    TaskQueues: map[string]workerpool.TaskQueue{
        "default": workerpool.NewRedisTaskQueue(rdb),
    },
})
```

## 🏗️ 架构设计

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Client API    │───▶│   WorkerPool     │───▶│   TaskQueue     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │                          │
                              ▼                          ▼
                       ┌──────────────┐         ┌─────────────────┐
                       │   Worker     │◀────────│ PriorityQueue   │
                       └──────────────┘         └─────────────────┘
                              │                          │
                              ▼                          ▼
                       ┌──────────────┐         ┌─────────────────┐
                       │ TaskHandler  │         │ Memory/Redis    │
                       └──────────────┘         └─────────────────┘
```

## 📚 详细文档

### 配置选项

```go
type WorkerPoolConfig struct {
    // Worker 数量，默认为 CPU 核数
    WorkerSize uint32
    
    // 任务队列映射，key 为队列名称
    TaskQueues map[string]TaskQueue
    
    // 队列优先级，数值越大优先级越高
    TaskQueuePriority map[string]int
    
    // 日志器，默认为标准输出
    Logger Logger
    
    // 上下文创建函数，可用于注入 request_id 等
    NewContext func() context.Context
    
    // 优雅关闭超时时间，默认 5 秒
    GracefulShutdownTimeout time.Duration
}
```

### 任务选项

```go
// 创建任务时的可选参数
task := workerpool.NewTask("handler_key",
    workerpool.WithTaskPayload([]byte("数据")),    // 任务数据
    workerpool.WithTaskQueue("high"),              // 指定队列
)

// 任务还支持超时设置
type TaskOptions struct {
    Payload []byte        // 任务载荷
    Timeout time.Duration // 任务超时时间，默认 1 小时
    Queue   string        // 目标队列，默认 "default"
}
```

### 错误处理

```go
var (
    ErrTaskQueueFull       = errors.New("task queue full")
    ErrTaskQueueEmpty      = errors.New("task queue empty")
    ErrTaskQueueNotFound   = errors.New("task queue not found")
    ErrTaskHandlerNotFound = errors.New("task handler not found")
)
```

## 🔧 自定义扩展

### 自定义队列

```go
type MyCustomQueue struct {
    // 自定义队列实现
}

func (q *MyCustomQueue) Enqueue(ctx context.Context, task *workerpool.Task) error {
    // 实现入队逻辑
    return nil
}

func (q *MyCustomQueue) Dequeue(ctx context.Context) (*workerpool.Task, error) {
    // 实现出队逻辑
    return nil, nil
}
```

### 自定义日志

```go
type MyLogger struct {
    logger *logrus.Logger
}

func (l *MyLogger) Info(ctx context.Context, format string, args ...interface{}) {
    l.logger.Infof(format, args...)
}

func (l *MyLogger) Warn(ctx context.Context, format string, args ...interface{}) {
    l.logger.Warnf(format, args...)
}
```

### 复杂任务处理器

```go
workerpool.RegisterTaskHandler("image_process", func(ctx context.Context, task *workerpool.Task) error {
    // 解析任务数据
    var req ImageProcessRequest
    if err := json.Unmarshal(task.Payload, &req); err != nil {
        return err
    }
    
    // 执行图片处理
    result, err := processImage(ctx, req)
    if err != nil {
        return err
    }
    
    // 保存结果
    return saveResult(ctx, result)
})
```

## 🧪 使用示例

### Web 服务集成

```go
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
    // 处理 HTTP 请求
    user := parseUser(r)
    
    // 异步发送欢迎邮件
    emailTask := workerpool.NewTask("welcome_email",
        workerpool.WithTaskPayload(user.Email),
        workerpool.WithTaskQueue("email"))
    
    h.workerPool.Submit(r.Context(), emailTask)
    
    // 立即返回响应
    w.WriteHeader(http.StatusCreated)
}
```

### 批量任务处理

```go
func ProcessBatch(items []Item) {
    pool := workerpool.NewWorkerPool(&workerpool.WorkerPoolConfig{
        WorkerSize: 50, // 并发处理 50 个任务
    })
    pool.Run()
    defer pool.Stop()
    
    for _, item := range items {
        task := workerpool.NewTask("process_item",
            workerpool.WithTaskPayload(item.ToBytes()))
        pool.Submit(context.Background(), task)
    }
}
```

### 定时任务调度

```go
func StartScheduler(pool workerpool.WorkerPool) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        // 每 5 分钟执行一次清理任务
        task := workerpool.NewTask("cleanup", 
            workerpool.WithTaskQueue("maintenance"))
        pool.Submit(context.Background(), task)
    }
}
```

## 📊 性能测试

运行基准测试：

```bash
go test -bench=BenchmarkWorkerPool -benchmem
```

典型性能数据（在 12th Gen Intel Core i3-12100F 上）：

```
BenchmarkWorkerPool_Submit-8           	 3853870	       322.7 ns/op	     482 B/op	       5 allocs/op
BenchmarkWorkerPool_Submit_FIFOQueue-8 	 2543247	       456.6 ns/op	     612 B/op	       8 allocs/op
```

## 🛠️ 依赖

### 核心依赖
- **Go 1.24+**: 使用最新 Go 特性
- **github.com/google/uuid**: UUID 生成
- **github.com/pkg/errors**: 错误处理增强
- **github.com/hedzr/go-ringbuf/v2**: 无锁环形缓冲区

### 可选依赖
- **github.com/redis/go-redis/v9**: Redis 队列支持
- **go.uber.org/automaxprocs**: 自动设置 GOMAXPROCS

## 🎯 最佳实践

1. **合理设置 Worker 数量**: 建议为 CPU 核数的 1-2 倍
2. **使用适当的队列容量**: 避免内存占用过大
3. **设置合理的任务超时**: 防止任务长时间占用资源
4. **优雅关闭**: 确保所有任务完成后再退出
5. **监控队列长度**: 及时发现性能瓶颈
6. **错误处理**: 任务处理器应妥善处理各种异常情况

## 🔗 相关链接

- [GitHub 仓库](https://github.com/simplely77/workerpool)
- [Go 并发编程指南](https://golang.org/doc/effective_go.html#concurrency)
- [任务队列设计模式](https://en.wikipedia.org/wiki/Message_queue)
- [性能优化技巧](https://dave.cheney.net/high-performance-go-workshop/dotgo-paris.html)

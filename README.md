# WorkerPool

一个高性能、功能丰富的 Go 语言协程池/任务队列框架，支持多队列、优先级调度、动态扩缩容等企业级特性。

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.24-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

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
go get github.com/simplely77/workerpool
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
    
    "workerpool"
)

func main() {
    // 1. 注册任务处理器
    wp.RegisterTaskHandler("email", func(ctx context.Context, task *wp.Task) error {
        fmt.Printf("发送邮件: %s\n", string(task.Payload))
        return nil
    })
    
    wp.RegisterTaskHandler("sms", func(ctx context.Context, task *wp.Task) error {
        fmt.Printf("发送短信: %s\n", string(task.Payload))
        return nil
    })
    
    // 2. 创建 WorkerPool
    pool := wp.NewWorkerPool(&wp.WorkerPoolConfig{
        WorkerSize: 10, // 10 个 Worker
        Logger:     wp.NewStdLogger(),
    })
    
    // 3. 启动 WorkerPool
    if err := pool.Run(); err != nil {
        log.Fatal(err)
    }
    defer pool.Stop()
    
    // 4. 提交任务
    emailTask := wp.NewTask("email", wp.WithTaskPayload([]byte("欢迎注册！")))
    smsTask := wp.NewTask("sms", wp.WithTaskPayload([]byte("验证码: 123456")))
    
    pool.Submit(context.Background(), emailTask)
    pool.Submit(context.Background(), smsTask)
    
    // 等待任务完成
    time.Sleep(2 * time.Second)
}
```

### 多队列和优先级

```go
// 创建多个队列，不同优先级
pool := wp.NewWorkerPool(&wp.WorkerPoolConfig{
    WorkerSize: 5,
    TaskQueues: map[string]wp.TaskQueue{
        "high":   wp.NewMemoryTaskQueue(1000),    // 高优先级队列
        "normal": wp.NewMemoryTaskQueue(2000),    // 普通队列
        "low":    wp.NewMemoryTaskQueue(500),     // 低优先级队列
    },
    TaskQueuePriority: map[string]int{
        "high":   100,  // 高优先级
        "normal": 50,   // 中优先级
        "low":    10,   // 低优先级
    },
})

// 提交到不同队列
urgentTask := wp.NewTask("process", 
    wp.WithTaskQueue("high"),
    wp.WithTaskPayload([]byte("紧急任务")))

normalTask := wp.NewTask("process", 
    wp.WithTaskQueue("normal"),
    wp.WithTaskPayload([]byte("普通任务")))
```

### Redis 队列

```go
import "github.com/redis/go-redis/v9"

// 使用 Redis 作为队列存储
rdb := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

pool := wp.NewWorkerPool(&wp.WorkerPoolConfig{
    TaskQueues: map[string]wp.TaskQueue{
        "default": wp.NewRedisTaskQueue(rdb),
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
task := wp.NewTask("handler_key",
    wp.WithTaskPayload([]byte("数据")),    // 任务数据
    wp.WithTaskQueue("high"),              // 指定队列
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

func (q *MyCustomQueue) Enqueue(ctx context.Context, task *wp.Task) error {
    // 实现入队逻辑
    return nil
}

func (q *MyCustomQueue) Dequeue(ctx context.Context) (*wp.Task, error) {
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
wp.RegisterTaskHandler("image_process", func(ctx context.Context, task *wp.Task) error {
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
    emailTask := wp.NewTask("welcome_email",
        wp.WithTaskPayload(user.Email),
        wp.WithTaskQueue("email"))
    
    h.workerPool.Submit(r.Context(), emailTask)
    
    // 立即返回响应
    w.WriteHeader(http.StatusCreated)
}
```

### 批量任务处理

```go
func ProcessBatch(items []Item) {
    pool := wp.NewWorkerPool(&wp.WorkerPoolConfig{
        WorkerSize: 50, // 并发处理 50 个任务
    })
    pool.Run()
    defer pool.Stop()
    
    for _, item := range items {
        task := wp.NewTask("process_item",
            wp.WithTaskPayload(item.ToBytes()))
        pool.Submit(context.Background(), task)
    }
}
```

### 定时任务调度

```go
func StartScheduler(pool wp.WorkerPool) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        // 每 5 分钟执行一次清理任务
        task := wp.NewTask("cleanup", 
            wp.WithTaskQueue("maintenance"))
        pool.Submit(context.Background(), task)
    }
}
```

## 📊 性能测试

运行基准测试：

```bash
go test -bench=. -benchmem
```

典型性能数据（在 8 核 CPU 上）：

```
BenchmarkWorkerPool_Submit-8           	 5000000	       263 ns/op	      64 B/op	       2 allocs/op
BenchmarkWorkerPool_Submit_FIFOQueue-8 	 2000000	       824 ns/op	      64 B/op	       2 allocs/op
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

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 🔗 相关链接

- [Go 并发编程指南](https://golang.org/doc/effective_go.html#concurrency)
- [任务队列设计模式](https://en.wikipedia.org/wiki/Message_queue)
- [性能优化技巧](https://dave.cheney.net/high-performance-go-workshop/dotgo-paris.html)

# 一、说明

Go 协程工具函数库，提供并发执行、流水线、限频、安全队列等高层抽象。

[![codecov](https://codecov.io/gh/ivfzhou/goroutine-util/graph/badge.svg?token=JO8RFXW1SP)](https://codecov.io/gh/ivfzhou/goroutine-util)
[![Go Reference](https://pkg.go.dev/badge/gitee.com/ivfzhou/goroutine-util.svg)](https://pkg.go.dev/gitee.com/ivfzhou/goroutine-util)

# 二、安装

```shell
go get gitee.com/ivfzhou/goroutine-util@latest
```

> 需要 Go 1.18+ (支持泛型)

# 三、功能概览

| 模块 | 说明 |
|------|------|
| **Runner** | 有界协程池，动态提交任务，支持错误快速退出 |
| **RunData** | 批量数据并发处理（一次性） |
| **RunConcurrently / RunSequentially** | 并发/顺序执行多个函数 |
| **Pipeline** | 批量流水线，任务依次经过多阶段处理 |
| **PipelineRunner** | 流式流水线，增量推送任务 |
| **Queue** | 线程安全队列，支持通道消费 |
| **ListenChan** | 多通道聚合监听 |
| **RunPeriodically** | 限频执行，控制调用间隔 |
| **Error / AtomicError** | 错误包装与原子错误存储 |

# 四、API 文档

## 4.1 Runner — 有界协程池

创建一个最多 `max` 个协程并发运行的执行器，支持动态添加任务。`max <= 0` 时不限制并发数。

```go
func NewRunner[T any](ctx context.Context, max int, fn func(context.Context, T) error) (
    add func(t T, block bool) error,
    wait func(fastExit bool) error,
)
```

- **`add(t, block)`** — 提交任务 `t`；`block=true` 时池满则阻塞
- **`wait(fastExit)`** — 等待所有任务完成；`fastExit=true` 时首个错误立即返回
- 首个错误或 panic 会终止所有协程，panic 自动恢复为 error 并携带调用栈
- `add` 必须在 `wait` 之前调用，否则 panic（返回 `ErrCallAddAfterWait`）

```go
ctx := context.Background()
add, wait := NewRunner(ctx, 10, func(ctx context.Context, url string) error {
    resp, err := http.Get(url)
    if err != nil { return err }
    defer resp.Body.Close()
    // ...
    return nil
})

for _, url := range urls {
    if err := add(url, true); err != nil { return err }
}
if err := wait(true); err != nil { return err }
```

## 4.2 RunData — 批量并发处理

将一组数据并发交给 `fn` 处理，发生错误立即终止。

```go
func RunData[T any](ctx context.Context, fn func(context.Context, T) error, fastExit bool, jobs ...T) error
```

```go
err := RunData(ctx, func(ctx context.Context, id int) error {
    return process(id)
}, true, 1, 2, 3, 4, 5)
```

## 4.3 并发 / 顺序执行

```go
// 并发运行多个函数，每个函数独立协程
func RunConcurrently(ctx context.Context, fn ...func(context.Context) error) (
    wait func(fastExit bool) error,
)

// 顺序运行多个函数，遇到错误即停止
func RunSequentially(ctx context.Context, fn ...func(context.Context) error) error
```

```go
wait := RunConcurrently(ctx,
    func(ctx context.Context) error { return taskA() },
    func(ctx context.Context) error { return taskB() },
    func(ctx context.Context) error { return taskC() },
)
if err := wait(true); err != nil { /* ... */ }
```

## 4.4 Pipeline — 批量流水线

将所有 `jobs` 依次经过各 `step` 处理，每个 step 内部并发处理 jobs。

```go
func RunPipeline[T any](
    ctx context.Context,
    jobs []T,
    stopWhenErr bool,
    steps ...func(context.Context, T) error,
) (successCh <-chan T, errCh <-chan error)
```

- **`successCh`** — 通过全部 step 的任务
- **`errCh`** — 运行中产生的错误
- **`stopWhenErr`** — 为 true 时首个错误终止整个流水线；false 则仅停止当前 job 的后续传递
- 每个 step 的默认并发上限由 `RunPipelineCacheJobs`（默认 10）控制

```go
successCh, errCh := RunPipeline(ctx, data, true,
    func(ctx context.Context, d Data) error { return validate(d) },
    func(ctx context.Context, d Data) error { return transform(d) },
    func(ctx context.Context, d Data) error { return save(d) },
)
for d := range successCh { fmt.Println("成功:", d) }
for err := range errCh   { log.Println("失败:", err) }
```

## 4.5 PipelineRunner — 流式流水线

流式版本，支持增量推送任务。step 返回 bool 而非 error（true = 传递给下一步）。

```go
func NewPipelineRunner[T any](ctx context.Context, steps ...func(context.Context, T) bool) (
    push func(T) bool,
    successCh <-chan T,
    endPush func(),
)
```

- **`push(t)`** — 提交一个任务，成功返回 true
- **`endPush()`** — 表示不再有新任务

```go
push, successCh, endPush := NewPipelineRunner(ctx,
    func(ctx context.Context, req Request) bool { return validate(req) == nil },
    func(ctx context.Context, req Request) bool { return process(req) == nil },
)

go func() {
    for _, req := range requests { push(req) }
    endPush()
}()

for result := range successCh { /* 处理成功结果 */ }
```

## 4.6 ListenChan — 多通道监听

同时监听多个通道，任意一个收到值即刻返回并关闭输出通道。

```go
func ListenChan[T any](chans ...<-chan T) <-chan T
```

```go
result := ListenChan(chA, chB, chC)
val := <-result // 取到第一个值
```

## 4.7 RunPeriodically — 限频执行

确保两次执行之间至少间隔指定时间（上次结束到下次开始之间）。

```go
func RunPeriodically(period time.Duration) func(fn func())
```

```go
run := RunPeriodically(time.Second)
run(func() { doSomething() }) // 立即执行
run(func() { doSomething() }) // 至少距上次结束 1s 后才执行
```

# 五、设计特点

- **Context 传播与取消**：所有 API 接受 `context.Context`，支持级联取消
- **Panic 恢复**：goroutine 内的 panic 自动捕获为 error，附带完整调用栈
- **原子错误存储**：基于 CAS 的 `AtomicError`，确保仅保留首个错误
- **闭包式 API**：通过闭包元组暴露能力，无需维护结构体状态
- **泛型支持**：全模块泛型，类型安全的并发处理

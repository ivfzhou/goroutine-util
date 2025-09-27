# 说明

Go 协程工具函数库

[![codecov](https://codecov.io/gh/ivfzhou/goroutine-util/graph/badge.svg?token=JO8RFXW1SP)](https://codecov.io/gh/ivfzhou/goroutine-util)
[![Go Reference](https://pkg.go.dev/badge/gitee.com/ivfzhou/goroutine-util.svg)](https://pkg.go.dev/gitee.com/ivfzhou/goroutine-util)

# 使用

```shell
go get gitee.com/ivfzhou/goroutine-util@latest
```

```golang
// NewRunner 提供同时最多运行 max 个协程运行 fn，一旦 fn 发生错误或恐慌便终止运行。
//
// ctx：上下文，终止时，停止所有 fn 运行，并在调用 add 或 wait 时返回 ctx.Err()。
//
// max：表示最多 max 个协程运行 fn。小于等于零表示不限制协程数。
//
// fn：为要运行的函数。入参 T 由 add 函数提供。若 fn 发生错误或恐慌，将终止所有 fn 运行。恐慌将被恢复，并以错误形式返回，在调用 add 或 wait 时返回。
//
// add：为 fn 提供 T，每一个 T 只会被一个 fn 运行一次。block 为真时，当某一时刻运行 fn 数量达到 max 时，阻塞当前协程添加 T。反之，不阻塞当前协程。
//
// wait：阻塞当前协程，等待运行完毕。fastExit 为真时，运行发生错误时，立即返回该错误。反之，表示等待所有 fn 都终止后再返回。
//
// add 函数返回的错误不为空时，与 fn 返回的第一个错误一致，且与 wait 函数返回的错误为同一个。
//
// 注意：若 fn 为空，将触发恐慌。
//
// 注意：在 add 完所有 T 后再调用 wait，否则触发恐慌，返回 ErrCallAddAfterWait。
//
// 注意：调用了 add 后，请务必调用 wait，除非 add 返回了错误。
func NewRunner[T any](ctx context.Context, max int, fn func(context.Context, T) error) (
    add func(t T, block bool) error, wait func(fastExit bool) error)

// NewPipelineRunner 创建流水线工作模型。
//
// ctx：上下文。当上下文终止时，将终止流水线运行，push 将返回假。
//
// steps：任务 T 依次在 steps 中运行。step 返回真表示将 T 传递给下一个 step 处理，否则不传递。
//
// push：向 step 中传递一个 T 处理，返回真表示添加成功。
//
// successCh：成功运行完所有 step 的 T 将从该通道发出。
//
// endPush：表示不再有 T 需要处理，push 将返回假。
func NewPipelineRunner[T any](ctx context.Context, steps ...func(context.Context, T) bool) (
    push func(T) bool, successCh <-chan T, endPush func())

// RunData 将 jobs 传给 fn，并发运行 fn。若发生错误，就立刻终止运行。
//
// ctx：上下文。如果上下文终止了，则终止运行，并返回 ctx.Err()。
//
// fn：要运行处理 jobs 的函数。
//
// fastExit：发生错误就立刻返回，不等待所有协程全部退出。
//
// jobs：要处理的任务。
//
// 注意：fn 为空将触发恐慌。
func RunData[T any](ctx context.Context, fn func(context.Context, T) error, fastExit bool, jobs ...T) error

// RunPipeline 将每个 jobs 依次递给 steps 函数处理。一旦某个 step 发生错误或恐慌，就结束处理，返回错误。
// 一个 job 最多在一个 step 中运行一次，且一个 job 一定是依次序递给 steps，前一个 step 处理完毕才会给下一个 step 处理。
// 每个 step 并发运行 jobs。
//
// ctx：上下文，上下文终止时，将终止所有 steps 运行，并在 errCh 中返回 ctx.Err()。
//
// stopWhenErr：当 step 发生错误时，是否继续处理 job。当为假时，只会停止将 job 往下一个 step 传递，不会终止运行。
//
// steps：处理 jobs 的函数。返回错误意味着终止运行。
//
// successCh：经所有 steps 处理成功 job 将从该通道发出。
//
// errCh：运行中的错误从该通道发出。
//
// 注意：若 steps 中含有空元素将会恐慌。
func RunPipeline[T any](ctx context.Context, jobs []T, stopWhenErr bool, steps ...func(context.Context, T) error) (
    successCh <-chan T, errCh <-chan error)

// RunPeriodically 运行 fn，每个 fn 之间至少间隔 period 时间。前一个 fn 运行完毕到下一个 fn 开始运行之间的间隔时间。
//
// period：每个 fn 运行至少间隔时间。
//
// 注意：若 period 为负数将会触发恐慌。
func RunPeriodically(period time.Duration) (run func(fn func()))
```

# 联系作者

电邮：ifzhou@126.com  
微信：h899123

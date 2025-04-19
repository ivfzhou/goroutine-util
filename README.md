# 说明

Go 协程工具函数库

# 使用

```shell
go get gitee.com/ivfzhou/goroutine-util@latest
```

```golang
// RunConcurrently 并发运行fn，一旦有error发生终止运行。
func RunConcurrently(ctx context.Context, fn ...func(context.Context) error) (wait func(fastExit bool) error)

// RunSequentially 依次运行fn，当有error发生时停止后续fn运行。
func RunSequentially(ctx context.Context, fn ...func (context.Context) error) error

// NewRunner 该函数提供同时最多运行 max 个协程运行 fn，一旦 fn 发生 error 或 panic 便终止运行。
//
// ctx 上下文，当 ctx canceled 将终止所有 fn 运行，并返回 ctx.Err()。
//
// max 表示最多多少个协程运行 fn。小于等于 0 表示不限制协程数。
//
// fn 为要运行的函数。T 由 add 函数提供。若 fn 返回 error 或者 panic 则终止所有 fn 运行。fn panic 将被恢复，并以 error 形式返回。
//
// add 为 fn 提供 T。block 为 true 表示当某一时刻运行 fn 数量达到 max 时，阻塞当前协程添加 T。反之，不阻塞当前协程。
//
// wait 阻塞当前协程，等待运行完毕。fastExit 为 true 表示发生错误时，函数立即返回。反之，表示等待所有 fn 都终止后再返回。
//
// add 函数返回的 error 不为 nil 时，是为 fn 返回的第一个 error，且与 wait 函数返回的 error 为同一个。
//
// 若 fn 为 nil 将触发 panic。
//
// 请注意在 add 完所有任务后再调用 wait，否则触发 panic 返回 ErrCallAddAfterWait。
func NewRunner[T any](ctx context.Context, max int, fn func (context.Context, T) error) (add func (t T, block bool) error, wait func (fastExit bool) error)

// RunData 并发将jobs传递给fn函数运行，一旦发生error便立即返回该error，并结束其它协程。
func RunData[T any](ctx context.Context, fn func (context.Context, T) error, fastExit bool, jobs ...T) error

// RunPipeline 将每个jobs依次递给steps函数处理。一旦某个step发生error或者panic，立即返回该error，并及时结束其他协程。
// 除非stopWhenErr为false，则只是终止该job往下一个step投递。
//
// 一个job最多在一个step中运行一次，且一个job一定是依次序递给steps，前一个step处理完毕才会给下一个step处理。
//
// 每个step并发运行jobs。
//
// 等待所有jobs处理结束时会close successCh、errCh，或者ctx被cancel时也将及时结束开启的goroutine后返回。
//
// 从successCh和errCh中获取成功跑完所有step的job和是否发生error。
//
// 若steps中含有nil将会panic。
func RunPipeline[T any](ctx context.Context, jobs []T, stopWhenErr bool, steps ...func (context.Context, T) error) (successCh <-chan T, errCh <-chan error)

// NewPipelineRunner 形同RunPipeline，不同在于使用push推送job。step返回true表示传递给下一个step处理。
func NewPipelineRunner[T any](ctx context.Context, steps ...func (context.Context, T) bool) (push func (T) bool, successCh <-chan T, endPush func ())

// ListenChan 监听chans，一旦有一个chan激活便立即将T发送给ch，并close ch。
//
// 若所有chans都未曾激活（chan是nil也认为未激活）且都close了，则ch被close。
//
// 若同时多个chans被激活，则随机将一个激活值发送给ch。
func ListenChan[T any](chans ...<-chan T) (ch <-chan T)

// RunPeriodically 依次运行fn，每个fn之间至少间隔period时间。
func RunPeriodically(period time.Duration) (run func (fn func ()))

```

# 联系作者

电邮：ifzhou@126.com

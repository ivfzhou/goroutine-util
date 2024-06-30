# 1. 说明

Go 协程工具函数库

# 2. 使用

```shell
go get gitee.com/ivfzhou/goroutine-util@latest
```

```golang
// RunConcurrently 并发运行fn，一旦有error发生终止运行。
func RunConcurrently(ctx context.Context, fn ...func(context.Context) error) (wait func(fastExit bool) error)

// RunSequentially 依次运行fn，当有error发生时停止后续fn运行。
func RunSequentially(ctx context.Context, fn ...func (context.Context) error) error

// NewRunner 该函数提供同时最多运行max个协程fn，一旦fn发生error便终止fn运行。
//
// max小于等于0表示不限制协程数。
//
// 朝返回的run函数中添加fn，若block为true表示正在运行的任务数已达到max则会阻塞。
//
// run函数返回error为任务fn返回的第一个error，与wait函数返回的error为同一个。
//
// 注意请在add完所有任务后调用wait。
func NewRunner[T any](ctx context.Context, max int, fn func (context.Context, T) error) (run func (t T, block bool) error, wait func (fastExit bool) error)

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

# 3. 联系作者

电邮：ifzhou@126.com

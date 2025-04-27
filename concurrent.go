/*
 * Copyright (c) 2023 ivfzhou
 * goroutine-util is licensed under Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *          http://license.coscl.org.cn/MulanPSL2
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
 * EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
 * MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
 * See the Mulan PSL v2 for more details.
 */

package goroutine_util

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrCallAddAfterWait = errType{errors.New("cannot call add after wait")}
)

type errType struct{ error }

// RunConcurrently 并发运行fn，一旦有error发生终止运行。
func RunConcurrently(ctx context.Context, fn ...func(context.Context) error) (wait func(fastExit bool) error) {
	if len(fn) <= 0 {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var err error
	select {
	case <-ctx.Done():
		err = ctx.Err()
		if err == nil {
			err = context.Canceled
		}
		return func(bool) error {
			return err
		}
	default:
	}

	ctx, cancel := context.WithCancel(ctx)
	wg := &sync.WaitGroup{}
	lock := &sync.Mutex{}
	wg.Add(len(fn))
	for _, f := range fn {
		if f == nil {
			panic("fn can't be nil")
		}
		go func(f func(context.Context) error) {
			var ferr error
			defer func() {
				if p := recover(); p != nil {
					ferr = fmt.Errorf("panic: %v [recovered]\n%s\n", p, stackTrace())
				}
				if ferr != nil && !errors.Is(ferr, context.Canceled) {
					lock.Lock()
					if err == nil || errors.Is(err, context.Canceled) {
						err = ferr
					}
					lock.Unlock()
					cancel()
				}
				wg.Done()
			}()
			select {
			case <-ctx.Done():
				lock.Lock()
				if ctxErr := ctx.Err(); ctxErr != nil && err == nil {
					err = ctxErr
				}
				lock.Unlock()
			default:
				ferr = f(ctx)
			}
		}(f)
	}
	go func() {
		wg.Wait()
		cancel()
	}()
	return func(fastExit bool) error {
		if fastExit {
			select {
			case <-ctx.Done():
				return err
			}
		}
		wg.Wait()
		return err
	}
}

// RunSequentially 依次运行fn，当有error发生时停止后续fn运行。
func RunSequentially(ctx context.Context, fn ...func(context.Context) error) error {
	if len(fn) <= 0 {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	select {
	case <-ctx.Done():
		err := ctx.Err()
		if err == nil {
			err = context.Canceled
		}
		return err
	default:
	}

	fnWrapper := func(f func(ctx context.Context) error) (err error) {
		defer func() {
			if p := recover(); p != nil {
				err = fmt.Errorf("panic: %v [recovered]\n%s\n", p, stackTrace())
			}
		}()
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		err = f(ctx)
		return
	}
	for _, f := range fn {
		if f == nil {
			panic("fn can't be nil")
		}
		err := fnWrapper(f)
		if err != nil {
			return err
		}
	}
	return nil
}

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
//
// 调用了 add 后，请务必调用 wait，除非 add 返回了 err。
func NewRunner[T any](ctx context.Context, max int, fn func(context.Context, T) error) (
	add func(t T, block bool) error, wait func(fastExit bool) error) {

	// 校验。
	if fn == nil {
		panic("NewRunner: fn can't be nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	// 如是 ctx canceled，就不必再分配资源，直接返回。
	select {
	case <-ctx.Done():
		err := ctx.Err()
		return func(t T, block bool) error { return err }, func(bool) error { return err }
	default:
	}

	var aerr atomic.Value     // 错误标志。
	var limiter chan struct{} // 限数器。
	if max > 0 {
		limiter = make(chan struct{}, max)
	}
	innerCtx, cancel := context.WithCancel(ctx) // 通知终止 fn 的上下文。
	wg := sync.WaitGroup{}                      // 计数器。
	errSetFlag := make(chan struct{})           // err 是否设置信号。
	errSetOnce := sync.Once{}                   // 设置 err 保护的函数。

	// 设置 err。
	setErrFn := func(fnErr error) {
		if fnErr != nil {
			errSetOnce.Do(func() {
				aerr.Store(fnErr)
				close(errSetFlag)
			})
		}
	}

	// 阻塞获取终止态的 err。
	getErrFn := func() error {
		<-errSetFlag
		v, _ := aerr.Load().(error)
		return v
	}

	// 包装执行函数，减少计数器，减少限数器，cancel innerCtx，以及设置 err。
	fnWrapper := func(t T, useLimiter bool) {
		var fnErr error
		defer func() {
			if p := recover(); p != nil {
				fnErr = fmt.Errorf("panic: %v [recovered]\n%s", p, getStackCallers())
			}
			if fnErr != nil {
				setErrFn(fnErr) // 先设置 err，再通知终止，避免 err 设置成 ctx.Err()。
				cancel()        // 通知终止运行。
			}
			if limiter != nil && useLimiter {
				<-limiter
			}
			wg.Done()
		}()
		select {
		case <-innerCtx.Done():
		default:
			fnErr = fn(ctx, t)
		}
	}

	waitCallFlag := make(chan struct{})
	add = func(t T, block bool) error {
		// 不允许在 wait 后再调用。
		select {
		case <-waitCallFlag:
			panic(ErrCallAddAfterWait)
		default:
		}

		wg.Add(1) // 增加计数器。

		// 不阻塞添加任务。
		if !block || limiter == nil {
			select {
			case <-innerCtx.Done(): // 任务已终止。
				err, _ := aerr.Load().(error)
				return err
			default:
			}
			go func() {
				useLimiter := false
				if limiter != nil {
					useLimiter = true
					limiter <- struct{}{}
				}
				fnWrapper(t, useLimiter)
			}()
			err, _ := aerr.Load().(error)
			return err
		}

		// 阻塞添加任务。
		select {
		case <-innerCtx.Done(): // 任务已终止。
			wg.Done() // 减少计数器。
			err, _ := aerr.Load().(error)
			return err
		case limiter <- struct{}{}: // 获取限数器。
			go fnWrapper(t, true)
			err, _ := aerr.Load().(error)
			return err
		}
	}

	waitOnce := sync.Once{}
	wait = func(fastExit bool) error {
		waitOnce.Do(func() {
			close(waitCallFlag)
			go func() {
				// 等待所有协程退出。
				wg.Wait()

				// 有可能是 ctx canceled。
				select {
				case <-innerCtx.Done(): // innerCtx 只有发生 err 或者 ctx canceled，才会 canceled。
					errSetOnce.Do(func() {
						aerr.Store(ctx.Err())
						close(errSetFlag)
					})
				default:
				}

				// 可能本就没有 err。
				errSetOnce.Do(func() {
					close(errSetFlag)
					cancel()
				})
			}()
		})

		// 快速退出，不等待所有协程完毕。
		if fastExit {
			return getErrFn()
		}

		wg.Wait()
		return getErrFn()
	}

	return
}

/*func NewRunnerWithChan[T any](ctx context.Context, max int, fn func(context.Context, T) error) (
	add func(block bool) chan<- T, wait func(fastExit bool) <-chan error) {

}*/

// RunData 并发将jobs传递给fn函数运行，一旦发生error便立即返回该error，并结束其它协程。
func RunData[T any](ctx context.Context, fn func(context.Context, T) error, fastExit bool, jobs ...T) error {
	if len(jobs) <= 0 {
		return nil
	}
	if fn == nil {
		panic("fn is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var err error
	select {
	case <-ctx.Done():
		err = ctx.Err()
		if err == nil {
			err = context.Canceled
		}
		return err
	default:
	}

	ctx, cancel := context.WithCancel(ctx)
	wg := &sync.WaitGroup{}
	lock := &sync.Mutex{}
	fnWrapper := func(job *T) {
		defer wg.Done()
		select {
		case <-ctx.Done():
			lock.Lock()
			if ctxErr := ctx.Err(); ctxErr != nil && err == nil {
				err = ctxErr
			}
			lock.Unlock()
			return
		default:
		}
		var ferr error
		defer func() {
			if p := recover(); p != nil {
				ferr = fmt.Errorf("panic: %v [recovered]\n%s\n", p, stackTrace())
			}
			if ferr != nil && !errors.Is(ferr, context.Canceled) {
				lock.Lock()
				if err == nil || errors.Is(err, context.Canceled) {
					err = ferr
				}
				lock.Unlock()
				cancel()
			}
		}()
		ferr = fn(ctx, *job)
	}
	for i := range jobs {
		wg.Add(1)
		fnWrapper(&jobs[i])
	}

	go func() {
		wg.Wait()
		cancel()
	}()

	if fastExit {
		select {
		case <-ctx.Done():
			return err
		}
	}

	wg.Wait()
	return err
}

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
func RunPipeline[T any](ctx context.Context, jobs []T, stopWhenErr bool, steps ...func(context.Context, T) error) (
	successCh <-chan T, errCh <-chan error) {

	tch := make(chan T, len(jobs))
	successCh = tch
	if len(steps) <= 0 || len(jobs) <= 0 {
		close(tch)
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithCancel(ctx)
	ch := make(chan T, len(jobs))
	go func() {
		for i := range jobs {
			ch <- jobs[i]
		}
		close(ch)
	}()
	errChans := make([]<-chan error, len(steps))
	var nextCh <-chan T = ch
	for i, step := range steps {
		if step == nil {
			panic("steps can't be nil")
		}
		nextCh, errChans[i] = startStep(ctx, step, cancel, nextCh, stopWhenErr)
	}
	go func() {
		for t := range nextCh {
			tch <- t
		}
		close(tch)
		cancel()
	}()

	return successCh, ListenChan(errChans...)
}

// NewPipelineRunner 形同RunPipeline，不同在于使用push推送job。step返回true表示传递给下一个step处理。
func NewPipelineRunner[T any](ctx context.Context, steps ...func(context.Context, T) bool) (
	push func(T) bool, successCh <-chan T, endPush func()) {

	if len(steps) <= 0 {
		ch := make(chan T)
		close(ch)
		return func(t T) bool { return false }, ch, func() {}
	}

	stepWrapper := func(ctx context.Context, f func(context.Context, T) bool, dataChan <-chan T) <-chan T {
		ch := make(chan T, cap(dataChan))
		go func() {
			wg := &sync.WaitGroup{}
			defer func() {
				wg.Wait()
				close(ch)
			}()
			for {
				select {
				case <-ctx.Done():
					return
				case t, ok := <-dataChan:
					if !ok {
						return
					}
					wg.Add(1)
					go func(t T) {
						defer wg.Done()
						select {
						case <-ctx.Done():
							return
						default:
						}
						var next bool
						defer func() {
							if p := recover(); p != nil {
								next = false
							}
							if next {
								ch <- t
							}
						}()
						next = f(ctx, t)
					}(t)
				}
			}
		}()
		return ch
	}

	if ctx == nil {
		ctx = context.Background()
	}

	jobQueue := &Queue[T]{}
	nextCh := jobQueue.GetFromChan()
	for i := range steps {
		nextCh = stepWrapper(ctx, steps[i], nextCh)
	}

	return jobQueue.Push, nextCh, jobQueue.Close
}

// ListenChan 监听chans，一旦有一个chan激活便立即将T发送给ch，并close ch。
//
// 若所有chans都未曾激活（chan是nil也认为未激活）且都close了，则ch被close。
//
// 若同时多个chans被激活，则随机将一个激活值发送给ch。
func ListenChan[T any](chans ...<-chan T) (ch <-chan T) {
	tch := make(chan T, 1)
	ch = tch
	if len(chans) <= 0 {
		close(tch)
		return
	}
	go func() {
		scs := make([]reflect.SelectCase, 0, len(chans))
		for i := range chans {
			if chans[i] == nil {
				continue
			}
			scs = append(scs, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(chans[i]),
			})
		}
		for {
			if len(scs) <= 0 {
				close(tch)
				return
			}
			chosen, recv, ok := reflect.Select(scs)
			if !ok {
				scs = append(scs[:chosen], scs[chosen+1:]...)
			} else {
				reflect.ValueOf(tch).Send(recv)
				return
			}
		}
	}()

	return
}

// RunPeriodically 依次运行fn，每个fn之间至少间隔period时间。
func RunPeriodically(period time.Duration) (run func(fn func())) {
	fnChan := make(chan func())
	lastAccess := time.Time{}
	errCh := make(chan any, 1)
	fnWrapper := func(f func()) (p any) {
		defer func() { p = recover() }()
		f()
		return nil
	}
	go func() {
		for f := range fnChan {
			time.Sleep(period - time.Since(lastAccess))
			errCh <- fnWrapper(f)
			lastAccess = time.Now()
		}
	}()
	return func(fn func()) {
		fnChan <- fn
		if p := <-errCh; p != nil {
			panic(p)
		}
	}
}

func getStackCallers() string {
	callers := make([]uintptr, 3*12)
	n := runtime.Callers(3, callers)
	callers = callers[:n]
	frames := runtime.CallersFrames(callers)
	callers = nil
	var (
		frame runtime.Frame
		more  bool
	)
	sb := &strings.Builder{}
	for {
		frame, more = frames.Next()
		_, _ = fmt.Fprintf(sb, "%s\n", frame.Function)
		_, _ = fmt.Fprintf(sb, "    %s:%v\n", frame.File, frame.Line)
		if !more {
			break
		}
	}
	return sb.String()
}

func stackTrace() string {
	sb := &strings.Builder{}
	var pc [4096]uintptr
	l := runtime.Callers(2, pc[:])
	frames := runtime.CallersFrames(pc[:l])
	for {
		frame, more := frames.Next()
		_, _ = fmt.Fprintf(sb, "%s\n", frame.Function)
		_, _ = fmt.Fprintf(sb, "    %s:%v\n", frame.File, frame.Line)
		if !more {
			break
		}
	}

	return sb.String()
}

func startStep[T any](ctx context.Context, f func(context.Context, T) error, notify func(), dataChan <-chan T, stopWhenErr bool) (
	<-chan T, <-chan error) {

	ch := make(chan T, cap(dataChan))
	errCh := make(chan error, cap(dataChan))
	go func() {
		ctx, cancel := context.WithCancel(ctx)
		wg := &sync.WaitGroup{}
		defer func() {
			wg.Wait()
			cancel()
			close(ch)
			close(errCh)
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case t, ok := <-dataChan:
				if !ok {
					return
				}
				wg.Add(1)
				go func(t T) {
					defer wg.Done()
					select {
					case <-ctx.Done():
						return
					default:
					}
					var err error
					defer func() {
						if p := recover(); p != nil {
							err = fmt.Errorf("panic: %v [recovered]\n%s\n", p, stackTrace())
						}
						if err != nil {
							if stopWhenErr {
								cancel()
								notify()
							}
							if !errors.Is(err, context.Canceled) {
								errCh <- err
							}
						} else {
							ch <- t
						}
					}()
					err = f(ctx, t)
				}(t)
			}
		}
	}()

	return ch, errCh
}

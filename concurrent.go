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
	"time"
)

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

// NewRunner 该函数提供同时最多运行max个协程fn，一旦fn发生error便终止fn运行。
//
// max小于等于0表示不限制协程数。
//
// 朝返回的run函数中添加fn，若block为true表示正在运行的任务数已达到max则会阻塞。
//
// run函数返回error为任务fn返回的第一个error，与wait函数返回的error为同一个。
//
// 注意请在add完所有任务后调用wait。
func NewRunner[T any](ctx context.Context, max int, fn func(context.Context, T) error) (
	run func(t T, block bool) error, wait func(fastExit bool) error) {

	if fn == nil {
		panic("fn can't be nil")
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
		return func(t T, block bool) error { return err }, func(bool) error { return err }
	default:
	}

	var limiter chan struct{}
	if max > 0 {
		limiter = make(chan struct{}, max)
	}
	ctx, cancel := context.WithCancel(ctx)
	wg := sync.WaitGroup{}
	once := &sync.Once{}
	lock := &sync.Mutex{}
	fnWrapper := func(t T) {
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
			if limiter != nil {
				<-limiter
			}
			wg.Done()
		}()
		ferr = fn(ctx, t)
	}
	run = func(t T, block bool) error {
		wg.Add(1)
		if block && limiter != nil {
			select {
			case <-ctx.Done():
				lock.Lock()
				if ctxErr := ctx.Err(); ctxErr != nil && err == nil {
					err = ctxErr
				}
				lock.Unlock()
				wg.Done()
				return err
			case limiter <- struct{}{}:
			}
			go fnWrapper(t)
			return err
		}
		go func() {
			if limiter != nil {
				select {
				case <-ctx.Done():
					lock.Lock()
					if ctxErr := ctx.Err(); ctxErr != nil && err == nil {
						err = ctxErr
					}
					lock.Unlock()
					wg.Done()
					return
				case limiter <- struct{}{}:
				}
			}
			fnWrapper(t)
		}()

		return err
	}
	wait = func(fastExit bool) error {
		once.Do(func() {
			go func() {
				wg.Wait()
				cancel()
			}()
		})
		if fastExit {
			select {
			case <-ctx.Done():
				return err
			}
		}
		wg.Wait()
		return err
	}

	return
}

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

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
	"sync"
	"sync/atomic"
)

// ErrCallAddAfterWait 在 NewRunner 中，如果调用 wait 后，再调用 add 时，将会返回。
var ErrCallAddAfterWait = errors.New("cannot call add after wait")

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
	add func(t T, block bool) error, wait func(fastExit bool) error) {

	// 处理函数不可为空。
	if fn == nil {
		panic("fn cannot be nil")
	}

	// 上下文终止了，就不必再分配资源，直接返回。
	userCtx := ctx
	if ctx == nil {
		ctx = context.Background()
	} else {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			return func(T, bool) error { return err }, func(bool) error { return err }
		default:
		}
	}

	var err AtomicError                 // 持有错误信息。
	errSetSignal := make(chan struct{}) // 错误是否设置的信号。
	errSetOnce := sync.Once{}           // 设置错误保护的函数。
	setErrFn := func(e error) {         // 设置错误信息。
		errSetOnce.Do(func() {
			err.Set(e)
			close(errSetSignal)
		})
	}
	getErrFn := func() error { // 阻塞获取终止态的错误信息。
		<-errSetSignal
		return err.Get()
	}

	wg := sync.WaitGroup{}    // 计数器。
	var limiter chan struct{} // 限制同时运行 fn 的数量。
	if max > 0 {
		limiter = make(chan struct{}, max)
	}
	innerCtx, cancel := context.WithCancel(ctx) // 通知终止 fn 运行的上下文。
	fnWrapper := func(t T, useLimiter bool) {   // 包装 fn 函数，恢复恐慌。
		var fnErr error
		defer func() {
			if p := recover(); p != nil {
				fnErr = wrapperPanic(p)
			}
			if fnErr != nil {
				setErrFn(fnErr)
			}
			if limiter != nil && useLimiter {
				<-limiter
			}
			wg.Done()
		}()
		select {
		case <-innerCtx.Done():
		default:
			fnErr = fn(userCtx, t)
		}
	}

	waitCallFlag := int32(0) // 持有 wait 调用标志。
	add = func(t T, block bool) error {
		// 不允许在 wait 后再调用 add。
		if atomic.LoadInt32(&waitCallFlag) > 0 {
			panic(ErrCallAddAfterWait)
		}

		select {
		case <-innerCtx.Done(): // 任务已终止，返回函数，不添加。
			return err.Get()
		default:
		}

		wg.Add(1) // 增加计数器。

		// 不阻塞添加任务。
		if !block || limiter == nil {
			go fnWrapper(t, false)
			return err.Get()
		}

		// 阻塞添加任务。
		select {
		case <-innerCtx.Done(): // 任务已终止。
			defer wg.Done() // 减少计数器。
			select {
			case <-errSetSignal: // 等待设置完错误信息。
			case <-ctx.Done(): // 避免上层上下文终止导致死锁。
			}
		case limiter <- struct{}{}: // 获取限数器。
			go fnWrapper(t, true)
		}
		return err.Get()
	}

	wait = func(fastExit bool) error {
		// 设置调用 wait 标志。
		if atomic.CompareAndSwapInt32(&waitCallFlag, 0, 1) {
			go func() {
				// 等待所有协程退出。
				wg.Wait()

				// 有可能是上层上下文终止了。
				select {
				case <-innerCtx.Done(): // innerCtx 只有发生错误或者上层上下文终止，才会终止。
					setErrFn(ctx.Err())
				default:
					// 本就没有发生错误，关闭 getErrFn 的等待。
					setErrFn(nil)
					cancel()
				}
			}()
		}

		// 快速退出，不等待所有协程完毕。
		if !fastExit {
			// 等待所有协程退出。
			wg.Wait()
		}

		return getErrFn()
	}

	return
}
